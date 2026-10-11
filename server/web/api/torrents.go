package api

import (
	"net/http"
	"server/torrshash"
	"strings"

	"server/dlna"
	gstreamer "server/gstreamer/bridge"
	"server/log"
	set "server/settings"
	"server/torr"
	"server/torr/state"
	"server/web/api/utils"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// Action: add, get, set, rem, list, drop
type torrReqJS struct {
	requestI
	Link     string `json:"link,omitempty"`
	Hash     string `json:"hash,omitempty"`
	Title    string `json:"title,omitempty"`
	Category string `json:"category,omitempty"`
	Poster   string `json:"poster,omitempty"`
	Data     string `json:"data,omitempty"`
	SaveToDB bool   `json:"save_to_db,omitempty"`
}

// torrents godoc
//
//	@Summary		Handle torrents informations
//	@Description	Allow to list, add, remove, get, set, drop, wipe torrents on server. The action depends of what has been asked.
//
//	@Tags			API
//
//	@Param			request	body	torrReqJS	true	"Torrent request. Available params for action: add, get, set, rem, list, drop, wipe. link required for add, hash required for get, set, rem, drop."
//
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/torrents [post]
func torrents(c *gin.Context) {
	var req torrReqJS
	err := c.ShouldBindJSON(&req)
	if err != nil {
		abortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	switch req.Action {
	case "add":
		{
			addTorrent(req, c)
		}
	case "get":
		{
			getTorrent(req, c)
		}
	case "set":
		{
			setTorrent(req, c)
		}
	case "rem":
		{
			remTorrent(req, c)
		}
	case "list":
		{
			listTorrents(c)
		}
	case "drop":
		{
			dropTorrent(req, c)
		}
	case "wipe":
		{
			wipeTorrents(c)
		}
	default:
		{
			abortWithJSONError(c, http.StatusBadRequest, errors.Errorf("unknown action: %q", req.Action))
		}
	}
}

func addTorrent(req torrReqJS, c *gin.Context) {
	if req.Link == "" {
		abortWithJSONError(c, http.StatusBadRequest, errors.New("link is empty"))
		return
	}

	// Magnet tracker URLs and remote torrent links may contain private passkeys.
	log.TLogln("add torrent request")
	req.Link = strings.ReplaceAll(req.Link, "&amp;", "&")

	var torrSpec *torr.TorrentSpec
	var torrsHash *torrshash.TorrsHash
	var err error

	if strings.HasPrefix(req.Link, "torrs://") {
		torrSpec, torrsHash, err = utils.ParseTorrsHash(req.Link)
		if err != nil {
			log.TLogln("error parse torrshash:", err)
			abortWithJSONError(c, http.StatusBadRequest, err)
			return
		}
		if req.Title == "" {
			req.Title = torrsHash.Title()
		}
		if req.Poster == "" {
			req.Poster = torrsHash.Poster()
		}
		if req.Category == "" {
			req.Category = torrsHash.Category()
		}
	} else {
		torrSpec, err = utils.ParseLinkContext(c.Request.Context(), req.Link)
		if err != nil {
			log.TLogln("error parse link:", err)
			abortWithJSONError(c, http.StatusBadRequest, err)
			return
		}
	}

	tor, err := torr.AddTorrent(torrSpec, req.Title, req.Poster, req.Data, req.Category)
	if err != nil {
		log.TLogln("error add torrent:", err)
		abortWithJSONError(c, http.StatusInternalServerError, err)
		return
	}

	go func() {
		if !tor.GotInfo() {
			log.TLogln("error add torrent:", "timeout connection get torrent info")
			return
		}

		if tor.Title == "" {
			tor.Title = torrSpec.DisplayName // prefer dn over name
			tor.Title = strings.ReplaceAll(tor.Title, "rutor.info", "")
			tor.Title = strings.ReplaceAll(tor.Title, "_", " ")
			tor.Title = strings.Trim(tor.Title, " ")
			if tor.Title == "" {
				tor.Title = tor.Name()
			}
		}

		if req.SaveToDB {
			torr.SaveTorrentToDB(tor)
		}
	}()

	if set.BTsets().EnableDLNA {
		dlna.Stop()
		dlna.Start()
	}
	c.JSON(200, tor.Status())
}

func getTorrent(req torrReqJS, c *gin.Context) {
	if req.Hash == "" {
		abortWithJSONError(c, http.StatusBadRequest, errors.New("hash is empty"))
		return
	}
	tor := torr.GetTorrent(req.Hash)

	if tor != nil {
		st := tor.Status()
		c.JSON(200, st)
	} else {
		abortWithJSONError(c, http.StatusNotFound, errors.New("torrent not found"))
	}
}

func setTorrent(req torrReqJS, c *gin.Context) {
	if req.Hash == "" {
		abortWithJSONError(c, http.StatusBadRequest, errors.New("hash is empty"))
		return
	}
	torr.SetTorrent(req.Hash, req.Title, req.Poster, req.Category, req.Data)
	c.Status(200)
}

func remTorrent(req torrReqJS, c *gin.Context) {
	if req.Hash == "" {
		abortWithJSONError(c, http.StatusBadRequest, errors.New("hash is empty"))
		return
	}
	if err := torr.RemTorrent(req.Hash); err != nil {
		abortWithJSONError(c, http.StatusConflict, err)
		return
	}
	gstreamer.Remove(req.Hash)
	// TODO: remove
	if set.BTsets().EnableDLNA {
		dlna.Stop()
		dlna.Start()
	}
	c.Status(200)
}

func listTorrents(c *gin.Context) {
	list := torr.ListTorrent()
	if len(list) == 0 {
		c.JSON(200, []*state.TorrentStatus{})
		return
	}
	var stats []*state.TorrentStatus
	for _, tr := range list {
		st := tr.Status()
		if st.ActiveReaders > 0 {
			for _, session := range tr.FlowStatus() {
				if session.ActiveReaders <= 0 {
					continue
				}
				summary := state.FlowPlaybackSummary{FileIndex: session.FileIndex, BufferWarning: session.BufferWarning}
				if session.PlaybackConsumptionRate > 0 {
					buffer, ratio := session.BufferAheadSeconds, session.SustainabilityRatio
					summary.BufferSeconds, summary.Sustainability = &buffer, &ratio
				}
				st.FlowPlayback = append(st.FlowPlayback, summary)
			}
		}
		stats = append(stats, st)
	}
	c.JSON(200, stats)
}

func dropTorrent(req torrReqJS, c *gin.Context) {
	if req.Hash == "" {
		abortWithJSONError(c, http.StatusBadRequest, errors.New("hash is empty"))
		return
	}
	torr.DropTorrent(req.Hash)
	gstreamer.Remove(req.Hash)
	c.Status(200)
}

func wipeTorrents(c *gin.Context) {
	torrents := torr.ListTorrent()
	for _, t := range torrents {
		if err := torr.RemTorrent(t.TorrentSpec.InfoHash.HexString()); err != nil {
			abortWithJSONError(c, http.StatusConflict, err)
			return
		}
	}
	// TODO: remove (copied todo from remTorrent())
	if set.BTsets().EnableDLNA {
		dlna.Stop()
		dlna.Start()
	}
	c.Status(200)
}
