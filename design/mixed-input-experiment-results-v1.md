# 中英混打離線實驗 v1：執行報告

本輪結果為 **NO_GO**。候選保留的條件式召回通過，但固定排名器把中文首邊界錯誤從 40/492 增至 178/492，多餘 revision 亦超標。另有正式資料資格與整段 context 呈現缺口，不能稱為完整合格的正式實驗。主 agent 的[最後判讀](mixed-input-experiment-decision-v1.md)另列方法重評；本輪沒有接入產品或發布安裝包。

## 事前固定與執行順序

[判定標準](mixed-input-experiment-acceptance.md)在 commit `1c927821b7ae7e4863cf8a783dbffc712429167b` 登錄，SHA256 `dd42aabb1767746bf62b9d7604ac78e9300c693c7f9c6def70448b6303df5b2a`。所有門檻、seed、family grouping 與 primary 定義維持不變。

1. 原版 baseline 固定 `e754019571573051d8c29e06526802c31d5172e2`；開始時 core/internal/cmd 相對它無差異。A/C 實際重播各自獨立 Session，公開資料 Private=false；每列 disposable store，neighbor/fuzzy/neural/autocomplete 關閉。
2. evaluator／資料在 `169adcffc304d1fa1e8b2b8bc769761248638a0b` 凍結後才 enumerate。schema、語意、split、guard、observer、事件與 candidate exhaustive 檢查先跑。
3. G3 通過後，單一 fitting 程式／規格在 `1244ee6ba1287562c9c701209d2eb8058e8731e2` 凍結，才擬合 train 與跑八個預定 dev margins。
4. 固定 dev 規則選 margin=0.25；模型／選擇紀錄在 `9203c4f6aace9c7261ece948fa614fce888976d5` 凍結，才跑**一次** final ranked test。未依負結果調參、改 cap、拆 family 或加資料。

Go 1.26.10、Linux amd64、GOMAXPROCS=4、GOFLAGS=-p=4；Python 3.12.14、NumPy 2.3.5，BLAS/OMP 單執行緒。假設路徑只讀 immutable values／詞庫／模型，沒有透過 Session 試打候選。

## 資料與來源

共有 33,361 rows、3,504 transitive components：3,010 Chinese、30,201 literal、150 ambiguous。起始採樣預算先固定為 3,000 lexical Chinese、3,000 synthetic literal、150 ambiguous，再加 mandatory/state cases。公開來源取得明確止於 USB+PCI，沒有逐次補樣本直到 CI 過關。

| split | Chinese groups | literal groups | ambiguous groups |
| --- | ---: | ---: | ---: |
| train | 1498 | 533 | 88 |
| dev | 501 | 167 | 23 |
| test | 492 | 164 | 39 |

Synthetic 中文意圖從公開詞庫 reading 轉寫 physical keys；不是從模型答案造標籤，也不是人類輸入分布。USB 9,400 rows、PCI 17,793 rows 保留原始 ASCII token，vendor/manufacturer／共享 token 以 transitive grouping 合併，不把同廠商型號當獨立觀測。來源、授權與固定規則見 [protocol](../testdata/mixed-boundary/protocol.json)及 [sources](../testdata/mixed-boundary/sources)。USB commit `5eb613762db1e34b9218da63a88f22dd26f743c5`；PCI commit `19e7dfd4f141b895c36a50ba500ea96470065e27`。

Pilot 分組數量合格；正式 literal test 僅164，未達400。部分分類亦不足20個 primary groups：digit-initial 11、context 1；literal primary 大多為型號，日期、URL、底線等不能據此宣稱有獨立分類證據。品質前改名的266個 pluralized-prefix rows 仍保留原 raw，其中英文詞籍未驗證；凍結的 known-prefix 分類包含這項限制，沒有事後重新分類。

## 候選、擬合與正式主分母

Phase1 主召回為 test unambiguous primary groups：**492/492** capped strict paths 存活，mandatory Chinese **10/10**，ambiguous **150/150** literal/Chinese 雙解存活。uncapped 相同，無 pruning loss。原始自動摘要誤用 all-row 分母，已另存透明修正報告；原檔沒有覆寫。

All-row 診斷為3009/3010。保留 dev 反例 `QQQ3visionsnj04ru7`（算計）：失敗 tone 先耗一次 opportunity，first window 關閉時 baseline 凍結「萬」而非「算」，後續「ㄐㄧ˙」不能接續。這是 baseline 歷史下的條件式召回，不證明所有政策歷史皆可召回。另從本次 ranked causal candidates 重算，test primary 仍為492/492。

