# 中英混打實驗 v1：主 agent 最後判讀

結論：**NO_GO，不派工把這版排名器接入輸入法。** 實驗由一位 agent 執行，主 agent 獨立核對。固定候選保留機制值得繼續研究，但這次固定的線性排名器與穩定 margin 明顯退步；必要證據也尚未完整。這不是可發布的輸入法版本。

## 判斷依據在結果前已固定

[判定標準](mixed-input-experiment-acceptance.md) 在 commit `1c927821b7ae7e4863cf8a783dbffc712429167b` 登錄，SHA256 為 `dd42aabb1767746bf62b9d7604ac78e9300c693c7f9c6def70448b6303df5b2a`。資料／evaluator 在 `169adcf` 凍結，訓練與 dev 選擇程序在 `1244ee6` 凍結，數值模型與 margin 在 `9203c4f` 凍結後才跑最終 ranked test。未依 test 調權重、margin、分組或門檻。

詳細執行、重現方法與限制見[執行報告](mixed-input-experiment-results-v1.md)。本判讀採預定 test independent groups 的 primary rows，不把大量同來源變體當作獨立證據。

## 固定門檻與結果

| 項目 | 事前要求 | 實際證據 | 判讀 |
| --- | --- | --- | --- |
| G1 evaluator | 所有必要正確性成立 | 已執行 schema、分組、guard、observer、exhaustive 與事件檢查通過；但整段 context 呈現／提交驗證有缺口 | 完整條件尚不可用 |
| G2 隱私／副作用 | protected generate/score、假設路徑學習／持久化皆 0 | 執行檢查皆 0，baseline 使用逐列 disposable stores | 已測範圍通過 |
| G3 候選 | capped recall ≥99%，mandatory 全保留，歧義兩解釋全保留 | test primary 492/492；mandatory 10/10；ambiguous 150/150 | 條件式召回通過 |
| G4 中文第一邊界 | 相對目前 aggressive preview2 錯誤下降 ≥30% | 40/492（8.13%）→178/492（36.18%）；相對下降 −345% | **失敗** |
| G5 英數誤轉 | 點估與 Wilson95 上限均 ≤1% | 1/164（0.61%）；上限 3.37% | **未達門檻** |
| G6 穩定性 | harmful=0；unnecessary ≤1%；窗口外變更=0 | harmful 0；unnecessary 10/656=1.52%；窗口外 0 | **失敗** |
| G7 不可破壞條件 | 無掉鍵／重複、完整 Enter／鎖定／保守路徑驗證 | run 內記錄無掉鍵／重複／提交不一致；context prelude 未納入 proposed 整段顯示與提交檢查 | 尚不可用 |
| G8 證據 | 正式樣本、各分類與必要 paired metrics 齊備 | 中文 test 492 組；literal 僅164組，未達400；分類／ownership 缺口仍在 | 尚不可用 |

因此不能宣稱完整正式實驗合格。即使先不看樣本與工具缺口，已可用的中文配對負證據與多餘改動率也不支持整合，故採 **NO_GO** 決策，而非把這輪只解讀成「再加資料就可能通過」。英數上限未過不等於已證明真實誤轉率大於1%；它表示現有證據無法支持預定安全要求。

中文 primary 中，原本正確而這版變錯的有146組，原本錯而修好的僅8組。候選有保留與最後能選對是不同問題。

## 從實例與對照能得到的結論

以下為凍結 raw events 的結果，並非事後手造測試：

| 按鍵 | 目前 aggressive preview2 | 本次排名器 |
| --- | --- | --- |
| `593` | 窄 | `593` |
| `WQ5j04fu06` | WQ賺錢 | WQ賺錢 |
| `284xo6` | 大雷 | `284xo6` |
| `QZXdropkicksw.6fu4` | QZXdropkicks投契 | `QZXdropkicksw.6fu4` |
| `ME-6100/4/DIO`，意圖 literal | ME而100/4/DIO | ME而100/4/DIO |

