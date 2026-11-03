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

//go:embed pages/assets/Add-CzL5UgHM.js
var AssetsAddCzL5UgHMjs []byte

//go:embed pages/assets/AddTorrentModal-DR1Uh85P.js
var AssetsAddTorrentModalDR1Uh85Pjs []byte

//go:embed pages/assets/Button-DALQN4Zf.js
var AssetsButtonDALQN4Zfjs []byte

//go:embed pages/assets/CacheMap-BK9uoLZs.js
var AssetsCacheMapBK9uoLZsjs []byte

//go:embed pages/assets/Certificates-D7c2lDnj.js
var AssetsCertificatesD7c2lDnjjs []byte

//go:embed pages/assets/CommandPalette-DERIHxig.js
var AssetsCommandPaletteDERIHxigjs []byte

//go:embed pages/assets/Dashboard-BxntTJ-n.js
var AssetsDashboardBxntTJnjs []byte

//go:embed pages/assets/FlowDiagnosticsDrawer-CjKJVLms.js
var AssetsFlowDiagnosticsDrawerCjKJVLmsjs []byte

//go:embed pages/assets/FlowHealthPanel-TL4OF2if.js
var AssetsFlowHealthPanelTL4OF2ifjs []byte

//go:embed pages/assets/GstRuntimeStatus-DSggCYu1.js
var AssetsGstRuntimeStatusDSggCYu1js []byte

//go:embed pages/assets/IntegrationSettings-BTkr_i4n.js
var AssetsIntegrationSettingsBTkri4njs []byte

//go:embed pages/assets/LANTest-D76ILyNx.js
var AssetsLANTestD76ILyNxjs []byte

//go:embed pages/assets/MaintenancePanel-CIKUpye0.js
var AssetsMaintenancePanelCIKUpye0js []byte

//go:embed pages/assets/Modal-CKphKB7C.js
var AssetsModalCKphKB7Cjs []byte

//go:embed pages/assets/PhonePairingModal-Bqnob3Xk.js
var AssetsPhonePairingModalBqnob3Xkjs []byte

//go:embed pages/assets/PlaybackLinks-BQj3EYNL.js
var AssetsPlaybackLinksBQj3EYNLjs []byte

//go:embed pages/assets/PosterSearch-DUhxjg5a.js
var AssetsPosterSearchDUhxjg5ajs []byte

//go:embed pages/assets/RecentPlayback-DUpwtTdH.js
var AssetsRecentPlaybackDUpwtTdHjs []byte

//go:embed pages/assets/RecentSessions-B_yq5RAI.js
var AssetsRecentSessionsByq5RAIjs []byte

//go:embed pages/assets/SessionCard-goBInxdL.js
var AssetsSessionCardgoBInxdLjs []byte

//go:embed pages/assets/Settings-JDUw3lrR.js
var AssetsSettingsJDUw3lrRjs []byte

//go:embed pages/assets/TorrentFilesDialog-B_L1YEH6.js
var AssetsTorrentFilesDialogBL1YEH6js []byte

//go:embed pages/assets/Torrents-D1rCahig.js
var AssetsTorrentsD1rCahigjs []byte

//go:embed pages/assets/VideoPlayer-Bjrb45Cl.js
var AssetsVideoPlayerBjrb45Cljs []byte

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

//go:embed pages/assets/index-BVhMT2DD.js
var AssetsindexBVhMT2DDjs []byte

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
