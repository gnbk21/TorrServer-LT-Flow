package template

import (
	_ "embed"
)

//go:embed pages/DEPENDENCIES.json
var DEPENDENCIESjson []byte

//go:embed pages/THIRD_PARTY_NOTICES.txt
var THIRDPARTYNOTICEStxt []byte

//go:embed pages/appearance.js
var Appearancejs []byte

//go:embed pages/assets/Add-BFKSt8uQ.js
var AssetsAddBFKSt8uQjs []byte

//go:embed pages/assets/AddTorrentModal-Bz_fp_ht.js
var AssetsAddTorrentModalBzfphtjs []byte

//go:embed pages/assets/Button-DALQN4Zf.js
var AssetsButtonDALQN4Zfjs []byte

//go:embed pages/assets/CacheMap-BmriVvaC.js
var AssetsCacheMapBmriVvaCjs []byte

//go:embed pages/assets/Certificates-TmPHUd6I.js
var AssetsCertificatesTmPHUd6Ijs []byte

//go:embed pages/assets/CommandPalette-Pv6RVNjr.js
var AssetsCommandPalettePv6RVNjrjs []byte

//go:embed pages/assets/Dashboard-D2MpC7fL.js
var AssetsDashboardD2MpC7fLjs []byte

//go:embed pages/assets/FlowDiagnosticsDrawer-CtJK2oZJ.js
var AssetsFlowDiagnosticsDrawerCtJK2oZJjs []byte

//go:embed pages/assets/FlowHealthPanel-DIuyN4-_.js
var AssetsFlowHealthPanelDIuyN4js []byte

//go:embed pages/assets/GstRuntimeStatus-CEd4NSdQ.js
var AssetsGstRuntimeStatusCEd4NSdQjs []byte

//go:embed pages/assets/IntegrationSettings-BgcIQzWM.js
var AssetsIntegrationSettingsBgcIQzWMjs []byte

//go:embed pages/assets/LANTest-Dd5OYl0j.js
var AssetsLANTestDd5OYl0jjs []byte

//go:embed pages/assets/MaintenancePanel-BqsoIhyr.js
var AssetsMaintenancePanelBqsoIhyrjs []byte

//go:embed pages/assets/Modal-Bc2xDGz0.js
var AssetsModalBc2xDGz0js []byte

//go:embed pages/assets/PhonePairingModal-Dp0QqqTn.js
var AssetsPhonePairingModalDp0QqqTnjs []byte

//go:embed pages/assets/PlaybackLinks-BQj3EYNL.js
var AssetsPlaybackLinksBQj3EYNLjs []byte

//go:embed pages/assets/PosterSearch-DUhxjg5a.js
var AssetsPosterSearchDUhxjg5ajs []byte

//go:embed pages/assets/RecentPlayback-BrJRo0Im.js
var AssetsRecentPlaybackBrJRo0Imjs []byte

//go:embed pages/assets/RecentSessions-CYjDt-iP.js
var AssetsRecentSessionsCYjDtiPjs []byte

//go:embed pages/assets/SessionCard-BZ9pDb9s.js
var AssetsSessionCardBZ9pDb9sjs []byte

//go:embed pages/assets/Settings-BbsiZ84s.js
var AssetsSettingsBbsiZ84sjs []byte

//go:embed pages/assets/TorrentFilesDialog-Cbg2SZEb.js
var AssetsTorrentFilesDialogCbg2SZEbjs []byte

//go:embed pages/assets/Torrents-CODkj74d.js
var AssetsTorrentsCODkj74djs []byte

//go:embed pages/assets/VideoPlayer-CeE699qn.js
var AssetsVideoPlayerCeE699qnjs []byte

//go:embed pages/assets/certificates-Cxtq0Hep.js
var AssetscertificatesCxtq0Hepjs []byte

//go:embed pages/assets/client-5nC5zTpw.js
var Assetsclient5nC5zTpwjs []byte

//go:embed pages/assets/createLucideIcon-I2O3k_i9.js
var AssetscreateLucideIconI2O3ki9js []byte

//go:embed pages/assets/format-B_YcZLV7.js
var AssetsformatBYcZLV7js []byte

//go:embed pages/assets/hls-D2SbReJ5.js
var AssetshlsD2SbReJ5js []byte

//go:embed pages/assets/index-DODHvqlO.js
var AssetsindexDODHvqlOjs []byte

//go:embed pages/assets/index-Z_6qNPBp.css
var AssetsindexZ6qNPBpcss []byte

//go:embed pages/assets/integrations-r9sBPdpF.js
var Assetsintegrationsr9sBPdpFjs []byte

//go:embed pages/assets/mediaBase-w48RrmPo.js
var AssetsmediaBasew48RrmPojs []byte

//go:embed pages/assets/queries-DJoJl9YN.js
var AssetsqueriesDJoJl9YNjs []byte

//go:embed pages/assets/redact-ie8nnS8E.js
var Assetsredactie8nnS8Ejs []byte

//go:embed pages/assets/rolldown-runtime-XKyaHtl9.js
var AssetsrolldownruntimeXKyaHtl9js []byte

//go:embed pages/assets/schemas-Ct8qkdL4.js
var AssetsschemasCt8qkdL4js []byte

//go:embed pages/assets/subtitles-C40jMbP2.js
var AssetssubtitlesC40jMbP2js []byte

//go:embed pages/assets/translation-B1O-l98W.js
var AssetstranslationB1Ol98Wjs []byte

//go:embed pages/assets/translation-B5XE6Py9.js
var AssetstranslationB5XE6Py9js []byte

//go:embed pages/assets/translation-BatjzuR1.js
var AssetstranslationBatjzuR1js []byte

//go:embed pages/assets/translation-C8wPCVRN.js
var AssetstranslationC8wPCVRNjs []byte

//go:embed pages/assets/translation-CWKXa9_a.js
var AssetstranslationCWKXa9ajs []byte

//go:embed pages/assets/translation-WMpOryXo.js
var AssetstranslationWMpOryXojs []byte

//go:embed pages/assets/useQuery-CwjCYSbm.js
var AssetsuseQueryCwjCYSbmjs []byte

//go:embed pages/assets/viewed-aux-rI_Q.js
var AssetsviewedauxrIQjs []byte

//go:embed pages/browserconfig.xml
var Browserconfigxml []byte

//go:embed pages/dlnaicon-120.png
var Dlnaicon120png []byte

//go:embed pages/dlnaicon-48.png
var Dlnaicon48png []byte

//go:embed pages/favicon-16x16.png
var Favicon16x16png []byte

//go:embed pages/favicon-32x32.png
var Favicon32x32png []byte

//go:embed pages/favicon.ico
var Faviconico []byte

//go:embed pages/icon.png
var Iconpng []byte

//go:embed pages/index.html
var Indexhtml []byte

//go:embed pages/logo.png
var Logopng []byte

//go:embed pages/mstile-150x150.png
var Mstile150x150png []byte

//go:embed pages/site.webmanifest
var Sitewebmanifest []byte
