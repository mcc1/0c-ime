# Mixed-input segmentation with retained boundary alternatives

Status: EGM architecture review supports the bounded offline direction. Implementation readiness has not passed: the Phase 0 evaluator contract below must be completed before coding Phase 1. No production behavior change is approved or implemented.
Source baseline: `e754019571573051d8c29e06526802c31d5172e2`.
Related evidence: [source audit](mixed-input-segmentation-audit.md).

## Problem Statement

This input method receives physical keys in Chinese mode and infers which runs represent English, numbers or Zhuyin. Several boundaries can be legal. For example, `WQ5j04fu06` can mean literal `WQ5` followed by `萬前`, or literal `WQ` followed by `賺錢`. The current parser selects a single boundary at the first tone and inserts its English prefix as literal text. Later word evidence cannot ordinarily reconsider that boundary.

Aggressive mode currently relaxes rules in that parser. It does not preserve competing interpretations. The recent digit-initial override fixes the example but leaves the general problem. Merely finding a valid Chinese reading cannot prove Chinese intent: account names and model numbers can use the same keys.

The proposed direction is a small reconsideration window inside the current uncommitted composition. Keep several exact key interpretations through the next completed syllable, then compare whole Chinese words and literal evidence before fixing the English/Chinese boundary. Do not redesign the entire input method or ask the current neural model to classify English versus Chinese.

## Goals and Non-Goals

Goals:

- Preserve the intended English prefix and recover the first Chinese syllable after it.
- Let following Chinese word evidence distinguish alternatives such as `萬前` and `賺錢`.
- Resume Chinese after an invalid syllable without turning the rest of the token into English.
- Keep conservative mode, explicit selections, original-key restoration, privacy and existing desktop protocol behavior stable during initial rollout.
- Measure exact interpretation recall separately from ranking quality and display stability.

Non-goals for the first experiment: full sentence re-parsing, changed defaults, external inference, training a new neural model, correction of already valid readings, multiple-key edits, new physical layouts, or changing the shipped neighbor-recovery behavior. The first experiment runs offline and cannot change settings, installed binaries or personal learning.

## Current System and Reuse

[The audit](mixed-input-segmentation-audit.md) maps the ordered parser, one-reading scores, neural contract and state risks. Reuse the existing keyboard mapping and strict Zhuyin composition checks, dictionary exact/prefix lookup, Chinese decoder and N-best search, English dictionary/vocabulary, original-key candidate behavior, and existing private-state checks. Existing English weights are vocabulary evidence, not calibrated probabilities comparable with Chinese scores.

The current [syllable lattice](../core/input/input.go) cannot directly express paths with different literal lengths. Keep raw-key boundary alternatives in a separate, bounded component and call the existing Chinese decoder for each proposed Chinese suffix. Do not enlarge the generic decoder to understand English literals in the first experiment.

## Proposed Architecture

A pure interpreter takes a raw key run plus read-only dictionary/configuration inputs and returns candidate interpretations. Each interpretation partitions every consumed key into unchanged literal text and exact Chinese readings. It records the raw-key span for each output segment. Original keys remain the source of truth, even when the visible preedit shows Chinese.

Candidate generation, comparison and session mutation are separate operations. Generate exact interpretations together rather than returning from recognized-prefix, unknown-prefix or typo-recovery branches. Always retain a literal interpretation. Existing account, English-word, numeric and punctuation checks become explicit eligibility or evidence inputs for the experiment, rather than accidental priority from code order. Explicit user choices remain hard constraints, never scoring features.

The experimental run is a raw-key buffer that survives the first completed tone; it is distinct from the current `mixed.raw` token, which can be cleared at that tone. Only that run and at most two completed Chinese syllables participate in reconsideration. Before the second syllable, show a provisional first result immediately; do not add an input delay. Only completion of the second syllable may reconsider that boundary; incomplete body keys update the pending display but cannot rerank the earlier boundary. At most one boundary revision is permitted per open window. Once the two-syllable decision is finalized, the boundary becomes fixed for subsequent typing. This deliberately does not solve words requiring three or more syllables to distinguish the boundary.

The first experiment uses standard Zhuyin with explicit tone keys and end-of-composition typing only. Space, punctuation, candidate selection, cursor movement, deletion, mode changes and other editing events are tested as boundary events, but experimental live handling remains disabled until their contracts below are implemented.

## Comparing Interpretations

First distinguish three questions:

1. Are the typed keys a legal exact reading sequence with dictionary support?
2. Which exact English/Chinese split is most plausible in context?
3. Which homophones should be shown after that split?

