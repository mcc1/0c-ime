# Mixed-input segmentation audit

Status: source audit and existing-test reproduction, 2026-11-01. No input behavior changed by this audit.
Baseline: `e754019571573051d8c29e06526802c31d5172e2` (neighbor preview 2).
Previous neighbor preview: `45701f861d849071bffc94dc2604cd4f63c23731`.

## Findings

### A1. The parser chooses one boundary before word decoding

In [finishMixedTone](../core/session/mixed.go), a completed tone triggers an ordered list of returns: explicit English vocabulary protection; space/punctuation/number handling; aggressive whole-reading acceptance; number/rarity protections; whole-reading acceptance; English-word protection; neighbor repair; then mixed-boundary inference. Success inserts literal English and one Chinese reading into the session. Subsequent Chinese decoding sees that reading, not the discarded alternative boundaries.

[mixedAggressiveBoundary](../core/session/mixed_policy.go) searches recognized English/number prefixes first, returns immediately if it finds one, then tries recovery after a previous failed tone, then unknown English prefixes. These groups do not compete together. Within each group it reduces possible boundaries to one result.

Impact: later word evidence can change homophones on the selected Chinese path but generally cannot reclaim a key already materialized as an English literal. There are narrow existing exceptions such as `reviseLetterI`; it is inaccurate to say the code never revises any earlier interpretation.

### A2. Boundary scores are isolated statistical reading scores

[mixedSyllableScore](../core/session/mixed.go) constructs a decoder with the personal statistical model, decodes one syllable with prefix support from default sentence-start context, and exponentiates its score. This is more than raw character frequency: dictionary prefix mass and personalization can matter. It is still not a joint English/Chinese boundary probability, current left-context word score, or neural score.

Known-prefix alternatives use a 10x move ratio, except an English suffix `s` uses 0.25. Unknown-prefix alternatives use 0.5. Conservative mode additionally uses rarity and unknown-word floors. Aggressive mode removes some floors, accepts exact whole readings earlier and expands eligible unknown prefixes, but retains group ordering and score-ratio selection. It is not a joint decoder with a different Chinese prior.

Impact: a common shorter reading can leave a valid initial key attached to English. The last fix adds `mixedDigitInitialExtension` to override this for one digit-key initial. That fixes a class of examples but adds another precedence rule; it does not address all boundary ambiguity.

### A3. Existing neural reranking cannot decide English versus Chinese directly

[rerankPath](../core/session/rerank.go) runs after `decodeWithLiterals`, independently on Chinese spans separated by literals. [rerank.Request and Supported](../core/rerank/rerank.go) use one reading sequence and a bounded candidate list. Ordinary requests require 2–32 syllables and candidate text of the same length; single-syllable text mode requires configured text weight and confirmed context. Existing neutral-tone expansion is a controlled exception within the Chinese span.

Impact: separate requests for `ㄨㄢˋ ㄑㄧㄢˊ` and `ㄓㄨㄢˋ ㄑㄧㄢˊ` do not establish calibrated comparable scores. The existing contract does not compare `WQ5萬前`, `WQ賺錢`, and raw Latin text as three interchangeable candidates. Changing this requires a distinct model contract, data and evaluation.

### A4. Neighbor repair is another early branch

[neighborChoices and acceptNeighbor](../core/session/neighbor_recovery.go) preserve whole valid readings, but repair is attempted before `mixedBoundary`. A unique repair can be accepted before an exact boundary after an unknown prefix is considered. Ambiguous repairs remain literal until selection. The code bounds substitutions and protects privacy, layout, tone and learning provenance.

Impact: a replacement must distinguish exact-boundary evidence from correction evidence. Reusing the adjacency geometry is reasonable; using branch order as the ranking decision is not sufficient.

### A5. State and interaction are as important as ranking

[Session](../core/session/session.go) has syllable-indexed literals, locks, displayed-path caches, cursor state, snapshots, learning provenance and confirmed document context. [mixed.go](../core/session/mixed.go) retains original keys and has display-preserving flush behavior. Commit uses displayed text and must not silently select a different hypothesis. Candidate selection and explicit literal selection are constraints. A parser that replays keystrokes through a live session risks duplicate learning and side effects.

Impact: retaining alternative boundaries needs raw-key span ownership and a defined finalization contract. Adding alternatives to `input.Lattice` alone is insufficient: [that lattice](../core/input/input.go) assumes a fixed sequence of syllable positions and does not represent arbitrary literal spans or variable key consumption.

## Reproduction

All checks use isolated in-memory personal stores and the verified public dictionary at `assets/data`; no daily user store is read or changed. The archive of preview 1 was extracted outside the working tree and given only the current regression test.

| Keys | Preview 1, aggressive, neighbor off | Preview 2, aggressive, neighbor off/on |
| --- | --- | --- |
| `WQ5j04fu06` | `WQ5萬前` | `WQ賺錢` |
| `ru,3WQ5j04fu06` | Same boundary defect follows from the same raw-token path; not separately asserted by the first-failure test run | `解WQ賺錢` |
| `hello5j04fu06` | Prior diagnostic produced `hello5萬前` | `hello賺錢` |

The fresh preview-1 regression exits at the first `WQ5萬前` failure, as expected. The preview-2 public-dictionary test passes five examples with neighbor repair both off and on. Existing policy, English-boundary, account-preservation and typo-resumption tests pass at preview 2. Source tests deliberately include aggressive conversions such as `gpt4`; their passing does not establish good product behavior for accounts.

The reported `WQ` becoming `WD` has not been reproduced. Neither this audit nor the numeric-initial fix establishes its cause.

Commands:

```sh
. /workspace/.0c-tools/activate.sh
umask 022
ZC_TEST_PUBLIC_DICTIONARY=/workspace/0c-ime/assets/data go test ./internal/datadir \
  -run 'TestMixedDigitInitialWithPublishedDictionary|TestNeighborRecoveryWithPublishedDictionary' -v
go test ./core/session \
  -run 'TestMixedPolicies|TestMixedAggressiveKeepsDigitInitialAfterEnglish|TestMixedEnglishThenZhuyinWithoutSpace|TestMixedUnknownEnglishStaysLiteral|TestMixedTypoDoesNotSwallowFollowingZhuyin' -count=1
```

## Evidence limits

These tests verify specific behavior and compatibility. They do not measure boundary recall, unintended conversion rate, display instability, performance on Windows, or ranking quality over unseen examples. Public-dictionary integration does not by itself execute or calibrate neural inference. More patch-specific passing tests cannot substitute for a held-out boundary evaluation.
