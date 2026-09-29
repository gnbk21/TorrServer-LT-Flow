package template

import (
	"crypto/md5"
	"fmt"
	"github.com/gin-gonic/gin"
)

func RouteWebPages(route gin.IRouter) {
	route.GET("/", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(Indexhtml))
		c.Header("Cache-Control", "no-cache")
		c.Header("ETag", etag)
		c.Data(200, "text/html; charset=utf-8", Indexhtml)
	})

	route.GET("/THIRD_PARTY_NOTICES.txt", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(THIRDPARTYNOTICEStxt))
		c.Header("Cache-Control", "public, max-age=3600")
		c.Header("ETag", etag)
		c.Data(200, "text/plain; charset=utf-8", THIRDPARTYNOTICEStxt)
	})

	route.GET("/assets/Add-pXLq2E_p.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsAddpXLq2Epjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsAddpXLq2Epjs)
	})

	route.GET("/assets/AddTorrentModal-AAEitUlO.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsAddTorrentModalAAEitUlOjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsAddTorrentModalAAEitUlOjs)
	})

	route.GET("/assets/Button-KTs46vB5.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsButtonKTs46vB5js))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsButtonKTs46vB5js)
	})

	route.GET("/assets/CacheMap-A0Vl1Dg9.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsCacheMapA0Vl1Dg9js))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsCacheMapA0Vl1Dg9js)
	})

	route.GET("/assets/Dashboard-BhLk2Rv2.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsDashboardBhLk2Rv2js))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsDashboardBhLk2Rv2js)
	})

	route.GET("/assets/FlowDiagnosticsDrawer-CiqEzHhv.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsFlowDiagnosticsDrawerCiqEzHhvjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsFlowDiagnosticsDrawerCiqEzHhvjs)
	})

	route.GET("/assets/GstRuntimeStatus-Cwkju5GU.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsGstRuntimeStatusCwkju5GUjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsGstRuntimeStatusCwkju5GUjs)
	})

	route.GET("/assets/PhonePairingModal-yuSAmrTx.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsPhonePairingModalyuSAmrTxjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsPhonePairingModalyuSAmrTxjs)
	})

	route.GET("/assets/PlaybackLinks-CyNpNLoB.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsPlaybackLinksCyNpNLoBjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsPlaybackLinksCyNpNLoBjs)
	})

	route.GET("/assets/PosterSearch-DyWxdSkx.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsPosterSearchDyWxdSkxjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsPosterSearchDyWxdSkxjs)
	})

	route.GET("/assets/Settings-D0gy1Xr-.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsSettingsD0gy1Xrjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsSettingsD0gy1Xrjs)
	})

	route.GET("/assets/TorrentFilesDialog-W4Pz-37Q.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsTorrentFilesDialogW4Pz37Qjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsTorrentFilesDialogW4Pz37Qjs)
	})

	route.GET("/assets/Torrents-BZ8l6j8N.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsTorrentsBZ8l6j8Njs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsTorrentsBZ8l6j8Njs)
	})

	route.GET("/assets/VideoPlayer-C7JJqRko.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsVideoPlayerC7JJqRkojs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsVideoPlayerC7JJqRkojs)
	})

	route.GET("/assets/client-Dr26AKOf.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsclientDr26AKOfjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsclientDr26AKOfjs)
	})

	route.GET("/assets/clipboard-BX4bd3OW.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsclipboardBX4bd3OWjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsclipboardBX4bd3OWjs)
	})

	route.GET("/assets/createLucideIcon-D7zlQbrg.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetscreateLucideIconD7zlQbrgjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetscreateLucideIconD7zlQbrgjs)
	})

	route.GET("/assets/hls-D9b4QHpD.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetshlsD9b4QHpDjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetshlsD9b4QHpDjs)
	})

	route.GET("/assets/index-DvLSKgo6.css", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsindexDvLSKgo6css))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/css; charset=utf-8", AssetsindexDvLSKgo6css)
	})

	route.GET("/assets/index-zDv5cfCc.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsindexzDv5cfCcjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsindexzDv5cfCcjs)
	})

	route.GET("/assets/integrations-WHdJYyGB.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsintegrationsWHdJYyGBjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsintegrationsWHdJYyGBjs)
	})

	route.GET("/assets/preload-helper-uBIymjUX.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetspreloadhelperuBIymjUXjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetspreloadhelperuBIymjUXjs)
	})

	route.GET("/assets/redact-C8NqrjZt.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsredactC8NqrjZtjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsredactC8NqrjZtjs)
	})

	route.GET("/assets/rolldown-runtime-CbXtAM7H.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsrolldownruntimeCbXtAM7Hjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsrolldownruntimeCbXtAM7Hjs)
	})

	route.GET("/assets/translation-B0NYojKI.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetstranslationB0NYojKIjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetstranslationB0NYojKIjs)
	})

	route.GET("/assets/translation-BIGisKhu.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetstranslationBIGisKhujs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetstranslationBIGisKhujs)
	})

	route.GET("/assets/translation-BWAK4bsq.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetstranslationBWAK4bsqjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetstranslationBWAK4bsqjs)
	})

	route.GET("/assets/translation-Bs9KBEWW.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetstranslationBs9KBEWWjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetstranslationBs9KBEWWjs)
	})

	route.GET("/assets/translation-CjVsGf_8.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetstranslationCjVsGf8js))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetstranslationCjVsGf8js)
	})

	route.GET("/assets/translation-ZAW2oU3J.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetstranslationZAW2oU3Jjs))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetstranslationZAW2oU3Jjs)
	})

	route.GET("/assets/useQuery-xJfTCyZ4.js", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(AssetsuseQueryxJfTCyZ4js))
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", etag)
		c.Data(200, "text/javascript; charset=utf-8", AssetsuseQueryxJfTCyZ4js)
	})

	route.GET("/browserconfig.xml", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(Browserconfigxml))
		c.Header("Cache-Control", "public, max-age=3600")
		c.Header("ETag", etag)
		c.Data(200, "text/xml; charset=utf-8", Browserconfigxml)
	})

	route.GET("/dlnaicon-120.png", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(Dlnaicon120png))
		c.Header("Cache-Control", "public, max-age=3600")
		c.Header("ETag", etag)
		c.Data(200, "image/png", Dlnaicon120png)
	})

	route.GET("/dlnaicon-48.png", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(Dlnaicon48png))
		c.Header("Cache-Control", "public, max-age=3600")
		c.Header("ETag", etag)
		c.Data(200, "image/png", Dlnaicon48png)
	})

	route.GET("/favicon-16x16.png", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(Favicon16x16png))
		c.Header("Cache-Control", "public, max-age=3600")
		c.Header("ETag", etag)
		c.Data(200, "image/png", Favicon16x16png)
	})

	route.GET("/favicon-32x32.png", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(Favicon32x32png))
		c.Header("Cache-Control", "public, max-age=3600")
		c.Header("ETag", etag)
		c.Data(200, "image/png", Favicon32x32png)
	})

	route.GET("/favicon.ico", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(Faviconico))
		c.Header("Cache-Control", "public, max-age=3600")
		c.Header("ETag", etag)
		c.Data(200, "image/vnd.microsoft.icon", Faviconico)
	})

	route.GET("/icon.png", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(Iconpng))
		c.Header("Cache-Control", "public, max-age=3600")
		c.Header("ETag", etag)
		c.Data(200, "image/png", Iconpng)
	})

	route.GET("/index.html", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(Indexhtml))
		c.Header("Cache-Control", "no-cache")
		c.Header("ETag", etag)
		c.Data(200, "text/html; charset=utf-8", Indexhtml)
	})

	route.GET("/logo.png", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(Logopng))
		c.Header("Cache-Control", "public, max-age=3600")
		c.Header("ETag", etag)
		c.Data(200, "image/png", Logopng)
	})

	route.GET("/mstile-150x150.png", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(Mstile150x150png))
		c.Header("Cache-Control", "public, max-age=3600")
		c.Header("ETag", etag)
		c.Data(200, "image/png", Mstile150x150png)
	})

	route.GET("/site.webmanifest", func(c *gin.Context) {
		etag := fmt.Sprintf("%x", md5.Sum(Sitewebmanifest))
		c.Header("Cache-Control", "no-cache")
		c.Header("ETag", etag)
		c.Data(200, "application/manifest+json", Sitewebmanifest)
	})
}
