# RuTracker search in Lampa

Lampa searches through its configured parser; Flow receives the selected torrent
for streaming. Adding RuTracker's announce URL to Flow does not index its forum.
The parser and TorrServer addresses are separate settings.

## Free local setup: Jackett

1. Install [Jackett for Windows](https://github.com/Jackett/Jackett#windows-installation)
   on the same PC as Flow. Open `http://127.0.0.1:9117` on that PC.
2. Choose **Add indexer**, select **RuTracker.org** (not RuTracker.RU), and enter
   your own free RuTracker account credentials in Jackett. Use its **Test** and
   save controls. Search successfully in Jackett before configuring the phone.
3. Enable Jackett's **External access** for your phone on the private LAN. Allow
   its private-network firewall rule if necessary; no router port forwarding is
   needed. Keep the API key private and use dashboard protection when available.
4. On the phone, open **Lampa → Settings → Parser**:
   - enable the parser;
   - type: **Jackett**;
   - address/link: `<PC-LAN-IP>:9117`, for example `192.168.1.50:9117`;
   - API key: copy the key shown by your Jackett dashboard.
5. Keep **Lampa → Settings → TorrServer** pointing to the existing Flow address,
   `<PC-LAN-IP>:8090`. Keep Just Player selected for playback.
6. Open a movie or series and choose **Torrents**. If tracker filters are active,
   include RuTracker. Both Jackett and Flow must be running on the PC.

`127.0.0.1` on the phone means the phone itself. The phone needs the PC's LAN
address, while the PC can use loopback to open its own Jackett dashboard.
Lampa takes the Jackett base address above; do not paste an individual Torznab
feed URL into that field.

Sources: [Lampa's Jackett configuration](https://github.com/yumata/lampa/wiki/Настройка-парсера-Jackett)
and [Jackett's supported semi-private trackers](https://github.com/Jackett/Jackett#supported-semi-private-trackers).

## If it does not return results

- **Jackett's indexer test fails:** resolve the RuTracker login, site access or
  challenge first. Check that you can log in from the same PC/network and that
  the site has no outstanding notice. Consult the
  [Jackett troubleshooting guide](https://github.com/Jackett/Jackett/wiki/Troubleshooting).
  A parser challenge/login failure is separate from Flow's metadata recovery.
- **Jackett works, phone cannot connect:** check the PC's current LAN address,
  external access, private-network firewall and Wi-Fi client isolation. A phone
  VPN or a browser's HTTPS-to-local-HTTP restrictions can also affect access.
- **Only some titles are missing:** try original/Russian title variants and the
  year; check indexer/category filters. Successful indexing does not guarantee
  that reachable peers currently supply the selected release.
- **Using someone else's JacRed:** its operator decides whether RuTracker is
  indexed and its update frequency. You cannot add tracker credentials to that
  shared server through Flow. A local Jackett gives you control without a paid
  provider. It uses the PC as a search service, not as a full-episode downloader.

Self-hosted JacRed is another option, but it needs its own RuTracker login,
database maintenance and potentially additional challenge handling. See the
[JacRed implementation's current RuTracker notes](https://github.com/trinity-aml/jacred-go#rutracker).
It is a larger setup than adding one indexer to Jackett.

These instructions were checked against primary project documentation on
23 October 2026. No Jackett installation, RuTracker login or phone configuration
has been performed or authenticated by Flow's automated tests.
