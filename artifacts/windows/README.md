# Windows 混打功能測試包 — 0.2.10-preview.mixed.1

[下載 0c-windows-mixed-preview-1.zip](0c-windows-mixed-preview-1.zip) · [SHA256](0c-windows-mixed-preview-1.zip.sha256) · [建置與驗證紀錄](build-report.json)

**需要先安裝正式版 Windows x64 0.2.9。這是升級測試包，不能全新安裝，也不支援 ARM64。**
未簽章，尚未在 Windows 實機驗證；保留詞庫、設定與更新偏好，附回復工具。

包含最新積極／保守混打及鄰鍵容錯，設定子選項與捲動修正；不含尚未完成品質實驗的 Shift 回切排名器。
從 fork main（上游 0.2.9）開獨立 build 分支，依序 merge 兩個功能分支，沒有修改功能分支。

1. 完整解壓 ZIP，完成目前組字並關閉設定視窗。
2. 使用原安裝輸入法的同一 Windows 帳號，右鍵 `0c-windows-mixed-preview-setup.exe`，以系統管理員身分執行。
3. 安裝完成後重新開啟設定與要測試的程式；依更新工具提示處理，不強制登出。
4. 確認中英混打開啟，使用設定頁或 `Aggressive.cmd` 切積極；`Neighbor-On.cmd` 開啟鄰鍵。
   內建輸入習慣若不能修改，先複製成自訂習慣。

安裝本身不改新功能開關與更新模式。`Manual-Updates.cmd` 可改手動更新；`Auto-Updates.cmd` 恢復自動。
`Rollback.cmd` 以同帳號管理員身分執行，可回復安裝前版本並保留目前詞庫；若已另安裝其他版本則拒絕。
詳見 ZIP 內 README、來源與授權。

來源 commit：`0f0ebbe02a80a43cd43ec6dc3ac77d1b58b27ff4`。源碼與 helper/test 原碼一併附在 ZIP；重建入口為
`scripts/build_windows_feature_preview.py`，設定程式用 Rust GNU 的 `tauri/custom-protocol` build。

SHA256：`a588dddde51de42404e67642fc86d903866286ebdee759ae3cd9a5a8eb97c521`

本機已驗證準備／防竄改／設定保留、交叉編譯、PE、ZIP、公開字典與 Chromium 6 項 UI 測試；
這些不代表 Windows 上的 TSF、原生 GUI、安裝／回復已驗收。