固定估計器是26維 pairwise logistic linear ranker，feature12係數固定0，不做 scaling/imputation/clipping；zero init、200 full-batch steps、learning rate0.05、L2=0.001。Train primary 中1498 Chinese、85 literal 有競爭 tone/pairs，共8871 pairs；4列因 ownership 不可用整列排除，missing-gold events=0；無競爭 tone 的 literal rows 仍保留評測分母。只有此一估計器，dev margins 固定 `[0,.1,.25,.5,1,2,4,8]`。

Dev 按 literal、harmful、Chinese-boundary、unnecessary incidence，再 margin 小者的 lexicographic 規則選0.25。其 Chinese error246/501、literal1/167、harmful0、unnecessary5/668；margin0雖中文較好，harmful=3，沒有事後改選。Train loss由0.693147降至0.353949，但 gradient L2仍0.09167；固定預算沒有證明已收斂，也沒有排除其他排名方法。

## Final test 與 gates

主品質各 group 固定一個 primary row；任一 terminal window 失敗即該 row 失敗。

| policy | Chinese 首邊界錯誤 | literal any-event conversion |
| --- | ---: | ---: |
| A aggressive preview2 | 40/492，8.13% | 主 paired metric unavailable；mapped-only 9/161 |
| C conservative | 106/492，21.54% | 主 paired metric unavailable；mapped-only 4/161 |
| causal longest-valid-suffix | 24/492，4.88% | 23/164，14.02% |
| 固定 ranker E | **178/492，36.18%** | **1/164，0.61%**；Wilson95上限 **3.37%** |

A/E 中文配對中146組原本正確變錯，僅8組原錯被修正；relative reduction=`(40-178)/40=-3.45`，沒有達到+30%。A/C 三個 literal primary ownership unsupported，不能以161個 mapped rows 冒充完整 paired 比較。舊 Phase1 longest 的 terminal/不完整候選數字亦不作正式 literal 比較，另以自身因果歷史重播未擬合的 longest 對照。

E primary 的 useful revision72/656、harmful0/656、unnecessary **10/656=1.524%**；82個 ownership revisions、primary fixed fallback1。全部26,768 ranked rows（含 variants/ambiguous）記錄185,070 raw keys，run 內 outside/key/commit failures皆0；全 rows revisions133、fallbacks91。這些分母不能混用。

| gate | 狀態 | 依據 |
| --- | --- | --- |
| G1 evaluator | unavailable | 已執行子項通過，但 context 整段呈現／提交仍有缺口 |
| G2 privacy／副作用 | pass（已測範圍） | protected generate/score=0；hypothetical learning/persistence=0，stores digest不變 |
| G3 recall | pass（上述固定歷史範圍） | 492/492、mandatory10/10、ambiguous150/150 |
| G4 中文邊界 | **fail** | 40→178 errors；相對下降−345% |
| G5 literal | **fail** | 點估0.61%，Wilson95上限3.37%>1%；正式樣本資格亦不足 |
| G6 stability | **fail** | unnecessary1.524%>1%；harmful及窗口外皆0 |
| G7 不可破壞條件 | unavailable | run 內檢查通過不能替代完整 prelude/commit 驗證 |
| G8 證據 | unavailable | literal164<400、分類及必要 paired ownership缺口 |

G4/G6 是可用的負證據；G1/G7/G8 是工具／資料資格不足。G5不能支持預定上限，不等於證明真實誤轉率必然大於1%。缺口不會把已測排名退步改稱通過。

## 失敗類型與實例

E 在凍結 declared categories 的錯誤：unknown-prefix49/60、invalid-first-resumption33/33、direct44/77、known-prefix40/193、capitalized6/97、digit-initial5/11、English-tail1/20、context0/1。完整 rows 診斷為 Chinese218/896錯、worst-case group179/492；literal189/25833誤轉，集中在同一 transitive group。primary 與 variants 均保留。

| raw keys | A 實際 commit | E 記錄的 run commit |
| --- | --- | --- |
| `593` | 窄 | `593` |
| `WQ5j04fu06` | WQ賺錢 | WQ賺錢 |
| `284xo6` | 大雷 | `284xo6` |
| `QZXdropkicksw.6fu4` | QZXdropkicks投契 | `QZXdropkicksw.6fu4` |
| `QQQ3vaguelys84hjo4` | QQQ3vaguely納粹 | QQQ3vaguely那h為 |
| `ME-6100/4/DIO`，literal | ME而100/4/DIO | ME而100/4/DIO |

