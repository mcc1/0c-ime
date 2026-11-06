# Windows 混打功能測試包 — 0.2.11-preview.mixed.1

[下載 0c-windows-mixed-preview-2.zip](0c-windows-mixed-preview-2.zip) · [SHA256](0c-windows-mixed-preview-2.zip.sha256) · [建置紀錄](build-report.json) · [驗證細節](verification-0210/validation-notes.md)

**需要先安裝正式版 Windows x64 0.2.10。** 這是升級測試包，不能全新安裝，也不支援 ARM64。
未簽章，尚未在 Windows 實機驗證安裝、回復與 TSF。

基於上游公開 `release/0.2.10` 原始碼，合併積極／保守混打、鄰鍵容錯及設定頁子選項。
保守模式保留上游帳號邊界；數字誤按後較積極的中文恢復只在積極模式啟用。
積極模式仍可能轉換帳號／型號。不含 Shift 回切研究排名器。

1. 完整解壓 ZIP，完成組字並關閉設定視窗。
2. 使用原安裝輸入法的同一 Windows 帳號，以系統管理員身分執行 `0c-windows-mixed-preview-setup.exe`。
3. 重新開啟設定與要測試的程式，確認中英混打開啟；執行 `Aggressive.cmd` 選積極，`Neighbor-On.cmd` 開啟鄰鍵容錯。
   內建習慣若不能修改，先複製成自訂習慣。

安裝保留個人詞庫、設定與更新模式，不自動打開新功能。可用 `Manual-Updates.cmd` 切手動更新、`Auto-Updates.cmd` 恢復自動。
`Rollback.cmd` 可回復安裝前版本並保留目前詞庫；若已另裝其他版本會拒絕。

來源 commit：`56a5881747f61a12ede914082673ed3329525937`。ZIP 附對應源碼與授權；重建入口為 `scripts/build_windows_feature_preview.py`。
官方二進位的 commit 尚未能公開取得，程式以已公開的 release 分支建置；TSF、字典及模型執行環境沿用已驗證的正式版 0.2.10 安裝。

Go 測試與 vet、兩套公開／官方字典、Chromium 設定頁、helper 防竄改與保留測試、Windows 交叉編譯及 ZIP 完整性均通過。
另以完整官方 0.2.10 套件驗證預備升級，產物通過完整驗證；這不代表已在 Windows 實機安裝驗收。

SHA256：`480569df0705023d346aaa009cf5dd1f8106878c47b6c89965bfe5cb503b868c`

歷史版（限正式版 0.2.9）保留在 commit `401d92d25ecd53b9eadab7ba20e8df999ad5333e`。
