# Windows UAC／混打試用版 — 0.2.12-preview.uac.1

[下載 0c-windows-uac-preview-1.zip](0c-windows-uac-preview-1.zip) · [SHA256](0c-windows-uac-preview-1.zip.sha256) · [建置紀錄](build-report.json) · [驗證範圍](verification/README.md)

**需要先安裝正式版 Windows x64 0.2.11。** [官方 0.2.11 安裝檔](https://0c.biggo.com/releases/0.2.11/0c-0.2.11-windows-amd64-setup.exe)。
若目前是舊正式版或之前的測試版，先升級／安裝上述正式版，再執行本升級測試工具。不支援全新安裝或 ARM64。

UAC 修正已 rebase 到上游 0.2.11；本包同時保留積極／保守混打及鄰鍵容錯。
這次**實際替換兩份 x64／x86 TSF DLL**，並非只換引擎。安全模式只保留啟用與按鍵放行，不連引擎、不建立候選介面；不強制背景切換輸入法，也不更動 mode-memory。

1. 完整解壓 ZIP，儲存工作、完成組字、關閉設定視窗。
2. 使用原安裝帳號，以系統管理員身分執行 `0c-windows-mixed-preview-setup.exe`。
3. **安裝完成後登出 Windows 再登入。** 舊 DLL 可能仍在應用程式裡；只關設定視窗不足以更新。
4. 在記事本選 0c，分別測 UAC 按「否」與按「是」之後回到同一視窗，是否仍需 Win＋Space。也確認手動選 ENG 仍正常。

安裝保留個人詞庫、設定與更新偏好，不自動打開新功能。設定裡確認中英混打開啟；內建習慣不可修改時先複製成自訂。
`Aggressive.cmd` 選積極，`Neighbor-On.cmd` 開鄰鍵容錯；`Conservative.cmd`／`Neighbor-Off.cmd` 可分別切回。
0.2.11 的已知型號（如 gpt4）保留原文；積極模式仍可能轉換其他帳號，需英數原文時切英文。
`Manual-Updates.cmd` 可改手動更新，`Auto-Updates.cmd` 恢復自動。

回復：同帳號管理員執行 `Rollback.cmd`，再登出登入；保留目前詞庫。已另裝其他版本時拒絕回復。

本包未簽章，TSF／設定介面以 Rust Windows GNU 編譯，原生 DLL 含品牌圖示與版本資訊；不額外依賴 GCC 執行階段 DLL。
Go、字典、Chromium、Wine COM、原生 PE 與完整官方套件升級預備測試通過；**原生 Windows UAC、註冊與回復仍待本次實機驗證**。

來源：[`edc07bc3838277d80ecff6403f5b0a61a6a0e929`](https://github.com/mcc1/0c-ime/tree/edc07bc3838277d80ecff6403f5b0a61a6a0e929)，ZIP 內附對應源碼、重建腳本、helper/tests 與授權。
SHA256：`bfac5458d5298df71c4d4b58186dc9269ea560431405a4db24a40a554f7b7299`。