最後一列是公開 PCI `1402:6144` 的原 token，也是錯誤 literal primary。合法中文 reading／詞籍證據不足以判別識別字意圖。直接中文及未知前綴常被保留 literal；failed-tone opportunity又縮短恢復窗口。這些現象支持重評排名目標與窗口，不能藉單一案例補例外。

**Context 呈現缺口：** prelude=`ru,3`、run=`WQ5j04fu06` 時，A commit為「解WQ賺錢」，E工具目前只記「WQ賺錢」。E 的 frozen context用於查詢，卻未把prelude visible接入完整composition提交。表中的 E 是 run-scoped 輸出，不冒充整段提交通過；沒有修改凍結 source或重跑品質把它洗掉。

## 驗證、重現與 evidence

已執行：3150個guard vectors（每個13 guards）、424事件observer on/off snapshot/store parity、prefix-only/join、custom/commit、selection、privacy/reset、所有finalizers與63/64/65-byte fixtures。Tiny/public各402234短字串、1977651逐鍵prefix、608508 tone-prefix、208236 Standard spelling roundtrips，candidate invariants零失敗。另在training後補full-state exhaustive：每輪3955302 Machine steps，tiny/public皆零失敗；沒有把這項晚補驗說成訓練前已完成。

`go test ./...`、`go vet ./...`、tagged全套test/vet、Fcitx5 test、upgrade/release checks、public-tree皆通過。沒有Windows TSF/UI/latency、native materialization、live window discovery或neighbor2×2成績。Primary test字面gold不足，surface/homophone error為null、count0，不能填成零錯誤。

精簡機讀結果見 [decision-v1.json](mixed-input-experiment-decision-v1.json)。大型原始候選／事件／rows及完整修正報告在 `/workspace/scratch/mixed-input-experiment-run/`：`enumerate-v1/`、`dev-v1/`、`test-v1/`、`phase1-corrected-report.json`、`final-report.json`、`examples-v1.json`、`artifact-index.json`。主 agent另獨立核對generation、重現相同模型／dev選擇、從最終raw rows重算；未挑另一個ranked test結果。

關鍵SHA256：

- fixtures：`53056950eeb2ed9330fc28cd587038c0d2ee3ea25017d95c8b9afa7e15fb6631`
- assignment：`1352d7a3c59b1dbf378730473f1db3daa96feb91e6638d88eb77fd3840649c3d`
- Chinese manifest：`3e93ed1ef3d2fc7443f64a0b5203294298181b84bed2cd6a144c48a4852aefee`
- English manifest：`83427160c73b81c4f17185c87b974bc159f115881c1aa22b6d5fa3c0060c060a`
- selected model：`f347d8ef562c3b830924693d378de5eab35ae1f53740781f5834d0493db91b21`
- final ranked rows：`2f4fe7d09971fa7e6c6b97c7e0b2169619927c874e0e3967890d6a16eba1e45c`

全部source/data hashes在 [Phase1 lock](../testdata/mixed-boundary/run-lock.json)及 [fitting lock](../testdata/mixed-boundary/fitting-run-lock.json)。[報表工具](../internal/mixedeval/reporttools/README.md)保留修正分母、例子、fitting診斷及finalizer原始碼；[full-state probe](../internal/mixedeval/probes/fullstate/main.go)與[causal-longest probe](../internal/mixedeval/probes/longest/main.go)均可重跑。artifact index記原始與版控source hash。

```sh
. /workspace/.0c-tools/activate.sh
umask 022
go test -tags=mixedeval ./core/mixedparse ./internal/mixedeval ./core/session
go run -tags=mixedeval ./cmd/0c-mixed-eval -phase validate
go run -tags=mixedeval ./cmd/0c-mixed-eval -phase enumerate \
  -lock testdata/mixed-boundary/fitting-run-lock.json -split all -out /tmp/new-mixed-enumeration
go run -tags=mixedeval ./internal/mixedeval/probes/fullstate
go run -tags=mixedeval ./internal/mixedeval/probes/longest /path/to/enumerate-v1 /tmp/new-longest
```

本輪held-out已被查看。以上供重現既有結果；新方法不可沿用這批test作未見測試。
