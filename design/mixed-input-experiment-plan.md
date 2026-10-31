# 中英混打邊界：離線實驗計畫

這次先回答「保留兩種切法，能否讓下一音節提供有用證據」，不修改目前輸入法或製作安裝包。[evaluator 契約 v1](mixed-input-evaluator-contract.md) 是 Phase 1 的實作依據；[schema](mixed-input-experiment/fixture.schema.json) 與[格式範例](mixed-input-experiment/examples.jsonl)讓事件與標籤可被機械檢查。這些文件沒有代表實驗已經執行或 EGM 已重新審查通過。

## 假設與對照

- H1：最多64個實體 ASCII key bytes、兩個明確聲調的 exact suffix 候選，能在 caps 內保留正確切法。先測 recall，錯過 intended path 時不訓練排名救場。
- H2：兩音節完整詞／完整雙詞的統計證據，能降低中文意圖的首個中文邊界錯誤，同時保護帳號、型號與純 literal。
- H3：只容許第二個完成聲調改一次 boundary、其餘事件 freeze，可限制干擾。這是待驗證假設；不把普通追加一音節算成 revision。

配對比較四條 policy：現行 preview-2 aggressive（含 digit-initial override）、原 conservative compatibility、相同候選 caps 的 longest-valid-suffix，以及稍後才擬合的 linear ranker。現行 aggressive 的帳號轉換如實記錄，不能為 baseline 手工改成標籤答案。第一輪 neighbor recovery、fuzzy、neural、English autocomplete 全關。所有查詢本機且唯讀，personal stores 每列隔離，公開 baseline 的 Private=false；專門 privacy case 才設 private。

範圍刻意限制為標準排列、組字尾端、3/4/6/7 明確聲調。第一聲空白、游標編輯與原樣選取作 compatibility／freeze 事件測試，不新增空白切分搜尋；E 的 incomplete pending 固定顯示 raw，尚不驗證產品 phonetic pending UI。context案例由fixture明確宣告frozen prelude與run起點（例如`ru,3`作prelude、`WQ5j04fu06`作run）；尚不驗證live自動發現這個切口。literal prefix 會重置中文延續 context；中文前文／retained prefix 的 reachability 用純 immutable projection 保留。無 live Session 可供 hypothetical 試打，不能靠 clone session 重播挑選最佳結果。

## 執行階段與產出

| 階段 | 工作與固定產出 | 進入下一階段條件 |
| --- | --- | --- |
| 0，這次文件 | 完成 C1 event/overflow/history、C2 dependencies/features/guards、C3 fixtures/grouping/split 契約 | 審查本規格與來源對應；尚未授權 Phase 1 實作 |
| 1a，未來 adapter | 新增純 context／guard projections、只觀察的 legacy ledger；原版與 instrumentation 逐事件 parity；schema/semantic validators、split assignment | 所有 parity、privacy、key ownership、finalization invariant 通過；任何 untraceable legacy byte fail closed |
| 1b，未來 generator | exact 枚舉、cap／literal／opaque legacy lane；輸出 uncapped/capped recall、各 guard/vector、A/C 與 longest baseline 報告 | 中文 intended recall ≥99%（pilot 僅探索性），mandatory case survival、無 key loss；若失敗先版本化檢討 generator/cap，不開始 fit |
| 2，另立研究規格 | 在 train 擬合小 linear ranker，dev 選穩定 margin；凍結數值規格後只跑一次 held-out | 同時通過下列品質門檻才允許提出 Phase 3 規格；負結果保留 |
| 3，另立 live 規格 | materialization/locks/restore/UTF-16/native TSF、Windows latency、neighbor 交互矩陣 | 完成 native evidence 前不發 opt-in 測試包；conservative 全程原路 |

Phase 1 的預定檔案清單（這次不建立程式）：

- `core/mixedparse/{types.go,context.go,guards.go,generator.go,features.go}` 與對應 `_test.go`：只接 values 與 read-only interfaces；不依賴 Session，不建立新個人資料 API。
- `core/session/mixed_eval_adapter.go`（build tag `mixedeval`）、`mixed_eval_adapter_test.go`，以及 `mixed.go` 的單一 `insertMixedTyped` nil observer hook：只觀察 retained-prefix provenance，零分支變化。observer interface 與 no-op 宣告於 `core/session/mixed_eval_observer.go`，一般編譯不暴露 runner。先 parity 再 refactor；不得把此工作當成替換 mixed parser。
- `internal/mixedeval/{fixtures.go,validate.go,split.go,replay.go,legacy.go,report.go}`、`cmd/0c-mixed-eval/main.go`（build tag `mixedeval`），`testdata/mixed-boundary/{fixtures.jsonl,aliases.json,assignment.json}`。report schema 以本契約固定欄位實作，未知欄 fail，不擴大現行 desktop proto。

