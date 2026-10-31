# 中英混打實驗：事先固定的判定標準 v1

狀態：在正式實驗、候選評測及排名訓練開始前登錄。以包含此文件的 Git commit 與本文件 SHA256 識別版本。這次使用者已授權執行實驗，並在通過後派工實作；先前文件的「尚未授權」是歷史狀態，不再構成重複詢問的理由。

本文件與[實驗計畫](mixed-input-experiment-plan.md)、[evaluator 契約](mixed-input-evaluator-contract.md)共同生效。下列定量門檻優先；其他事項依契約。固定標準不是宣稱方法已可行。單一 agent 執行，主 agent 核對證據並判讀。

## 一、判定結果

- **GO_IMPLEMENT（通過）**：資料、實作正確性、候選品質、排名品質與畫面穩定性的必要條件全部通過，且有可重跑的結果。主 agent 核對後直接派工實作，不再要求重複授權。
- **NO_GO（未通過）**：有效實驗在任一必要條件失敗。保留失敗報告，分析候選搜尋、排名訊號、窗口限制或方法本身的原因；不得把失敗改稱通過。
- **INCONCLUSIVE（證據不足）**：資料不足、必要指標 unavailable、工具／環境未能完成驗證，或實驗有無法排除的洩漏／實作偏差。不得派工替換產品分段。可先完成不依賴結果的驗證工作；仍無法補足時如實結案。

只有小型 pilot 通過不是 GO_IMPLEMENT。缺少 Windows 原生測試不會冒充通過：本次 GO 只允許實作與整合驗證，不允許發布安裝包或宣稱產品 release 通過。

## 二、實驗開始前必須鎖定

1. 原版基準固定 `e754019571573051d8c29e06526802c31d5172e2`。目前分支僅新增文件，runtime tree 的等價性另記。對照為 aggressive preview 2、conservative、同 cap 的 longest-valid-suffix；neighbor/fuzzy/neural/autocomplete 關閉。
2. 資料包、英文資料、實驗程式與規格版本均寫入 run-lock：實際 hash、Git commit、dirty diff hash、執行工具與命令。個人資料只用 disposable stores，不讀日常 user.json。
3. 資料內容、意圖標籤、families、aliases、assignment、primary rows、例外清單，在讀取正式品質結果前凍結。先跑資料結構／語意／數量驗證可以；不能用 baseline 或 proposed 結果挑案例。
4. Phase 2 的 fitting 程式、特徵順序、數值處理、seed、訓練預算、候選 margin 清單與 dev 選擇規則，必須另提交並記 hash，之後才跑排名訓練／test。只可用 train 擬合、dev 選固定候選；test 只作最終判定。
5. 預先列出各指標的分母、不可用情況與 pass boolean；不得在結果出來後改分母、排掉難例或把 unavailable 填成零。每次 run 的原始報告與命令保留。

允許修復實驗程式 bug，但必須記錄修復、使先前受影響結果失效並重跑。同一批已查看的 held-out 結果不能再當新方法的未見測試。方法、cap、特徵語意、資料標籤或門檻變更，須建立新版本 protocol，保留本版負結果，不能覆蓋後繼續宣稱本版通過。

## 三、資料資格

- Pilot 至少 100 Chinese、100 literal、30 ambiguous rows；依契約分組，train/dev/test 各至少 10 Chinese groups、10 literal groups、3 ambiguous groups。pilot 用來找 evaluator 與搜尋問題，不足以支持實作 GO。
- 正式 test 預先收足 **至少 400 Chinese-intent groups 與 400 literal-intent groups**，每組的 primary row 依契約在看結果前固定。這比舊提案的最低 300 更嚴格，理由在實驗前已知：零錯誤 n=300 的 Wilson95 上限約 1.264%，n=400 約 0.951%，後者才可能滿足原訂 1% 上限。份數在看 conversion 結果前決定，不逐次加樣本直到過關。
- 60/20/20 分配及 family/word/prefix/template/context 的 transitive grouping 依契約；hash split、alias overlap 或資料數量不合格時 fail，不重抽 seed、不拆相關群湊數。
- 中文至少覆蓋已知／未知／大寫英文前綴、數字鍵聲母、英文字尾碰撞、invalid-first resumption、直接中文及 context cases；literal 至少覆蓋帳號、型號、版本、日期、小數、時間、URL、底線識別字。category membership 在資料凍結時記錄，每個適用 test category 至少 20 independent primary groups；不適用 category 必須事先說明並限制結論。
- 歧義案例不強迫單一意圖，但 literal 與 intended Chinese 解釋都須保留。相同完整條件互斥的單一意圖標籤，改 ambiguous 或拒收。格式範例不算完整實驗資料。
- 原提案列出的 mandatory cases、原按鍵還原、手選、privacy、reset、63/64/65 bytes、prefix-only 及 two-word cases 全部保留。公開或 synthetic 資料的來源、標籤方法與限制需披露；大量同模板變體不得宣稱獨立證據。

