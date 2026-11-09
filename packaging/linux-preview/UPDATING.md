# 測試版更新與來源

版本：0.2.11-preview.mixed.1。基於公開上游 release/0.2.10（5ab090929c18ff8c05df6bc77a19a2a63587e939），與 Windows 試用版使用相同的組字邏輯。保守模式保留上游帳號邊界；積極模式允許較積極的中文恢復。Shift 自動回中文的研究排名器未合入。

原始碼：https://github.com/mcc1/0c-ime/tree/build/linux-mixed-input-preview-0210
套件 release.json 記錄精確的 revision；以該 revision 取得源碼即可重建。所有檔案由 SHA256SUMS 與 release.json 記錄雜湊。第三方聲明隨套件附上，字典及模型保留原有來源與配對資訊。

這是手動分發的個人測試版，不是上游正式版。安裝保留原有更新偏好；自動更新若套用正式版，可能替換測試功能。需要固定此版時，可用 `~/.local/bin/0c-update mode manual` 改手動更新，之後用 `mode auto` 恢復。
更新、回復或移除請依 INSTALL.md 與安裝器提示操作，先儲存工作並登出圖形桌面。