未來 CLI 固定：`go run -tags=mixedeval ./cmd/0c-mixed-eval -data assets/data -fixtures testdata/mixed-boundary/fixtures.jsonl -aliases testdata/mixed-boundary/aliases.json -assignment testdata/mixed-boundary/assignment.json -phase enumerate -split all -out <isolated-directory>`。`-phase validate` 只做結構／語意／assignment／adequacy；`enumerate` 讀同一 frozen run-lock，產生 candidates.jsonl、events.jsonl、report.json，不能 fit 或修改 assignment；新增 split 用獨立 `-phase assign`，輸出 provisional assignment 供 review，若已存在就拒絕覆寫。`-split all|train|dev|test`；未凍結的 ranker 不接受 ranked phase，無安裝／發布子指令。

report header 必含契約列出的 provenance。candidate rows 固定含 `fixture_id,event,window,candidate_id,lane,segments,readings,support,raw_score,score_status,features,feature_version,history,pruned`；event rows 含 `policy,state,raw,visible,pending,partition,revision_count,closure_reason,generate_calls,score_calls,delegate_count,commit,locks`，缺值 JSON null，不以0冒充測量。header／report／row schema 實作時直接依這些欄位，不變動 fixture schema；timing 另記 `duration_ns` 與 event category。

## 資料與避免洩漏

pilot 下限100 Chinese、100 literal、30 ambiguous rows，含 known/unknown/capitalized prefix、帳號與型號、dates/decimals/time/URLs/underscores、invalid-first resumption、context/retained prefix、original keys與手選、reset/privacy/settings、63/64/65 bytes。每 split 必須有至少10 Chinese groups、10 literal groups、3 ambiguous groups；不足即補獨立原型與新版 split，不搬 family。這是 debug與方向判斷資料，不能據此聲稱 release 百分比。

mandatory examples：`593`、`microsoft593fmp6`、`WQ5j04fu06`、`ru,3WQ5j04fu06`、`hello5j04fu06`、`hellosu3cl3`、`testsu.3`、`appsul4`、`useru/4`、`api2k7`；literal labels 包括 `kevin87`、`nina87`、`lisa0304`、`gpt4`、`h264`、`gpt4o`。不同明確context可有同raw與不同intent但必須同group；完整條件相同卻互斥的labels必須改ambiguous或拒收；`WQ`→`WD`仍是未重現回報，不列成已修復結果。

所有 raw／意圖只用公開來源（附版本、license／允許用途）或人工 synthetic；不得寫入私人 corpus、trace或 user.json。family 同 prefix、中文詞、literal prototype、template/source、context 的任一 overlap 都 union，transitive component 只能在一個 split。固定 SHA256 seed=`20261101` 60/20/20，assignment check-in。不能在看過 test 結果後改 family／seed或挑掉失敗列。

release validation 額外需要 frozen test ≥300個 Chinese-intent independent groups與≥300個 literal-intent groups，以每 group predetermined primary row 等權；variants另報worst case與row數。300是最低數量，**不保證1% Wilson95上限通過**；零轉換 n=300 的上限約1.264%，仍須補獨立 test families並以新 protocol／dataset 凍結，不沿用已看的 test來調參。

## 指標、tie 與停止規則

以下主品質指標以 held-out unambiguous primary rows 計算；pilot另附所有 rows 作diagnostic。若一列多個 terminal windows，先以列內任一失敗算該列失敗；不把長列當多個獨立樣本。ambiguous只測 literal與Chinese accepted alternatives survival，排除 forced top-one。分母0輸出null+count0，不輸出100%。每個計數、分母與Wilson95區間都寫出。Wilson 用z=1.959963984540054，標準two-sided interval：center=(p+z²/2n)/(1+z²/n)，half=z√(p(1−p)/n+z²/4n²)/(1+z²/n)。

| 指標 | 分子／分母及可用階段 |
| --- | --- |
| intended-boundary recall | accepted完整raw partition+exact reading在capped strict set內的 Chinese rows / Chinese rows；另報uncapped及pruning loss；Phase1可用 |
| first-Chinese-boundary error rate | 選擇的第一Chinese byte offset錯／缺失的Chinese rows / Chinese rows；同first offset但reading錯另報完整partition error；A/C/longest Phase1可用，proposed待Phase2 |
| complete-partition／exact-reading error | partition+ordered reading不在accepted集合 / Chinese rows；reading與同reading homophone錯分開報 |
| Chinese-word output error | selected partition/readings正確但Chinese surface不在accepted_text / 正確reading且有text gold的Chinese rows；另報end-to-end output error / 全有text gold Chinese rows |
| literal conversion | 任一raw literal byte自動改為Chinese／改寫literal的 literal rows / literal rows；explicit selection不算自動 conversion |
| useful revision | 前shown不在gold、後shown在gold且ownership改變的rows / 所有unambiguous rows |
| harmful revision | 前shown在gold、後shown不在gold且ownership改變的rows / 所有unambiguous rows |
| unnecessary revision | 前後shown都不在gold但ownership改變的rows / 所有unambiguous rows；等價gold間切換另報benign |
| changes per key | ownership revision事件數 / 進入eligible window的raw key事件數；普通extension排除，word-only display changes另計 |
| outside-window changes | freeze後既有byte ownership改變的event數（及受影響rows）／所有key events（及rows）；必須0 |
| timing | 單event parser generation+features+policy總duration與A配對差值；每policy／category報樣本數、p50/p95/p99，cold load另報；Phase1非Windows只算探索 |

