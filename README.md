# Kubuntu 26.04 混打測試版

版本：0.2.8-preview.mixedpolicy.1，Intel／AMD 64 位元，Fcitx5。
已同步上游 0.2.7，包含 593「窄」修正與保守／積極混打選項。
包含正式字典、神經選字／補標點模型、可攜 Python 和設定程式，完整壓縮包約 390 MiB。
這是未簽章測試版，不是上游正式發行版。沒有鄰近按鍵容錯。

先安裝系統相依：

```sh
sudo apt update
sudo apt install fcitx5 fcitx5-config-qt fcitx5-frontend-gtk3 fcitx5-frontend-qt5 im-config python3 libwebkit2gtk-4.1-0
```

下載程式會取得固定 commit 的分段檔，校驗每段及完整套件的 SHA256，再解壓縮。下載中斷可重跑，會重用已校驗的分段。

```sh
curl -fL https://github.com/mcc1/0c-ime/raw/refs/heads/test-builds/linux-mixedpolicy-preview-1/0c-linux-test.py -o 0c-linux-test.py
python3 0c-linux-test.py --install
```

安裝步驟以一般使用者執行，不要 sudo。安裝到 ~/.local/，保留個人詞庫，記下安裝器顯示的備份路徑。
已有靈犀時請先登出圖形桌面，從文字主控台登入再安裝。只想下載、不立即安裝時去掉 --install。
完整說明、回復與移除指令在解壓後的 INSTALL.md。

安裝後登出再登入。在 Fcitx 5 設定加入「靈犀」。尚未使用 Fcitx5 時先用 im-config 選 Fcitx5；KDE Wayland 的虛擬鍵盤選 Fcitx 5。

若使用內建輸入習慣，請先在設定程式的「輸入習慣」複製成自訂，再開中英混打。全新安裝也可先執行 `~/.local/bin/0c-update profile copy 0c --name 混打測試 -apply`；既有自訂風格不必複製。

```sh
~/.local/bin/0c-update set mixed-input on -apply
~/.local/bin/0c-update set mixed-input-policy aggressive -apply
# 切回保守（預設）
~/.local/bin/0c-update set mixed-input-policy conservative -apply
```

積極模式可能把帳號、型號轉成中文，需要保持英數原文時切英文輸出。
測試：593、microsoft593fmp6、microsoft5993fmp6su3cl3、kevin87、lisa0304、gpt4。

完整套件 SHA256：`2d74d9567649df3538e6ad43f9ac8c49f30713f04feca7b4db826046cf0b3ee5`。
引擎及外殼原始碼：[cc753158](https://github.com/mcc1/0c-ime/tree/cc753158f2c4b8fcce95b891ea434d8addb7e7a8)。
以 Ubuntu 22.04、Go 1.26.10、Rust 1.99.0 建置；使用原作者 scripts/package_release.py、prepare_neural.py 與 Tauri 設定介面。
所有授權、模型來源、檔案雜湊與精確來源 revision 隨套件提供。