Dictionary reachability answers the first question. The proposed statistical boundary ranker should answer the second. The existing Chinese decoder and optional neural reranker answer the third. A recognized Chinese word is evidence, not a hard instruction to convert an account.

Use the existing statistical decoder to score the complete one- or two-syllable suffix, including exact multi-syllable words and the frozen Chinese left context where available. Keep prefix-only dictionary support separate from a complete word. Do not compare raw decoder totals across different lengths as if they were boundary probabilities, and do not call neural inference separately for each reading then compare its outputs.

The offline experiment compares two rankers over the same candidate set:

- Baseline: existing aggressive decision, expressed as one candidate, including the preview-2 digit-initial override.
- Proposed: a small linear ranking score using explicit features: exact Chinese key coverage; number of Chinese syllables; complete-word versus prefix-only support; mean statistical log probability per Chinese syllable; recognized English prefix; known custom English token; numeric/identifier-pattern evidence; and a penalty for changing an already displayed boundary.

A score difference is not a probability of user intent. Coefficients and decision margins are not established by this document. Phase 0 specifies the feature and fixture contracts; Phase 1 generates and measures candidate recall; Phase 2 fits and evaluates ranking. No live rollout is allowed before the resulting coefficient table, stability margin and benchmark results are written into a follow-up implementation specification. If this experiment cannot beat the baseline without account regressions, stop rather than invent further exceptions.

The eventual conservative/aggressive option should alter the relative Chinese/literal preference and decision margin while sharing exact candidate generation. During this work the shipped conservative implementation remains unchanged as a compatibility reference. Replacing that implementation is a later, separately evaluated decision.

## Interaction and State Contracts

Raw keys belong only to the current uncommitted composition; there is no replay into an application or access to arbitrary document text. Previewing a candidate must not change personal vocabulary, update counters or call session learning. Only the chosen interpretation is materialized into existing session syllables/literals, with original keys and provenance attached.

Provisional means the system may revise the English/Chinese boundary while more keys arrive within the bounded window. Fixed means automatic boundary changes are no longer allowed; users can still edit or choose original-key candidates. Fixed does not mean committed to the application.

| Event | Required behavior before enabling live use |
| --- | --- |
| Next completed syllable | Recompute exact alternatives within the open window; preserve fixed prefix and manual choices. Finalize after the second completed syllable. |
| Enter or Commit | Commit exactly the last visible text; do not run a new ranking decision. Incomplete keys keep existing visible-phonetic versus literal behavior. |
| Down / candidate open | Freeze the displayed boundary before opening candidates. Preserve same-reading candidates and original physical keys. Alternative-boundary UI is deferred. |
| Explicit Chinese or literal selection | Freeze the chosen span; future parsing cannot reclaim its keys. |
| Space or punctuation | Finalize the displayed boundary and delegate the event to the existing handling, exactly once. Space keeps its existing first-tone role when applicable; no new space-triggered alternative search in the first version. |
| Cursor/navigation/edit events | Finalize the displayed boundary, materialize it, discard alternatives and delegate the event to existing editing logic once. Backspace then edits existing materialized input; alternative reconstruction is deferred. |
| Cancel / reset / focus loss ending composition | Discard alternatives with the composition; never carry raw keys to another field. Existing focus-loss commit behavior remains shell-owned. |
| English mode, private, paused, local blocked, unsupported layout or toneless mode | Use existing path; no experimental search or inference. If privacy changes mid-window, discard alternatives and preserve current displayed text through existing private handling. |
| Settings or model generation changes | Freeze current displayed boundary under the previous snapshot; use new settings for a subsequent window only. |

Completing the second syllable may visibly revise the previously displayed English/Chinese boundary within the open window. This may be distracting. Stability is a release criterion, not an assumed benefit. Enter-time surprise, changes outside the window, lost/duplicated keys, and changes across explicit selections are unacceptable.

## Neighbor Recovery and Neural Models

Run the first boundary experiment with neighbor recovery disabled. Exact key segmentation must be evaluated independently of typo repair. Then measure the cross-product of both features using the existing repair implementation; do not silently replace it with a new correction algorithm in the same change.

A later integration should enumerate exact paths before correction paths, retain the existing opt-in limits, and preserve the existing no-self-learning rule for automatic neighbor repairs. It needs a separate reviewed contract for ambiguity, penalties and user confirmation. This proposal does not claim that simply adding a correction penalty will calibrate intent.

The current neural scorer may improve homophones only after a Chinese reading sequence is selected. Reusing it for cross-reading or mixed literal comparison requires new evidence and is outside this plan. No network service or new neural model is needed to establish whether retaining boundaries helps.

