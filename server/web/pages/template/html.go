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

//go:embed pages/assets/Add-lQnnLANi.js
var AssetsAddlQnnLANijs []byte

//go:embed pages/assets/AddTorrentModal-DPXhRIVP.js
var AssetsAddTorrentModalDPXhRIVPjs []byte

//go:embed pages/assets/Button-DALQN4Zf.js
var AssetsButtonDALQN4Zfjs []byte

//go:embed pages/assets/CacheMap-bJn1r6lG.js
var AssetsCacheMapbJn1r6lGjs []byte

//go:embed pages/assets/Certificates-CUrOl9Yi.js
var AssetsCertificatesCUrOl9Yijs []byte

//go:embed pages/assets/CommandPalette-4Hw5R51x.js
var AssetsCommandPalette4Hw5R51xjs []byte

//go:embed pages/assets/Dashboard-h7P1PWg7.js
var AssetsDashboardh7P1PWg7js []byte

//go:embed pages/assets/FlowDiagnosticsDrawer-BL9jK0WW.js
var AssetsFlowDiagnosticsDrawerBL9jK0WWjs []byte

//go:embed pages/assets/FlowHealthPanel-H2g5fcOD.js
var AssetsFlowHealthPanelH2g5fcODjs []byte

//go:embed pages/assets/GstRuntimeStatus-tnQ6Hun0.js
var AssetsGstRuntimeStatustnQ6Hun0js []byte

//go:embed pages/assets/IntegrationSettings-BHs5Zbh-.js
var AssetsIntegrationSettingsBHs5Zbhjs []byte

//go:embed pages/assets/LANTest-B3gbytzi.js
var AssetsLANTestB3gbytzijs []byte

//go:embed pages/assets/MaintenancePanel-Bm1glnEV.js
var AssetsMaintenancePanelBm1glnEVjs []byte

//go:embed pages/assets/Modal-D_emqY2_.js
var AssetsModalDemqY2js []byte

//go:embed pages/assets/PhonePairingModal-BkpzGFJI.js
var AssetsPhonePairingModalBkpzGFJIjs []byte

//go:embed pages/assets/PlaybackLinks-BU5zzN1n.js
var AssetsPlaybackLinksBU5zzN1njs []byte

//go:embed pages/assets/PosterSearch-DUhxjg5a.js
var AssetsPosterSearchDUhxjg5ajs []byte

//go:embed pages/assets/RecentPlayback-xeWdwi4r.js
var AssetsRecentPlaybackxeWdwi4rjs []byte

//go:embed pages/assets/RecentSessions-BDU4IOuq.js
var AssetsRecentSessionsBDU4IOuqjs []byte

//go:embed pages/assets/SessionCard-BNJz3TzR.js
var AssetsSessionCardBNJz3TzRjs []byte

//go:embed pages/assets/Settings-Bx2APu6F.js
var AssetsSettingsBx2APu6Fjs []byte

//go:embed pages/assets/TorrentFilesDialog-CXYSjO3D.js
var AssetsTorrentFilesDialogCXYSjO3Djs []byte

//go:embed pages/assets/Torrents-DBgyqaSH.js
var AssetsTorrentsDBgyqaSHjs []byte

//go:embed pages/assets/VideoPlayer-BHw4VzhS.js
var AssetsVideoPlayerBHw4VzhSjs []byte

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

//go:embed pages/assets/index-CV1nVqAc.css
var AssetsindexCV1nVqAccss []byte

//go:embed pages/assets/index-Tg9fx5PA.js
var AssetsindexTg9fx5PAjs []byte

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

//go:embed pages/assets/subtitles-DgEUQwDV.js
var AssetssubtitlesDgEUQwDVjs []byte

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
