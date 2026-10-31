# Shift-return 計分演練（假設輸出，非實驗結果）

本文件用來檢查[protocol](shift-return-protocol-v1.md)是否會獎勵正確行為、懲罰誤傷。下列選擇全部是假設，不是程式實跑成績；不能算進正式資料。

每列的完整任務從中文模式開始，先按Shift，再逐鍵輸入raw，最後Enter。E0不以後文更新切點證據；EC使用後文。策略看不到「意圖」欄。所有同一raw的列有相同可見設定／前文，不能假裝系統可以同時猜對。

| 任務 | raw | 欲輸出 | E0假設最終顯示 | EC假設最終顯示 |
| --- | --- | --- | --- | --- |
| a，中文 | `microsoft593vul3` | microsoft窄小 | `microsoft593vul3` | microsoft窄小 |
| b，注音式帳號 | `microsoft593vul3` | `microsoft593vul3` | `microsoft593vul3` | microsoft窄小 |
| c，中文 | `WQ5j04fu06` | WQ賺錢 | WQ賺錢 | WQ賺錢 |
| d，型號 | `gpt4` | `gpt4` | `gpt4` | `gpt4` |

假設EL和E0同結果。a/b同一collision component，c/d各一component。a/b不能拆train/test，也不能因中文詞籍存在就刪掉b。

## 逐鍵怎麼看a

| raw prefix | EC暫時顯示 | 計分解釋 |
| --- | --- | --- |
| `microsoft593` | `microsoft593` | 此時仍有兩種意圖，不要求永久定案；記首音節尚未成功 |
| `microsoft593v` | `microsoft593v` | 後字未完成，不能偷用未來的`ul3` |
| `microsoft593vul3` | microsoft窄小 | 後文到達後自動修好，記context repair、最終成功；仍需記恢復花了4鍵 |
| Enter | commit=microsoft窄小 | 必須等於上一個顯示，Enter不能再換另一個答案 |

同一份EC在b也會顯示「microsoft窄小」，因此b是最終literal誤轉。即使這是一個正常中文字詞，也不能當帳號任務的成功。這正是要測的取捨。

## 這四列的手算

- return意圖a/c：E0錯1/2，EC錯0/2，錯誤下降100%；a是淨修好，c維持正確。
- literal-only意圖b/d：E0錯0/2，EC錯1/2。即使中文看起來有進步，仍不通過literal限制；樣本2也完全不具正式資格。
- a/b完整觀測相同且兩意圖等權，對該component最多同時命中1/2；若改為全保留literal，只會把成功與失敗交換，不是解決歧義。
- a中途literal後自動組中文，不能算最終失敗；若a在第二音節曾顯示正確、後來變回literal而最後仍錯，則算最終失敗。
- 假如cut/readings正確但字面是「microsoft窄筱」（小／筱同讀音），意圖／讀音項可成功，整段字面項失敗；不得把兩項混為一個90%成功率。
- 若EC直到Enter才從literal改中文，即使commit看似正確，亦違反Enter不得排名的條件；若首字raw早已送入應用程式、只能轉後一字，不能拿shadow預測當輸出成功。

正式報告須同時提供return、literal、英文span保留、context修好／改壞、讀音及字面的分母與結果。這份演練說明為何不能只選a這類改善案例，也不能要求所有早期暫時顯示一次就猜中。