排名器常保留 literal，連原本直接中文也可能退回原鍵。這與「未知前綴後難恢復」一致，但目前不能把原因只歸給某一權重或 margin。固定訓練預算的 loss 有下降，仍有非零梯度，沒有證明最佳化已收斂，也沒有排除其他線性模型或神經模型。

同一凍結資料的因果 longest-valid-suffix 對照把中文錯誤降至24/492，相對下降40%，但 literal 誤轉達23/164（14.02%）。因此直接採「能組就優先中文」也無法同時滿足兩個目標。這項對照使用自己的逐事件歷史；舊條件式／terminal longest 數字不作正式比較。

兩次 tone opportunity 的窗口也有風險：失敗的聲調鍵會消耗機會。保留的 dev 反例 `QQQ3visionsnj04ru7`（意圖「算計」）顯示，先前錯誤上下文可能使後續完整詞路徑無法接續。G3 是以固定 baseline 歷史量測的條件式召回，不保證新政策自己的所有歷史都能召回。

## 證據限制與獨立核對

- 資料為 synthetic 中文意圖及公開 USB／PCI 型號詞，不能推論真實使用者誤按分布。literal 只有164個獨立 test groups；相依來源不可拆組湊400。
- 凍結的 known-prefix 分類含266列未驗證英文詞籍的複數化前綴。維持原標籤、披露限制，不在看結果後重新分類。數字聲母、context 等部分分類不足20組。
- 三個 baseline literal primary 的 ownership unsupported，必要完整 paired literal 比較不可用；161個 mapped rows 的數字只作診斷。primary test 沒有完整 homophone 字面 gold，中文字面錯誤不能填0。
- context 案例 baseline 提交「解WQ賺錢」，proposed 工具只記後段「WQ賺錢」。這是整段呈現／提交驗證缺口，不能將 run 內 commit 一致冒充整 composition 驗證。
- fixture 指定 run 起點；live window discovery、Windows TSF/UI、延遲、neighbor 交互作用未驗證。本輪沒有新增 EGM 多階段 review，沒有發布安裝包。

主 agent 獨立重播26769個 test generation rows，語意結果一致；重現訓練取得相同模型 hash；重做固定 dev 選擇取得相同 margin 0.25。最終 ranked test 沒有為挑結果重跑，主 agent 從凍結 raw rows 獨立重算上述品質數字並檢查代表性事件。

最終模型 SHA256：`f347d8ef562c3b830924693d378de5eab35ae1f53740781f5834d0493db91b21`。ranked rows SHA256：`2f4fe7d09971fa7e6c6b97c7e0b2169619927c874e0e3967890d6a16eba1e45c`。獨立判讀 JSON 位於 `/workspace/scratch/mixed-input-experiment-run/parent-final-metric-adjudication.json`。大型 raw evidence 留在同一目錄，精簡重現工具與報告入版控。

## 下一個方法應如何改

不把本版套入產品，也不針對「窄」或已見 test 個案補權重。保留目前產品的 aggressive／conservative 行為與作者帳號／型號規則。

下一份 protocol 應先處理工具與資料資格，再評估更窄的介入方式：讓既有 decoder 繼續負責一般直接中文與已確定的中文組字，新方法只在有衝突的英文／中文切點提出替代路徑；這是待驗證假說，不是已證實改進。候選保留機制可作為研究基礎。

窗口要另比較「聲調機會」與「成功音節／詞邊界」的效果，同時保持 raw64 等成本上限，避免一次誤按耗掉恢復機會。排名須同時處理完整讀音、英文／識別字完整性與先前上下文；穩定規則需避免首次偏向 literal 後長期抑制正確恢復。不能只以更長中文為勝。

正式下一輪之前，先補完整 prelude materialization、unsupported ownership 與字面 gold；固定採樣預算，收足400個獨立 literal test groups及分類，並預先定義精確英文詞籍。新訓練／選模／收斂規則、窗口比較和門檻都需重新登錄。本輪 test 已被看過，只能作回歸資料；新的方法須使用新的未見 test。這些是失敗後的方法評估，尚未派工執行下一輪或產品整合。
