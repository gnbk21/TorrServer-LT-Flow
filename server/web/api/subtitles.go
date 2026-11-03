package api

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"path"
	"server/flow"
	config "server/settings"
	"server/torr"
	"server/torr/storage/torrstor"
	"strconv"
	"strings"
	"time"
)

// Admission spans read, text conversion and response ownership so fast resident
// reads cannot release the limit while many converted buffers are still held.
var subtitleResponses = make(chan struct{}, 2)

func subtitles(c *gin.Context) {
	video, err := strconv.Atoi(c.Param("id"))
	if err != nil || video < 1 {
		abortWithErrorCode(c, 400, "INVALID_SUBTITLE_SELECTION", flow.ErrSubtitle, 1)
		return
	}
	torrent := torr.GetTorrentInfo(c.Param("hash"))
	if torrent == nil {
		abortWithErrorCode(c, 404, "NOT_FOUND", errors.New("not found"), 1)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("Referrer-Policy", "no-referrer")
	if raw, selected := c.GetQuery("track"); selected {
		// Raw subtitle fetches do not acquire media authority merely because
		// HTTP authentication is disabled. Signed aliases have their own claim.
		if config.CurrentFlow().RequirePlaybackToken && c.GetString(gin.AuthUserKey) == "" {
			abortWithErrorCode(c, 401, "SUBTITLE_AUTH_REQUIRED", errors.New("authentication or a subtitle-scoped playback capability is required"), 1)
			return
		}
		track, err := strconv.Atoi(raw)
		if err != nil || track < 1 {
			abortWithErrorCode(c, 400, "INVALID_SUBTITLE_SELECTION", flow.ErrSubtitle, 1)
			return
		}
		serveSelectedSubtitle(c, torrent, video, track, c.DefaultQuery("format", "vtt"))
		return
	}
	rows, truncated := torrent.SubtitleCandidates(video)
	c.JSON(200, gin.H{"schema_version": 1, "source": "torrent_metadata", "known": torrent.SubtitleVideoPath(video) != "", "tracks": rows, "truncated": truncated})
}
func serveSelectedSubtitle(c *gin.Context, torrent *torr.Torrent, video, track int, format string) {
	if format != "original" && format != "vtt" {
		abortWithErrorCode(c, 400, "INVALID_SUBTITLE_FORMAT", flow.ErrSubtitle, 1)
		return
	}
	candidate, found := torrent.SelectedSubtitle(video, track)
	if !found {
		abortWithErrorCode(c, 404, "NOT_FOUND", errors.New("not found"), 1)
		return
	}
	if candidate.Size > flow.SubtitleInputLimit {
		abortWithErrorCode(c, 413, "SUBTITLE_TOO_LARGE", flow.ErrSubtitle, 1)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("X-Content-Type-Options", "nosniff")
	mime := "text/vtt; charset=utf-8"
	if format == "original" {
		switch candidate.Format {
		case "srt":
			mime = "application/x-subrip"
		case "ass", "ssa":
			mime = "text/x-ssa"
		}
	}
	if c.Request.Method == http.MethodHead {
		c.Header("Content-Type", mime)
		if format == "original" {
			c.Header("Content-Length", strconv.FormatInt(candidate.Size, 10))
		}
		c.Status(200)
		return
	}
	select {
	case subtitleResponses <- struct{}{}:
		defer func() { <-subtitleResponses }()
	default:
		c.Header("Retry-After", "1")
		abortWithErrorCode(c, 503, "SUBTITLE_DEFERRED", torrstor.ErrAuxiliaryDeferred, 1)
		return
	}
	deadline := time.Now().Add(10 * time.Second)
	ctx, cancel := context.WithDeadline(c.Request.Context(), deadline)
	defer cancel()
	controller := http.NewResponseController(c.Writer)
	if controller.SetWriteDeadline(deadline) == nil {
		// Do not leave a subtitle deadline on a reused video connection.
		defer controller.SetWriteDeadline(time.Time{})
	}
	data, _, err := torrent.SubtitleBytes(ctx, video, track)
	if err == nil && format == "vtt" {
		data, err = flow.SubtitleWebVTTContext(ctx, data, "."+candidate.Format)
	} else if err == nil {
		err = flow.ValidateSubtitleContext(ctx, data, "."+candidate.Format)
	}
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		if errors.Is(err, torrstor.ErrAuxiliaryDeferred) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) && c.Request.Context().Err() == nil {
			c.Header("Retry-After", "1")
			abortWithErrorCode(c, 503, "SUBTITLE_DEFERRED", err, 1)
		} else {
			abortWithErrorCode(c, 422, "SUBTITLE_UNAVAILABLE", flow.ErrSubtitle, 1)
		}
		return
	}
	ext := candidate.Format
	if format == "vtt" {
		ext = "vtt"
	}
	c.Header("Content-Disposition", `inline; filename="subtitle.`+ext+`"`)
	c.Data(200, mime, data)
}
func capabilityResource(c *gin.Context, torrent *torr.Torrent, claim flow.PlaybackClaim) bool {
	resource := c.Param("resource")
	if resource == "" {
		return false
	}
	ext := strings.ToLower(path.Ext(torrent.SubtitleVideoPath(claim.Index)))
	if resource == "/media"+ext {
		return false
	}
	if claim.SubtitleIndex > 0 {
		if selected, ok := torrent.SelectedSubtitle(claim.Index, claim.SubtitleIndex); ok {
			expected := "/media." + selected.Format
			localized := "/media." + selected.Language + "." + selected.Format
			if resource == expected || selected.Language != "" && resource == localized {
				serveSelectedSubtitle(c, torrent, claim.Index, claim.SubtitleIndex, "original")
				return true
			}
		}
	}
	abortWithErrorCode(c, 404, "NOT_FOUND", errors.New("not found"), 1)
	return true
}