## Alternatives Considered

| Alternative | Benefit | Reason not selected as the first implementation |
| --- | --- | --- |
| More priority rules or always take the longest legal suffix | Small change | Fails ambiguous English endings, accounts and valid shorter readings; keeps decisions irreversible. Retain as benchmark baselines. |
| Always prefer Chinese whenever a reading exists | Matches some Chinese-mode intentions | Account/model-number collisions are real; validity is not intent. |
| Neural classification before segmentation | Potential contextual signal | Current contract and calibration do not support this comparison; adds inference and data requirements before candidate recall is established. |
| One global graph over the full composition | Can express long-range ambiguity | Requires new cursor, literal, lock and ranking integration. A bounded window can first test whether the core benefit exists. |
| Bounded exact boundary alternatives | Reuses Chinese decoding and limits revision | Selected for an offline experiment; ranking and live-state feasibility remain explicit release gates. |

## Risks, Limits and Success Criteria

Candidate caps may remove the intended path; dictionary support may miss names; a two-syllable window may be too short; statistical scoring may prefer common wrong words; and aggressive intent can conflict with literal identifiers. Keep the literal lane and old decision available, measure candidate recall before top-choice accuracy, and report failures by category. Do not claim a valid reading guarantees the intended word.

Use only public or synthetic strings and isolated stores. Store no private text or user JSON in fixtures or logs. A private request must cause zero experimental inference and zero learning. Automatic speculative alternatives must never train either English or Chinese personal models.

The proposed release gate is: every safety invariant passes; no conservative-mode output changes; all mandatory regression examples pass in their declared intent; at least 99% intended-boundary recall within caps; at least 30% relative reduction in first-Chinese-boundary error rate on the same held-out Chinese-intent cases: `(baseline error rate - proposed error rate) / baseline error rate >= 0.30`; when the baseline has zero errors, the proposal must also have zero and no relative-improvement claim is made; at most 1% literal-intent conversions in the aggressive identifier set; zero harmful revisions (a previously intended boundary becomes unintended); at most 1% unnecessary within-window revisions (a changed boundary still does not match the labeled intent); and zero boundary revisions after window closure; and added parser p95 below 2 ms and p99 below 5 ms on a named Windows x64 machine. Report useful revisions (wrong boundary becomes intended) separately. Denominators for harmful and unnecessary revisions are all unambiguous replayed runs; ordinary extension is not a boundary revision. Select the stability margin on development data before measuring these held-out gates. These are proposed thresholds, not measured results. The initial dataset is a pilot: it cannot establish the percentage gates. Before any release decision, require at least 300 independent held-out literal-intent families and 300 held-out Chinese-intent families, report counts and Wilson 95% intervals, and keep family variants grouped. The 1% upper bound for literal conversion must hold for the interval as well as the point estimate. Do not count generated near-duplicates as independent evidence.

## Detailed Implementation Plan

### Phase 0 — Complete the offline evaluator contract before coding

The EGM readiness review found the following open Major items. This architecture document does not resolve them by calling the work a small refactor. The next deliverable is `design/mixed-input-evaluator-contract.md` (proposed), reviewed against the implementation-readiness standard before Phase 1 begins:

- **C1 — Event and state contract:** specify raw-run creation, first-tone retention, incomplete-second-syllable display, second-tone resolution, each finalization event, manual locks and reset. Decide the 63/64/65-key overflow transitions, counting tone bytes and completed spans explicitly. Specify a separate legacy-baseline replay adapter, raw-ownership extraction and which metrics are unavailable before a proposed ranker exists. Include expected per-event traces; hypothetical candidates must not mutate a live session.
- **C2 — Feature and dependency contract:** enumerate the exact numeric/identifier guards, arguments and immutable session-context inputs. Separate context-free spelling from contextual dictionary continuation; state whether a literal prefix resets that continuation. Fix the statistical model loader, isolated vocabulary/store, decoder options and left-context behavior across literals. Provide a versioned feature table with independent English/custom-vocabulary flags, numeric values and missing flags (including nonfinite scores), literal-lane values, and worked vectors for an exact word, two exact words and prefix-only support. The feature list in Phase 1 remains provisional until this contract is complete.
- **C3 — Fixture and leakage contract:** provide a JSONL schema and sample rows for Chinese, literal, ambiguous and selection/finalization events. Fix event objects, candidate-selection references, raw partitions, reading serialization and per-event/final assertions. Define explicit family IDs, normalization, overlapping-family grouping and deterministic initial split assignment; reject cross-split overlap, malformed spans and invalid readings. Require a revised dataset when grouping leaves insufficient independent held-out families.