gold correctness判定對同一舊raw prefix投影，避免第二音節尚未輸入時因未來suffix不存在而把首tone判錯。revision只比較前一 completed-tone offset的bytes與completed spans；同一 first offset 的 reading改變仍屬ownership/span identity revision。Enter不能新增ranking。proposed數據在ranker存在前填null，scripted顯示不當成成效。

fit前必須另凍結training config、coefficients精度、missing-value處理、causal history規則與margin。Phase1採legacy-lagged history只做feature audit；不可用gold previous state。ranked replay tie規則固定：score以float64依vector index依序累加；exact equality才算tie，不設epsilon；同分先留目前shown（若仍在set）、再legacy exact-equivalent、再firstChinese offset asc、identity UTF8 asc，literal offset視為+∞。第二tone先在候選中找投影至firstToneEnd與目前shown相同的最高分延伸作incumbent，同分依上述順序；普通extension沒有revision。挑戰者只有 `score(new)-score(incumbent)>frozen_margin` 才可改。若沒有相容延伸（含cap pruning），freeze舊shown並將新bytes原樣附加，記`fixed_fallback`、score=null、final partition可為Chinese後接literal；這是不可排名的保護輸出，不增候選lane、不算generator recall成功，仍計output error。不可將舊短candidate ID拿去直接和較長raw的candidate ID比「是否存在」。margin必須非負finite，dev選好後不能改test。若ranker numerical score非finite立即fail。

Go候選生成要求：所有安全invariants通過、mandatory accepted candidate存活、held-out recall≥99%。排名進live規格的Go要求：conservative逐事件輸出完全不變；中文first-boundary error相對A下降≥30%：`(eA-eE)/eA>=.30`，eA=0時E必須0且不宣稱改善；literal conversion點估與Wilson95上限均≤1%；harmful revision=0、unnecessary revision≤1%、closed後revision=0；原始keys不變／不遺失／不重複。品質門檻除literal要求CI上限外按點估，但全部報CI，不能只報百分比。

任一安全、privacy零inference／零學習、parity或資料洩漏失敗立即Stop；無法在相同凍結test同時達門檻也Stop，保留負結果。不得為了過關再加帳號例外。Windows x64命名機器上的added parser p95<2ms、p99<5ms是後續release gate，這次不假造measurement。

neighbor後續另立2×2：legacy/proposed boundary × existing neighbor off/on，固定同資料與models，分exact與repair結果，保留valid-original、private、ambiguous repair、no-self-learning測試。它不改本輪neighbor-off的主要結論；neural僅在選定reading後評同音字，沒有cross-reading neural比較。

## 驗證範圍

未來Phase1測試須包含：exhaustive alphabet=`a,s,u,j,0,5,3,4,6,7,A,_,.`、長度0–5（402234個字串，含空字串），每字串逐鍵檢查coverage/cap/determinism；dictionary獨立tiny lexicon與public lexicon各一輪，public輪只對到tone的prefix執行decode。另窮舉Standard所有1–3 body keys×四explicit tones驗SpellExact；用ctx空／Chinese tail／literal `3`／retained prefix、大小寫English/custom fixtures做guard parity。必須包含`-Inf`、prefix-only、two exact words、模型generation與privacy前置檢查。較長64-byte cases用固定event fixtures，不宣稱exhaustive64。

這次實際驗證：Draft202012 schema與5列範例通過；6個缺欄／非法key負例拒收；11個partition做span／literal／key→reading spot check，relative links、LF／trailing whitespace、公開樹檢查通過。isolated scratch程式直接呼叫現行decoder，tiny lexicon的single/whole/two_words/prefix raw score分別為−2/−2/−4/−Inf，符合worked vectors；另抽取來源digitsReadAsChinese與numberLike函式，在本例限定的reachability projection確認index24=1。這不是完整guard parity、模型校準或公開資料上的排名結果。schema parse不等於跑輸入實驗。未跑generator、baseline replay、fitting、held-out、neural或Windows latency；未跑與文件變更無關的整套產品測試。
