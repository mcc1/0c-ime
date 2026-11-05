# Claude 接手入口：中英混打與 Shift 回中文實驗

更新：2026-11-05。Fork：<https://github.com/mcc1/0c-ime>；上游：<https://github.com/0c-biggo/0c-ime>。
請先 checkout `research/experiment-records`，讀本文件與 `design/shift-return-acquisition-decision-v1.md`。
目前工作沒有背景 agent 或品質實驗繼續執行，也沒有等待中的安裝包／正式模型。

## 分支慣例（使用者 2026-11-05 指定）

- **一個 PR 只放一個功能或一個研究主題**，不要混合。順手發現的其他修正另開分支；上游也有的 bug 從上游 `main` 開分支、送上游 PR。
- 需要混合多個功能的安裝包時，另開 build 分支（例如 `test-builds/*`）把那些功能 merge 進來再打包，不回頭改功能分支。
- 不 force-push、不改寫既有歷史；搬移內容用新 commit 加 merge。

## 分支與 PR 範圍

更新：2026-11-05。產品功能分支以 stacked PR 使用；實驗分支不開 PR。

### 產品功能（fork）

| 分支 | 建議 base | 內容 |
| --- | --- | --- |
| `feat/mixed-input-aggressive`（原 `fix/mixed-digit-rarity`） | fork `main` | 保守／積極混打選項的完整功能：中文優先、打錯一鍵後恢復、英文後數字聲母（WQ賺錢）、學過的英數詞與大寫邊界（wq賺了、V了）、`mixedAbsorbNonLetters`、設定頁「混打組字傾向」子選項、`design/aggressive-mixed-boundary.md`。已合併上游 0.2.8。仍含 593 學習修正與 Fcitx 5.0 修正（已另送上游 #55、#54，上游合併並同步 fork `main` 後會從 diff 消失） |
| `feat/keyboard-neighbor-recovery` | `feat/mixed-input-aggressive` | 鄰鍵容錯（獨立 opt-in）、設定頁「鄰鍵容錯」子選項、積極模式測試的鄰鍵開／關變體 |
| `fix/settings-refresh-position` | `feat/keyboard-neighbor-recovery` | 只剩設定頁捲動修正，與上游 #53 相同；上游合併後可移除 |
| `fix/mixed-digit-initial-boundary` | — | 內容已併入 `feat/mixed-input-aggressive`，無自己的差異，可刪除 |

### 已送上游（各自從上游 `main` 開出）

| 分支 | 上游 PR | 內容 |
| --- | --- | --- |
| `fix/settings-scroll-position` | 0c-biggo/0c-ime#53 | 背景更新與切回視窗時保留捲動位置 |
| `fix/fcitx5-0-compat` | 0c-biggo/0c-ime#54 | Ubuntu 22.04（Fcitx 5.0）編譯 |
| `fix/mixed-digit-learning` | 0c-biggo/0c-ime#55 | 學了其他字之後 593 不再變數字（保守模式） |

### 實驗（不開 PR）

| 分支 | 內容／停點 |
| --- | --- |
| `experiment/mixed-boundary-v1` | 舊全域排名器、凍結資料與負結果；`2820128`，NO_GO |
| `design/shift-return-v2` | 下一輪設計、EGM 修正與交換契約；`c512578` |
| `experiment/shift-return-v2` | P0–P3 隔離工具、pre-quality errata；`18d4f77` |
| `research/experiment-records` | 本交接、實際取得資料、標註、資格失敗與完整歷史證據封存 |

實驗分支已 merge 上游 0.2.8，但沒有跟進 2026-11-05 的產品整併；做 acquisition r2 前先 merge `feat/keyboard-neighbor-recovery`。
`design/mixed-input-segmentation-plan` 仍保留在 `18d4f77` 作原工作分支。Fork 另有 `test-builds/*` 測試版分支，不改寫它們。

## 使用者核心目標與範圍

- 中文模式下希望完整讀音／成詞線索能積極恢復中文，尤其英文或一次誤按之後，並保留帳號、型號、數字等原作者規則。
- 使用者認為積極與保守都有道理，因此已有獨立選項，不應整批刪掉保護規則。
- 鄰鍵修正是另一個 PR／選項，依設定 layout；本 Shift 實驗不啟用它。
- 本輪只研究：使用者先 Shift 明示英文起點，後文是否能協助判定何處自動回中文，並回修**未提交**原鍵。不是自動找英文起點，不改一般中文 decoder，不改已送 application 的文字。
- 不應只在當下第一個音節永久定案；長度、詞證據、後文回修的組合及順序是主要問題。