## 四、必要門檻與固定判法

| ID | 指標 | 通過條件 |
| --- | --- | --- |
| G1 | evaluator 正確性 | schema/semantic/assignment 驗證、guard projection parity、baseline observer parity、key ownership/cap/determinism、事件與 freeze 不變量全部通過；每項有實際執行數，零測試不算通過 |
| G2 | privacy／副作用 | protected 狀態 experimental generate/score 呼叫數為 0；hypothetical evaluation 學習／持久化次數為 0；baseline 自己的 disposable commit 學習分開記錄；無跨列 personal state |
| G3 | 候選召回率 | capped strict set 的 intended-boundary recall ≥99%；所有 mandatory Chinese intended path 存活；每個 eligible ambiguous case 的 literal 與 Chinese 解釋均存活；uncapped recall 只作診斷，不能代替 capped 成績 |
| G4 | 中文第一邊界 | 同 test primary rows 上相對 aggressive preview 2 的錯誤率下降 ≥30%：`(eA-eE)/eA >= 0.30`。eA=0 時 eE 必須=0，且明確標「無可量測改善」，不可報無限改善 |
| G5 | 英數誤轉 | literal conversion 點估 ≤1%，且 Wilson two-sided 95% 上限 ≤1%；以預定的 literal groups/primary rows 計算 |
| G6 | boundary 穩定性 | harmful revision=0；unnecessary revision≤1%；closed window 外 ownership revision=0；normal extension 不算 revision；scripted replay 不算模型成效 |
| G7 | 不可破壞條件 | 原按鍵零遺失／零重複；Enter 提交最後顯示；不跨手選／literal lock 改 boundary；conservative 逐事件結果與固定基準完全相同 |
| G8 | 證據完整性 | 所有必要 paired metrics 可用；mandatory/release primary case 的必要 ownership 不得 unsupported；無洩漏、無事後挑列；原始結果、fixture/split/model hashes 與重跑命令齊備 |

主品質分母、projection、row/group 聚合、Wilson 公式、missing-score 與 tie/margin 規則依契約及計畫，不在實驗後重新解釋。完整 partition／reading error、中文字面錯誤、各 category、mapped coverage 與 literal worst-case group incidence 均必須報出，不能只摘最好的總分。未設定門檻的診斷數值不另充作通過理由。

## 五、執行順序與提前停止

1. 完成 evaluator、資料與規格的一致性檢查；凍結 protocol／資料／assignment／程式。
2. 先跑 tiny／exhaustive／parity／事件與安全檢查。G1、G2 或 G7 失敗時先判斷 evaluator bug 或方法問題；在修復前不訓練，不報正式品質通過。
3. 跑 capped/uncapped generation 與 mandatory recall。有效結果未通過 G3 時 **NO_GO，停止 fitting**，分析搜尋／兩音窗口／cap 的限制。
4. G3 通過後才依預先提交的 fitting 規格跑 train/dev；凍結數值模型與 margin，最後跑 test。G4–G8 任一失敗即 NO_GO；必要 evidence 缺失則 INCONCLUSIVE，兩者都不能派工替換 parser。
5. 主 agent 檢查 gate 表、來源與失敗分母，重跑代表性資料與核心檢查後才給最後判讀。只在全部必要 gates 通過時派工實作。

## 六、GO 後的實作範圍與發布分界

GO 後派工先完成 live integration 規格及實作：raw ownership、窗口建立、materialization、候選鎖定、restore、cursor、privacy、learning provenance、conservative fallback。不得把 fixture 人工指定的 run 起點當作產品自動分段已解決。

Windows x64 原生 TSF、UI、回復及延遲是後續 release gates；added parser p95<2ms、p99<5ms 仍保留。缺少它們時可以做實作與整合測試，但不能發布新版安裝包或把 Linux timing 說成 Windows 成績。neighbor 交互作用依原計畫另做 2×2，neural 不作跨 reading 判定。