Phases 2 and 3 remain separate evidence gates. Passing an architecture review does not imply this contract or a live ranker is ready. The consolidated review report records C1–C3 as open rather than resolved.

### Phase 1 — Build an offline comparison, leave product behavior unchanged

Reuse `internal/eval.Provenance`, `ProvenanceData`, `WriteReport` and `Percentile` for report metadata and timing; keep the family-based three-way split because the existing `internal/eval.Split` is a content-based two-way split. The live-engine `Typist` is not the hypothetical-candidate evaluator. Create `core/mixedparse/` (proposed) with pure generation and raw-span validation, and `cmd/0c-mixed-eval/` (proposed) for replaying public key events against a read-only dictionary and isolated stores. No new persisted setting or desktop protocol field is required. Do not call a live `Session.Handle` while evaluating hypothetical candidates. Refactor shared strict key-to-reading checks from `core/session/mixed.go` / `mixed_policy.go` into the new package only after byte-for-byte parity tests; wrappers keep current behavior.

Candidate representation: raw-key byte offsets for an ASCII run; literal spans; ordered Chinese syllables with original key spans; whether the suffix is covered by one exact dictionary word, a sequence of exact words, or only a dictionary prefix; the candidate's English/literal evidence; and a boundary identity based on key ownership. For comparing successive events, project both candidates onto the previous event's last completed-tone offset: compare the literal/Chinese ownership of those existing keys and their completed syllable spans. Ignore appended keys. Thus `WQ賺` extended to `WQ賺錢` has no boundary change, whereas `WQ5萬` changed to `WQ賺錢` changes ownership of `5`. A first event with no prior completed tone has no stability penalty. Every input byte is covered exactly once and appears in order. Separate hypothesis data from displayed text and from application commit state.

For the generator, use the existing 64-key pending-token limit. Enumerate ordered, lossless syllables ending in an explicit tone, with at most three body keys in standard Zhuyin; completed tone is included in the consumed span. A valid Chinese suffix has at most two completed syllables. Enumerate all possible starts of that suffix within the token, requiring a nonempty literal prefix when testing a mixed boundary. Whole Chinese readings and fully literal runs are separate candidates. Invalid spans may remain literal so that valid later syllables can resume Chinese. Keep exactly the typed keys for literal spans; no lowercase normalization or substitution. Dictionary prefix-only entries are marked and cannot be confused with full words.

Set an initial cap of eight exact interpretations plus one protected fully literal lane and one protected legacy-result lane. Deduplicate by raw partition and readings. For excess exact candidates, prefer complete-word support, greater Chinese key coverage, then earlier raw start and lexicographic reading order. Record intended-path pruning in evaluation. This pruning order is an experimental starting point, not established intent ranking. If recall fails the gate, change the cap/order in an explicitly reported experiment before proceeding.

The proposed feature inventory, to be made executable in C2, is: coverage = Chinese-consumed keys / run length; syllables = count / 2; full-word support = 1 only for an exact dictionary entry covering the full Chinese suffix; prefix-only = 1 only if the suffix cannot be fully covered by exact dictionary entries and needs an unfinished dictionary continuation; a fully decoded two-word phrase is not prefix-only; statistical feature = the best exact full-suffix log score / syllable count, or a separate missing-score flag when unavailable; recognized-English/custom-vocabulary = boolean exact-prefix matches; numeric/identifier = separate boolean outputs of the existing guards with their names reported; stability = 1 when the boundary identity differs from the previously displayed candidate. All other features are zero for the literal lane; use a distinct literal-lane flag. Do not fabricate an English language-model probability from dictionary weights.

Create public fixtures under `testdata/mixed-boundary/` (proposed). Each JSONL row has `id`, `intent` (`chinese`, `literal`, `ambiguous`), key events, expected raw partition/readings when unambiguous, accepted text alternatives, and source category. Include finalization events, not just final strings. An ambiguous row requires both literal and Chinese candidates to survive; exclude it from forced top-one accuracy.

Before fitting, prepare at least 100 mixed Chinese rows and 100 literal rows, plus 30 ambiguous rows. Split 60/20/20 by English-prefix and Chinese-word families, using a checked-in split assignment; variants of a family cannot cross splits. Fit on train, choose margins on development, report held-out once; a split change invalidates the previous held-out result. Do not derive all cases from the bug examples.

