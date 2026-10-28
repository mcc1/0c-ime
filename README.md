# Windows 0.2.7 混打測試更新包

[下載 ZIP](./0c-windows-mixedpolicy-preview-2.zip?raw=true)（約 17 MB）

測試版版本為 **0.2.8-preview.mixedpolicy.1**，引擎與 Linux 測試版相同，已同步上游 0.2.7。
適用目前已安裝正式版 **0.2.7** 的 Windows x64，同一個 Windows 帳號。不能全新安裝，也不適用 ARM64。
重用經校驗的原有 0.2.7 TSF、UI、設定程式、字典和模型，換上測試引擎與更新工具，保留個人詞庫和既有混打傾向。

1. 完整解壓 ZIP。
2. 儲存工作，用原安裝帳號右鍵 `0c-mixed-policy-test-setup.exe`，以系統管理員身分執行。
3. 安裝完成後登出 Windows，再登入。
4. 雙擊 `Aggressive.cmd` 使用積極模式；`Conservative.cmd` 切回保守。請先開中英混打；內建輸入習慣須先複製成自訂。

未設定混打傾向時預設保守。積極模式可能轉換帳號／型號，需要原文時切英文輸出。
測試包不會自行更改更新模式；要固定測試版，可雙擊 `Manual-Updates.cmd`，要恢復自動更新則雙擊 `Auto-Updates.cmd`。
更新模式偏好會保留，包括回復正式版之後；回復後仍可用 Auto-Updates.cmd 恢復自動。

回復：同帳號右鍵 `Rollback.cmd`，以系統管理員身分執行；完成後登出再登入。保留目前個人詞庫。
只會在本測試版仍是作用中的版本時執行回復。

安裝程式、引擎和更新工具未簽章。Go 組字／設定／更新測試、測試包校驗、Windows x64 交叉編譯已通過；Windows 實機安裝、回復、原生 TSF 密碼欄位仍需實測。
本版沒有鄰近按鍵容錯。詳細說明、雜湊與原始碼在 ZIP 內。

ZIP SHA256：`37d1a1436ac476ff988dfdd29382721bb6c6bde2ee174391a502b343cd55b390`
引擎／更新工具來源：[cc753158](https://github.com/mcc1/0c-ime/tree/cc753158f2c4b8fcce95b891ea434d8addb7e7a8)。
