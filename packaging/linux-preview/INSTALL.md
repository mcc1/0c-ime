# Kubuntu 混打測試版

版本：0.2.11-preview.mixed.1。適用 Intel／AMD x86_64、Kubuntu 26.04 與 Fcitx5。
包含正式字典、神經選字／補標點模型、可攜 Python、設定程式、Fcitx5 與 IBus 外殼。
包含與 Windows 試用版相同的保守／積極混打、鄰鍵容錯及設定頁選項分組；以公開上游 0.2.10 原始碼為基底。
這是未簽章的個人測試包，不是原作者的正式發行版。

## 安裝

先安裝系統相依：

```sh
sudo apt update
sudo apt install fcitx5 fcitx5-config-qt fcitx5-frontend-gtk3 fcitx5-frontend-qt5 im-config python3 libwebkit2gtk-4.1-0
```

若平常已用 Fcitx5，保留現有輸入法設定。若尚未使用 Fcitx5，執行 `im-config -n fcitx5`，並登出再登入。
KDE Plasma Wayland 請在「系統設定 → 鍵盤 → 虛擬鍵盤」選 Fcitx 5；不同 Plasma 版本的選單位置可能不同。

解壓縮後在本目錄執行（這兩步不用 sudo）：

```sh
python3 install.py check
python3 install.py install
```

程式安裝至目前使用者的 `~/.local/`，保留個人詞庫。若已安裝靈犀，請先登出圖形桌面，從 Ctrl+Alt+F3 文字主控台登入再執行安裝；原作者的安裝器會檢查桌面是否已停止。
請保存安裝完成時顯示的備份路徑，回復時需要它。

安裝後登出再登入。在「Fcitx 5 設定」取消「僅顯示目前語言」，搜尋「靈犀」並加入清單。
`~/.local/bin/0c-settings` 開啟設定程式；也可以從應用程式選單開啟。

## 測試

若選用內建輸入習慣，先在設定程式「輸入習慣」複製成自訂，才能修改中英混打；已有自訂風格則不必再複製。
全新安裝也可用 `~/.local/bin/0c-update profile copy 0c --name 混打測試 -apply` 複製靈犀風格來測試。
在設定程式開啟中英混打，並切換「混打組字傾向」。預設為保守。請使用標準注音排列、有聲調的輸入方式。
命令列同樣可用（以一般使用者在桌面終端執行）：

```sh
~/.local/bin/0c-update set mixed-input on -apply
~/.local/bin/0c-update set mixed-input-policy aggressive -apply
~/.local/bin/0c-update set neighbor-key-recovery on -apply
```

鄰鍵容錯第一版適用標準注音／QWERTY／積極混打，針對單一鄰鍵及明確聲調；不修改本來合法的讀音，不修純數字，多義時保留原文供選字。可用 `set neighbor-key-recovery off -apply` 關閉。

切回保守：

```sh
~/.local/bin/0c-update set mixed-input-policy conservative -apply
```

比較以下按鍵，按 Enter 送出，每項測試前先清空組字：

- `ki3`、`helloki3cl3`：比較鄰鍵容錯開／關；開啟時應可修成「我」、「hello我好」。
- `593`：兩種模式都應能組出「窄」，學習其他字之後也要成立。
- `microsoft593fmp6`：保守保留帳號／型號規則；積極應能切出 microsoft＋窄＋群／裙。
- `microsoft5993fmp6su3cl3`：觀察誤按後能否繼續組後面的中文。
- `kevin87`、`lisa0304`、`gpt4`：保守應保留原文；積極可能轉成中文。
- 切到英文輸出時，以上內容應保持原文。

積極模式與帳號／型號保留有取捨，輸入需要保持原樣的英數時請切英文模式。
回報請附 Kubuntu 版本、X11／Wayland、保守／積極、實際按鍵與輸出。
避免在回報中貼真實帳號、密碼或公司敏感內容。

## 回復或移除

先登出圖形桌面，從文字主控台登入，再回到解壓的套件目錄執行：

```sh
python3 install.py rollback --backup '安裝時顯示的備份路徑'
```

不加 `--restore-user`，保留目前個人詞庫。第一次安裝、沒有舊版要回復時可用：

```sh
python3 install.py uninstall
```

重新登入後生效。移除後字典、個人詞庫及備份保留。系統 apt 安裝的 Fcitx5 與相依套件不會移除。