Required cases and coverage categories: `593`; `microsoft593fmp6`; `WQ5j04fu06`; `ru,3WQ5j04fu06`; `hello5j04fu06`; `hellosu3cl3`; `testsu.3`; `appsul4`; `useru/4`; `api2k7`; unknown/capitalized prefixes; accounts `kevin87`, `nina87`, `lisa0304`, `gpt4`, `h264`, `gpt4o`; dates, decimals, time, URLs and underscores; invalid first syllable followed by valid Chinese; original-key selection and manual locks. Do not force literal outputs as the current aggressive baseline: report its actual conversions. The `WQ`→`WD` report is an open reproducer, not a solved assertion.

Use consistent metric names: intended-boundary recall is the fraction of unambiguous Chinese-intent runs with an accepted raw partition and exact reading sequence anywhere in the candidate set; first-Chinese-boundary error rate is the fraction of those runs whose selected first Chinese key offset is wrong or absent. Report complete-partition errors separately from that first-offset metric. Also report Chinese-word output error, literal conversion count and rate on literal-intent runs, pruning count, changes per key, changes outside the window, and per-event latency by category and policy. Candidate recall and baseline metrics are available in Phase 1; proposed top-choice and display metrics remain unavailable until C1 supplies a replay policy and Phase 2 supplies the ranker. Preserve baseline rows and compare paired results; separate selection among exact reading sequences from word selection within the same reading; neighbor correction is disabled in this experiment. Public dictionary hashes and source commits go into the output header. Phase 1 ends with buildable code, parity tests, invariant tests and a candidate-recall report. No installer is produced.

### Phase 2 — Decide whether ranking is justified

Fit a deterministic pairwise linear ranker on unambiguous training rows, using the versioned feature contract from C2 and the typed/literal candidates generated in Phase 1. Use a checked-in seed and training configuration; no personal data. Compare against the legacy decision and longest-valid-suffix baselines on development data. Freeze coefficients and margin before held-out evaluation. Record exact numerical definitions and dependencies in a versioned score specification before adding runtime code. Do not change coefficients to make a single held-out regression pass.

This phase is explicitly a research gate. Coefficients, fitting-tool choice and required data expansion are not yet implementation-ready decisions. If no model meets the stated recall, accuracy, literal-conversion and stability gates, reject live integration and publish the negative result. If it passes, the output is a follow-up executable specification containing constants, deterministic tie order (keep currently displayed candidate, then legacy result, then raw-boundary order), and proposed aggressive preference. Conservative stays on its current path.

### Phase 3 — Specify and implement session integration only after Phase 2 passes

The follow-up spec must list changes to `core/session/session.go`, `mixed.go`, `mixed_policy.go`, candidate handling, `symbols.go`, snapshots/restore and learning provenance. Reuse `decode.Lock`, `mixedKeys`, `neighborKeys`, displayed-path caches and existing privacy guards. Define the window's frozen start state and output-to-raw mapping. Evaluate candidates without mutation; install only the chosen view once per key event. Do not introduce arbitrary session serialization or replay as a shortcut.

Freeze rules must match the interaction table. Verify with invariant tests and event traces: key coverage, cursor mapping in runes and UTF-16, candidate-open stability, Backspace at literal/Chinese boundaries, manual selections, cancellation, focus transitions, settings changes, simplified output, and no double processing. Explicitly exclude speculative alternatives from learning; preserve existing learning for finalized exact input and manual selection. Extend snapshot restoration for the in-memory window or finalize it before any existing restore operation; the follow-up spec must choose one before implementation starts.

Add the experiment behind a compile-time/test-only switch first. Keep the normal aggressive path as fallback and conservative path unchanged. Only after Windows native TSF tests, latency measurements and neighbor on/off interaction tests pass should a separately versioned opt-in test package replace aggressive behavior. Do not add a third settings option in this experiment; resolve the integration contract before adding a user-facing control.

### Validation and Exit Conditions

Use `go test ./...`, `go vet ./...`, the Fcitx5 tests, upgrade/release checks and public-tree checks from [CONTRIBUTING.md](../CONTRIBUTING.md). C1 must fix the generated-input alphabet and maximum exhaustive length before coding; record feasibility limits explicitly. Use invariant tests for key coverage and caps, and per-event traces covering every row of the interaction table, including editing and privacy. Existing shipped behavior is the parity baseline until a new expected output is explicitly categorized in the fixture set.

This document requests a decision about the architecture and the offline experiment. It does not authorize silently deploying a speculative ranker or claim production implementation readiness. Later phases require their own completed evidence and specification, rather than an implementer guessing unknown scoring constants.