已見 mandatory 例：`593`→窄；`microsoft593fmp6` 意圖 microsoft窄裙；真實 prelude `ru,3`→解，再 Shift 輸入 `WQ5j04fu06` 意圖解WQ賺錢。`解WD5萬前` 是原錯誤表面，不是 gold。
無 Shift 的 `593` 只作原 Session 相容對照，不能當自動 Shift 回切成功。

## 已有結論：不要混淆兩輪

1. **舊全域 mixed-boundary v1：NO_GO。** 中文 primary 錯誤 40/492 → 178/492；不要接進產品。完整結論在 `design/mixed-input-experiment-decision-v1.md`／`mixed-input-experiment-results-v1.md`。其 test 已見，只能回歸，不可宣稱新 held-out。
2. **Shift-return v2：方法品質 NOT_RUN；資料取得提前停止，G0 未通過。** 不是積極組字／後文方法已被否定。

本輪實際建立且兩個真實 AI 身份標註的任務有 1,500 筆（750 return、750 literal）。原先 8 筆語境分類錯誤，修正描述並重新雙標後納入；原拒收／新舊 ID 保留，同一預算不算 1,508。
相依 family／語意近重複／固定路徑與版本 payload 模板合併後：372 components，train 209、dev 74、test 69、regression 20。兩 intent 共用 group，不可相加當 n。
Test context-contrast 只有 69，低於 100；test return/literal 各 69，低於各 400；其他固定長度／類別配額未齊。直接建置 CLI 的 validate exit **3**，紀錄在可讀資料目錄。

停止原因是原固定 750 contrast 配置被真實相依性大幅合併；繼續大量填 none-context 的一般題不會按原意補足這個隊列。這是取得階段的提前停止，**不是**已耗盡 6,000 預算，也不是證明任意合法新資料都不可能通過。
未建立正式 run-lock，未跑品質 enumerate、正式 fit、dev 選模、final test；沒有模型品質分數或安裝版可發布。

## 檔案與還原

- 可直接閱讀：`testdata/shift-return-acquisition-v1/`，含 exact-byte 最終 bank、fixtures、assignment、sources、quota、報告與 validate 結果。
- 完整資料／兩個 annotator 的逐筆證據、作者程式、拒收、讀音修訂、alias、所有中間 closure、P0–P3、失敗 log／修正來源：`testdata/experiment-archive-20261102/` 的 `shift-return-records`。
- 舊負結果的完整 raw events/candidates、fit/dev/test、獨立重核：`mixed-boundary-records`。
- EGM 歷史 review／re-audit／snapshots：`review-records`。
- 當時唯讀公開 Chinese/English/LM pack／manifest／notices：`public-runtime-data`。授權與來源條款隨檔保留。

各 gzip 超過 40 MiB 時有序分卷；`archive-index.json` 記每卷、合併包及每個原檔 SHA-256。不是新的 formal lock。

```sh
python3 testdata/experiment-archive-20261102/restore.py --verify-only
python3 testdata/experiment-archive-20261102/restore.py --destination /tmp/ime-records
```

歷史 root 是 `/workspace`，還原後包含 `scratch/...` 與 `0c-ime/assets/data/...`。
要恢復原 layout 才用 `--destination /workspace`；helper 遇到不同現存 bytes 或 symlink 會拒絕覆寫。
原 Sources／locks 的絕對路徑與 hashes 保留；不要偷偷改舊 lock 讓換環境的檢查過關，需另立 relocation metadata。
只省略本機指令檔／私有檔名與可重建 cache／編譯執行檔，詳見 archive index omissions；沒有個人 user.json 或憑證。原本缺失的歷史來源也不會憑空補造。

## 工具、驗證與限制

原環境：Go 1.26.10、Python 3.12.14、NumPy 2.3.5、jsonschema 4.26.0；GOMAXPROCS=4、GOFLAGS=-p=4。
系統名為 go 的程式可能不是 Go compiler，必須確認 `go version`。原工具路徑 `/workspace/.0c-tools/go/bin/go`；實驗 cache 與一般 build cache 分開。

```sh
export PATH=/workspace/.0c-tools/go/bin:$PATH
export GOPATH=/workspace/.0c-tools/gopath
export GOCACHE=/workspace/.0c-tools/shift-return-go-cache
export GOMAXPROCS=4 GOFLAGS=-p=4
umask 022
go test -count=1 -tags=shiftreturneval ./internal/shiftreturn ./cmd/0c-shift-eval
go vet -tags=shiftreturneval ./internal/shiftreturn ./cmd/0c-shift-eval
python3 -m unittest discover -s internal/shiftreturn -p 'test_*.py'
python3 internal/shiftreturn/fit.py verify-numerical design/shift-return-v2/numerical.json
python3 design/shift-return-v2/check_examples.py
python3 scripts/check_public_tree.py
```

