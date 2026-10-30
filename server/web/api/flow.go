package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"server/flow"
	"server/lt"
	sets "server/settings"
	"server/torr"
	"time"
)

type FlowStatusResponse struct {
	Hash              string                       `json:"hash"`
	Integrity         *flow.IntegritySnapshot      `json:"integrity,omitempty"`
	Startup           torr.FlowStartupStatus       `json:"startup"`
	Sessions          []torr.FlowSessionStatus     `json:"sessions"`
	Trackers          []torr.FlowTrackerDiagnostic `json:"trackers"`
	Network           torr.FlowNetworkStatus       `json:"network"`
	Sparse            lt.SparseSnapshot            `json:"sparse"`
	StorageIO         lt.StorageIO                 `json:"storage_io"`
	Timeline          *flow.TimelineSnapshot       `json:"timeline,omitempty"`
	SampledAt         string                       `json:"sampled_at,omitempty"`
	NextEpisodeWarmup *torr.NextEpisodeStatus      `json:"next_episode_warmup,omitempty"`
}

// flowStatus exposes a bounded diagnostic snapshot for one live torrent.
// It shares the existing API authentication middleware.
func flowStatus(c *gin.Context) {
	// Observing diagnostics must not promote a dropped torrent or extend its
	// lifetime. Playback and explicit preload remain responsible for activation.
	t := torr.GetTorrentInfo(c.Param("hash"))
	if t == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "torrent not found"})
		return
	}
	timeline := t.FlowTimeline()
	warmup := t.NextEpisodeWarmupStatus()
	integrity := t.FlowIntegrity()
	c.JSON(http.StatusOK, FlowStatusResponse{Hash: c.Param("hash"), Integrity: &integrity, Startup: t.FlowStartup(), Sessions: t.FlowStatusWithTraces(c.Query("traces") != "false"), Trackers: t.FlowTrackers(), Network: torr.NetworkStatusSnapshot(), Sparse: t.SparseStatus(), StorageIO: lt.StorageIOStats(), Timeline: &timeline, SampledAt: time.Now().UTC().Format(time.RFC3339Nano), NextEpisodeWarmup: &warmup})
}

func nextEpisodeWarmup(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	var request struct {
		FileIndex int    `json:"file_index"`
		Action    string `json:"action"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid warmup request"})
		return
	}
	t := torr.GetTorrentInfo(c.Param("hash"))
	if t == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "torrent not found"})
		return
	}
	if err := t.SelectNextEpisodeWarmup(request.FileIndex, request.Action); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": errorMessage(err)})
		return
	}
	c.JSON(http.StatusAccepted, t.NextEpisodeWarmupStatus())
}

func flowNetwork(c *gin.Context) {
	c.JSON(http.StatusOK, torr.NetworkStatusSnapshot())
}

func retryMetadata(c *gin.Context) {
	t := torr.GetTorrentInfo(c.Param("hash"))
	if t == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "torrent not found"})
		return
	}
	if err := t.RetryMetadataRecovery(); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": errorMessage(err)})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"accepted": true})
}

func flowTray(c *gin.Context) {
	c.JSON(http.StatusOK, torr.SnapshotFlowTray())
}

func flowControl(c *gin.Context) {
	var request struct {
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage(err)})
		return
	}
	var paused bool
	switch request.Action {
	case "pause":
		paused = true
	case "resume":
		paused = false
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown Flow action"})
		return
	}
	if err := torr.SetFlowPaused(paused); err != nil {
		abortWithJSONError(c, http.StatusServiceUnavailable, err)
		return
	}
	c.JSON(http.StatusOK, torr.SnapshotFlowTray())
}

func flowSessions(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, flow.ReadSessionSummaries(sets.Path, sets.CurrentFlow().DiagnosticHistory))
}
