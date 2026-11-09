# Validation notes

The package's public dictionary and paired models were checked without changing their source metadata or binding hashes. Full installation and keystroke tests ran only in isolated Ubuntu 26.04 test accounts. Their learned stores contain synthetic test input. No real user home was installed into or included in the package.

Shared fixture modes were normalized so the separate test accounts could read and execute files unpacked by the build account. The official archive contains owner-only execution bits; its original archive and all file contents remained unchanged. Fresh installation, upgrade from that verified official 0.2.10 archive, rollback and user/settings/update-mode preservation passed.

The initial GitHub /raw download endpoint returned HTTP 504. The downloader now uses raw.githubusercontent.com directly and retries transient transport errors. Reusing its verified cache, every published part was downloaded and the combined archive hash and all 2425 release manifest entries were checked successfully.

Native KDE Plasma Wayland and Qt6 application input remain untested. The settings launch test used an isolated Xvfb display with WebKit sandbox disabled only inside the container. The package and installation instructions do not disable the user's WebKit sandbox.
