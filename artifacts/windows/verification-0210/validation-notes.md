# Windows mixed-input preview on 0.2.10

Source: build/windows-mixed-input-preview-1, commit 56a5881747f61a12ede914082673ed3329525937.
Upstream public release source: 5ab090929c18ff8c05df6bc77a19a2a63587e939 (release/0.2.10).

The official signed public-beta feed reports 0.2.10. Its detached Ed25519 signature was checked against the pinned project key p8+04oez+UEucOQTdHmA7sjI1Pixh5Q4VtepvVBqlb0=. The official ZIP SHA-256 and all 1331 files in its release manifest were verified. The official ZIP embeds revision c337155d2cf9ab0729f358fe27e585ca288d27e6, which GitHub does not currently expose. This preview builds from the published release branch, not from that unavailable revision. Windows native source in the public release branch is unchanged from 0.2.9; the helper keeps the user's verified 0.2.10 TSF, candidate UI, dictionary and neural runtime unchanged.

The merge exposed incompatible account-boundary expectations. Conservative mode now keeps the public upstream 0.2.10 digit-prefix/account boundary rule and reversed-tone recovery. Only aggressive mode extends recovery after a digit-containing typo when the following syllable contains letters. Existing aggressive/conservative tests now explicitly distinguish those policies. Original upstream late-digit-initial tests remain unchanged. Aggressive mode can convert intended accounts/model names; it is an intentional preference and retains original keys in candidates.

Validation:
- core/session tests passed after the policy distinction; core/engine, internal/settings and cmd/0c-server passed in the combined run.
- cmd/0c-update initially had TestParentWatchNoticesParentExit time out during simultaneous compilation. Its isolated retry passed; a subsequent full updater test run also passed. No production parent-watch code changed.
- Go vet passed for core/session, core/engine, internal/settings, cmd/0c-update and cmd/0c-server.
- Published public dictionary and the dictionary extracted from the verified official 0.2.10 ZIP both passed the digit-initial and neighbor-recovery integration tests. Official dictionary manifest: e356ef4aafade0b2897956a12ac8b8f849051e77557f2d9a88adb746daa94c4e.
- Chromium mixed-input option grouping: 2 tests passed. Chromium scroll preservation/navigation: 5 tests passed. WebKit suite skipped because Playwright WebKit is unavailable.
- Rust settings cross-build uses --locked --release --target x86_64-pc-windows-gnu --features tauri/custom-protocol, Rust 1.99.0, version 0.2.11-preview.mixed.1. Cargo.lock is unchanged from the previous settings build, so the exact collected dependency/SDK notices and pinned unmodified WebView2 loader are reused.
- Package builder checks helper preparation, source/receipt tampering, settings preservation, unsupported base versions, embedded payload hashes, Windows x64 PE/imports and every ZIP member hash. See package-build.log for completed results.

Not run: native Windows installation, rollback, TSF input or the Windows settings GUI. Linux cross-build and fixture tests do not establish those behaviors.
