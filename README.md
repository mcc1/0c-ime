# 靈犀注音 Windows 混打測試版

版本：`0.2.7-preview.mixedpolicy.1`，適用於**已安裝 0.2.6 的 Windows x64**，不是全新安裝包，也不是上游官方發行版。

[下載測試 ZIP](https://github.com/mcc1/0c-ime/raw/refs/heads/test-builds/windows-mixedpolicy-preview-1/0c-windows-mixedpolicy-preview-1.zip) · [SHA-256](0c-windows-mixedpolicy-preview-1.zip.sha256)

解壓後，以原安裝帳號的系統管理員身分執行 `0c-mixed-policy-test-setup.exe`，安裝完成後登出再登入。
預設保守模式；雙擊 `Aggressive.cmd`／`Conservative.cmd` 切換。需先開啟中英混打。
回復時以系統管理員身分執行 `Rollback.cmd`，再登出登入。

本包未簽章。Go 與元件校驗測試已通過；Windows 原生安裝、介面及 TSF 密碼欄位尚待實機驗證。
詳細步驟、測試案例、完整原始碼快照與組裝程式均在 ZIP 內。

原始碼：[cccc2d1](https://github.com/mcc1/0c-ime/commit/cccc2d1397a574433d8faf90a1f6f6b7d1b12c3d)

ZIP SHA-256：`633c942d8fd0e1a5f4155af1eea3e0819dc24872998e13c7c3a18c89d296aa2a`
