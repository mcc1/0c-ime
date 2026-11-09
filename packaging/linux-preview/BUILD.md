# Linux preview build

Build from a clean committed checkout on Linux amd64 with Go 1.26.10, Rust 1.99.0, Fcitx5 development headers, WebKitGTK 4.1, GTK3 and a C++20 compiler. The settings build uses the repository's Ubuntu 22.04 Docker image; test the resulting package in Ubuntu 26.04 with Fcitx5 and a fresh user home.

1. Use the verified public dictionary (manifest SHA-256 3e93ed1ef3d2fc7443f64a0b5203294298181b84bed2cd6a144c48a4852aefee) and its matching public selection/punctuation models and portable Linux Python runtime. Validate with scripts/package_release.py prepare_shipped_data(public=True); keep source metadata and model bindings unchanged. These are the same paired assets used in the earlier published Linux preview. Do not substitute the separately verified official dictionary: its manifest differs.
2. `python3 scripts/build_settings_linux.py --version 0.2.11-preview.mixed.1 --out build/linux-preview-settings`
3. `python3 scripts/package_release.py --preview --version 0.2.11-preview.mixed.1 --data /path/to/verified/data --neural /path/to/verified/neural --settings build/linux-preview-settings/linux-amd64 --release-docs packaging/linux-preview --out build/linux-preview-release`
4. Verify package checksums, install into an isolated Ubuntu 26.04 account, test Fcitx5 keystrokes and settings launch, then rollback. Do not install into the build host's actual user home.

Go package, model-pairing and license checks are performed by the existing packager. No input data or private user settings are included. This branch is for combined test builds; feature PR branches remain separate.
