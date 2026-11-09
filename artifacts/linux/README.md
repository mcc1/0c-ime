# Kubuntu 26.04 Linux 混打試用版

版本：**0.2.11-preview.mixed.1**，Intel／AMD x86_64，Fcitx5。可全新安裝，無須先裝正式版。
完整包約 390 MiB，含設定介面、公開字典、配對模型、可攜 Python、Fcitx5 與 IBus 外殼。
與 Windows 試用版使用相同的積極／保守組字和鄰鍵容錯；不含 Shift 自動回中文的研究排名器。
使用公開字典與模型，與正式版可能有選字差異。這是未簽章的個人試用包。

先安裝系統相依（Kubuntu 26.04／Plasma 6 包含 Qt6 前端）：

```sh
sudo apt update
sudo apt install curl python3 fcitx5 fcitx5-config-qt fcitx5-frontend-gtk3 fcitx5-frontend-qt5 fcitx5-frontend-qt6 im-config libwebkit2gtk-4.1-0
```

以一般使用者下載與安裝，**不要 sudo 執行安裝程式**。若已有靈犀，先儲存工作並登出圖形桌面，從 Ctrl+Alt+F3 文字主控台安裝。

```sh
curl -fL https://github.com/mcc1/0c-ime/raw/refs/heads/build/linux-mixed-input-preview-0210/artifacts/linux/0c-linux-test.py -o 0c-linux-test.py
python3 0c-linux-test.py --install
```

下載程式核對每段與完整套件 SHA256，再解壓並執行原作者的使用者目錄安裝器。中斷可重跑，會沿用已驗證的快取。
只下載與解壓可省略 `--install`。套件安裝至 `~/.local/`，保留個人詞庫、設定與更新偏好；記下安裝器顯示的備份路徑。

安裝後登出再登入，在 Fcitx5 設定加入「靈犀」。尚未使用 Fcitx5 時先執行 `im-config -n fcitx5`；KDE Wayland 在「虛擬鍵盤」選 Fcitx5。
從應用程式選單或 `~/.local/bin/0c-settings` 開啟設定。

若使用內建輸入習慣，先複製成自訂習慣，再開中英混打、選積極並開鄰鍵容錯。新安裝可用：

```sh
~/.local/bin/0c-update profile copy 0c --name 混打測試 -apply
~/.local/bin/0c-update set mixed-input on -apply
~/.local/bin/0c-update set mixed-input-policy aggressive -apply
~/.local/bin/0c-update set neighbor-key-recovery on -apply
```

改為 `mixed-input-policy conservative` 可回保守；`neighbor-key-recovery off` 可關鄰鍵。
積極模式仍可能轉換帳號／型號；需要英數原文時切英文輸出。鄰鍵第一版適用標準注音／QWERTY／積極模式、單一誤按與明確聲調。
可測試 `593`、`microsoft593fmp6`、`ru,3WQ5j04fu06`（解WQ賺錢）、`ki3` 與 `helloki3cl3`。

若要固定此版，可執行 `~/.local/bin/0c-update mode manual`；之後 `mode auto` 恢復自動更新。
回復與移除見解壓後 INSTALL.md；回復前先登出桌面，使用安裝時顯示的備份路徑，預設保留目前個人詞庫。

驗證：Ubuntu 26.04 隔離帳號的全新安裝、8 項真實 Fcitx5→GTK 輸入、神經模型啟用、設定 GUI 啟動、從正式版 0.2.10 升級與回復，以及詞庫／設定／更新模式保留均通過。
另通過 Fcitx5 C++ 外殼測試、公開字典測試、配對與來源檢查及完整套件雜湊驗證。**尚未驗證實體 Kubuntu 的 KDE Wayland 與 Qt6 應用程式輸入。**

[建置紀錄](build-report.json) · [驗證紀錄](verification/) · [重建說明](../../packaging/linux-preview/BUILD.md)
精確源碼：[e640e46f7af75e3c4c56c8d33ee6073bf494c27c](https://github.com/mcc1/0c-ime/tree/e640e46f7af75e3c4c56c8d33ee6073bf494c27c)（AGPL-3.0-only），授權與第三方聲明隨包附上。
SHA256：`7c1625841aef280bb21259777fe0999eb535f697d3047a43841d793a6ca80cbb`。