P0–P3／normal tests/vet 的歷史成功 logs 已封存；主 agent 另獨立核對。不要把 tests 或 toy fits 當品質實驗。
`-count=1` 可避免 Go cache 忽略被呼叫 Python 檔變動。`SHIFTRETURN_DATA_INVENTORY=1` 會跑固定 public parity／20-case 操作校準，不是新樣本品質。
工具全需 `shiftreturneval` tag，沒有產品 Session 接線；舊 v1 的部分 production observer/helper 是既存隔離評測用接口，不能說整個實驗歷史都零 source hook。

已知缺口：

- 398 份歷史 seen-input 來源中僅 397 有 exact copy；舊 pre-errata `report_test.go` hash `cb17f3c3baad0dddf95c8b783c3f2488359fb1c1c3fe0f4360eea33289bfc976` 尚無原 bytes。保留 conservative seen 與 hash，不能宣稱 provenance 全齊。
- 固定 `x`+`593`×64 校準會耗盡 250,000 logical lex requests（105 次 generation、3,571 次 Best）；保留原鍵並記 exhausted。17 音節對照完成。不能藉刪長題或調上限藏掉此負例。
- Formal preflight free≥10 GiB，aggregate raw evidence≤8 GiB；每次正式階段要重查。Reducers 目前在記憶體保留解碼 evidence，壓縮大小不能代替 RAM 可行性。
- 原文件誤把 `v3` 當 invalid；實際 SpellExact 接受 ㄒˇ。保留反例，invalid worked case 改 `vv3`，不加 production phonotactics guard。這是品質前明示 erratum。

## 不可偷偷改的實驗規則

主條件 EC-revise；E0=B、EL=B+L、EW=B+W、EC=B+L+W 各別擬合，同一資料／程序，各跑 fixed/revise，共八 final 條件。
不因 test 選較好 arm；B 不偷看未來；runtime Query/Step 不看 gold；training labels 由外部 builder 加入。
維持正常 decoder、原 Source 的 Layered/Long 與 neutral 語意；不加入 fuzzy／鄰鍵／神經／autocomplete。
Context 效益需反映 boundary/readings，不能只報同音字換字。

來源／資料先 freeze → G1 enumerate 合格 → fit/train-only → dev 固定 margin grid → numeric freeze → 八條件 test 一次 → 獨立 reduce。
Initial immutable source lock → fit lock → execution/evidence locks，不改舊檔。Selected manifest 恰一個 `selected-model-manifest.json`，四個明示 absolute path/hash，保持祖先 source/fit hashes。
未完成／unsupported 保留完整 n，明示 unavailable、null rates/CIs；不得把缺資料當 0 error 或刪困難列。詳細門檻與數值以 protocol/schema 為準。

## Claude 下一步

先讀決策與取得報告，重核 archive。**下一個方法問題是資料取得／相依單位，不是立刻換模型或調權重。**
先另寫 pre-quality acquisition r2：從實際 source/vendor/content/template families 建來源 sampling frame，取得足夠多樣的真實 literal 延續或明示合成意圖，先檢查結構與供給再投入大量 authorship。
若要改 contrast 配置，先明示新 revision 與 budget；保留此次結果，維持原品質門檻，禁止換 seed、拆真 family 或依已知 test membership 補題。新的真實獨立雙標與資料／來源凍結通過後才跑模型。
不要將 v1 或本輪已看過的 test 任務回收成 unseen；不要將 G0 資料失敗說成方法失敗或已成功。
本次只完成封存與 fork push，沒有啟動 acquisition r2。

## EGM 與 code-review workflow

使用者先前要求兩種 workflow 的環境安裝。還原 installer 與 pins 在 archive 的 `setup/` 目錄：

```sh
bash testdata/experiment-archive-20261102/setup/install-review-skills.sh
```

它還原到 `/workspace/.0c-tools/review-skills`，保留完整 EGM 相對引用與來源 license/hash；code-review 是 Codex adapter，不能冒稱已在 Claude 原生 plugin picker 註冊。
EGM pinned `403d8881d62022490637d740657b33f01e321ea8`；官方 code-review source pinned `a7ba5b93db0c7c8d17f3f1d25853adca79598a1a`。
需要 fresh-context Goldfish reviewer，不要由父 context 假扮；沒有 native agents 就標 review 不完整。
截至 c512578 的設計曾完成五階段／九個有效 reviewer 及 focused re-audit；其後 errata 有測試與父審，但沒有冒稱另跑完整 EGM。
安裝檔存在、plugin 登錄、實際成功 review 是不同狀態。使用 workflow 不等於有權發 GitHub 評論或對外訊息。
