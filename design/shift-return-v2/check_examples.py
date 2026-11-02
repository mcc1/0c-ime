#!/usr/bin/env python3
"""Check design schemas, raw ownership and analytic arithmetic; no IME scoring."""
import copy
import hashlib
import json
import math
import random
import unicodedata
from pathlib import Path

from jsonschema import Draft202012Validator

ROOT = Path(__file__).resolve().parent
SCHEMA = json.loads((ROOT / "schema.json").read_text())
Draft202012Validator.check_schema(SCHEMA)
VALIDATOR = Draft202012Validator(SCHEMA)
KEYS = dict(pair for pair in (
    "1ㄅ", "qㄆ", "aㄇ", "zㄈ", "2ㄉ", "wㄊ", "sㄋ", "xㄌ", "eㄍ", "dㄎ", "cㄏ",
    "rㄐ", "fㄑ", "vㄒ", "5ㄓ", "tㄔ", "gㄕ", "bㄖ", "yㄗ", "hㄘ", "nㄙ",
    "uㄧ", "jㄨ", "mㄩ", "8ㄚ", "iㄛ", "kㄜ", ",ㄝ", "9ㄞ", "oㄟ", "lㄠ", ".ㄡ",
    "0ㄢ", "pㄣ", ";ㄤ", "/ㄥ", "-ㄦ", " ˉ", "6ˊ", "3ˇ", "4ˋ", "7˙",
))


def fixture_semantics(f):
    VALIDATOR.validate(f)
    raw = "".join(e["key"] for e in f["events"] if e["type"] == "key")
    assert raw == f["expected_raw"]
    assert f["events"][0]["type"] == "shift"
    assert f["events"][-1]["type"] == "enter"
    for gold in f["gold"]:
        end = 0
        for seg in gold["segments"]:
            assert seg["begin"] == end < seg["end"] <= len(raw)
            end = seg["end"]
            if seg["kind"] == "chinese":
                body = raw[seg["begin"]:end]
                mapped = "".join(KEYS[k] for k in body)
                if mapped.endswith("ˉ"):
                    mapped = mapped[:-1]
                assert mapped == seg["reading"], (body, mapped, seg["reading"])
            else:
                assert seg["reading"] == ""
        assert end == len(raw)


def close(a, b):
    assert math.isclose(a, b, rel_tol=0, abs_tol=1e-12), (a, b)


def canonical(value):
    if isinstance(value, float):
        return 0.0 if value == 0 else value
    if isinstance(value, list):
        return [canonical(x) for x in value]
    if isinstance(value, dict):
        return {k: canonical(v) for k, v in value.items()}
    return value


def encoded(value):
    return json.dumps(canonical(value), sort_keys=True, separators=(",", ":"),
                      ensure_ascii=False, allow_nan=False).encode()


def digest(value):
    return hashlib.sha256(encoded(value)).hexdigest()


def main():
    fixtures = [json.loads(x) for x in (ROOT / "examples.jsonl").read_text().splitlines()]
    assert len({f["id"] for f in fixtures}) == len(fixtures)
    for f in fixtures:
        fixture_semantics(f)
    # Exercise structural and semantic rejection, not only positive JSON parsing.
    invalid = copy.deepcopy(fixtures[0])
    del invalid["events"][1]["key"]
    assert not VALIDATOR.is_valid(invalid)
    invalid = copy.deepcopy(fixtures[0])
    invalid["gold"][0]["segments"][1]["begin"] += 1
    try:
        fixture_semantics(invalid)
    except AssertionError:
        pass
    else:
        raise AssertionError("ownership gap was accepted")
    invalid = copy.deepcopy(fixtures[0])
    invalid["gold"][0]["segments"][1]["reading"] = "ㄨㄢˋ"
    try:
        fixture_semantics(invalid)
    except AssertionError:
        pass
    else:
        raise AssertionError("wrong raw-to-reading label was accepted")

    n = json.loads((ROOT / "numerical.json").read_text())
    xs = n["population"]
    mean = sum(xs) / len(xs)
    std = math.sqrt(sum((x - mean) ** 2 for x in xs) / len(xs))
    close(mean, n["mean"])
    close(std, n["std"])
    for x, expected in zip(xs, n["standardized"]):
        close((x - mean) / std, expected)
    close(n["initial_loss"], math.log(2))
    close(n["initial_gradient"], -n["pair_difference"] / 2)
    close(n["after_one_update"], -n["lr"] * n["initial_gradient"])
    w, d, lam = n["probe_w"], n["pair_difference"], n["lambda"]
    close(n["probe_loss"], math.log1p(math.exp(-w*d)) + lam*w*w/2)
    close(n["probe_gradient"], -d/(1+math.exp(w*d)) + lam*w)
    z = 1.959963984540054
    close(n["zero_error_wilson_upper"], z*z/(400+z*z))
    assert n["zero_error_wilson_upper"] < .01

    errata = json.loads((ROOT / "errata-examples.json").read_text())
    assert errata['observed_api_case']['n'] == 2 and errata['invalid_body_case']['raw'] == 'x593vv3'
    assert errata['temporal_case']['first_correct'] == 3
    metric_validator = Draft202012Validator(SCHEMA['$defs']['metric'])
    metric_validator.validate(errata['unavailable_metric'])
    wrong = copy.deepcopy(errata['unavailable_metric']); wrong['value'] = .4
    assert not metric_validator.is_valid(wrong)
    manifest = json.loads((ROOT / "selected-model-manifest-example.json").read_text())
    VALIDATOR.validate(manifest)

    report = json.loads((ROOT / "worked-report.json").read_text())
    VALIDATOR.validate(report)
    assert len(report["conditions"]) == 8
    for metrics in report['conditions'].values():
        for metric in metrics.values():
            assert metric['errors'] == metric['observed_errors'] + metric['unavailable_components']
            assert metric['status'] == ('complete' if metric['n'] else 'unavailable')

    ec = report["conditions"]["EC-revise"]
    assert ec["return_boundary"]["errors"] == 0
    assert ec["literal_only"]["errors"] == 1
    assert ec["literal_only"]["wilson95"][1] > .01
    assert report["outcome"] == "INCONCLUSIVE"
    assert report["safety_status"] == "not_run"
    assert all(v is None for v in report["safety"].values())
    fabricated = copy.deepcopy(report)
    fabricated["safety"]["key_failures"] = 0
    assert not VALIDATOR.is_valid(fabricated)
    assert len(report["paired"]) == 10
    assert len({(x["baseline"], x["challenger"], x["cohort"])
                for x in report["paired"]}) == 10
    for comparison in report["paired"]:
        if comparison["cohort"] == "context_contrast":
            assert comparison["n"] == 0
            assert comparison["absolute_error_reduction_ci95"] is None

    model_path = ROOT / "canonical-model-sample.json"
    model = json.loads(model_path.read_text())
    VALIDATOR.validate(model)
    assert encoded(model) + b"\n" == model_path.read_bytes()
    assert digest(model) == n["canonical_model_sha256"]
    assert encoded({"w": -0.0}) == b'{"w":0.0}'
    ordered = sorted(n["ordered_observations"], key=lambda x: (
        {"literal": 0, "return": 1}[x["intent"]], x["component_id"],
        x["fixture_id"], x["event_index"], x["candidate_identity"]))
    assert [x["fixture_id"] for x in ordered] == n["expected_observation_order"]
    close(sum(x["x"] for x in ordered) / len(ordered), n["mean"])

    bank = json.loads((ROOT / "bank-example.json").read_text())
    VALIDATOR.validate(bank)
    assert unicodedata.normalize("NFC", bank["prompt"]) == bank["prompt"]
    assert bank["id"] == digest({k: bank[k] for k in (
        "source_ref", "intent", "prompt", "raw", "gold")})[:24]
    assert bank["prototype_id"] == bank["id"]
    labels = digest({k: bank[k] for k in ("intent", "raw", "gold")})
    assert len({a["annotator_id"] for a in bank["annotations"]}) == 2
    assert all(a["labels_sha256"] == labels for a in bank["annotations"])

    for case in json.loads((ROOT / "lifecycle.json").read_text())["cases"]:
        before, expected = case["before"], case["expected"]
        if case["event"] == "cancel":
            assert expected["app"] == before["app"]
            assert expected["prelude"] == before["prelude"]
            assert expected["view"] == before["app"] + before["prelude"]
            assert expected["commit"] == ""
        else:
            full = before["prelude"] + before["display"] + case["append_text"]
            assert expected["commit"] == full
            assert expected["app"] == before["app"] + full
            assert expected["view"] == expected["app"]
            assert expected["prelude"] == expected["display"] == ""
        assert expected["closed"] and not expected["eligible"]

    cohort = json.loads((ROOT / "cohort-example.json").read_text())
    selected = [c["id"] for c in cohort["components"]
                if c["primary_return"]["contrast_eligible"]]
    assert selected == cohort["expected_contrast_components"]
    counts = {k: 0 for k in cohort["expected_primary_categories"]}
    for c in cohort["components"]:
        counts[c["primary_return"]["category"]] += 1
    assert counts == cohort["expected_primary_categories"]

    occurrences = [json.loads(x) for x in (ROOT / "occurrences.jsonl").read_text().splitlines()]
    for occurrence in occurrences:
        VALIDATOR.validate(occurrence)
        query = {k: occurrence[k] for k in (
            "fixture_id", "condition", "event_index", "raw")}
        assert occurrence["query_id"] == "q:" + digest(query)
    assert occurrences[0]["candidate"]["id"] == occurrences[1]["candidate"]["id"]
    assert occurrences[0]["query_id"] != occurrences[1]["query_id"]
    assert occurrences[0]["candidate"]["features"] != occurrences[1]["candidate"]["features"]

    seen = json.loads((ROOT / "seen-example.json").read_text())
    assert digest(seen["row_ids"]) == seen["component_id"]
    assert digest(seen["sources_empty"]) == seen["seen_empty_sha256"]
    assert digest(seen["sources_seen"]) == seen["seen_nonempty_sha256"]
    def is_seen(sources):
        return any(m["prefix"].lower() in sources["seen_prefixes"] or
                   m["raw"] in sources["seen_raw"] or
                   any(w in sources["seen_contents"] for w in m["contents"])
                   for m in seen["members"])
    assert not is_seen(seen["sources_empty"])
    assert is_seen(seen["sources_seen"])
    bucket = int(hashlib.sha256(("shift-return-v2:" + seen["component_id"]).encode()).hexdigest(), 16) % 10
    expected_split = "train" if bucket < 6 else "dev" if bucket < 8 else "test"
    assert expected_split == seen["expected_empty_split"]
    assert seen["expected_seen_split"] == "regression"

    runtime = [json.loads(x) for x in (ROOT / "runtime-snapshots.jsonl").read_text().splitlines()]
    for snapshot in runtime:
        VALIDATOR.validate(snapshot)
        assert snapshot["kind"] == "snapshot"
        assert snapshot["query_id"] is None
        assert "gold_ids" not in snapshot and "candidate" not in snapshot
    assert runtime[0]["handoff"] is None
    assert runtime[1]["handoff"]["action"] == "commit_only"
    invalid_runtime = copy.deepcopy(runtime[0])
    invalid_runtime["gold_ids"] = []
    assert not VALIDATOR.is_valid(invalid_runtime)

    metadata_fixtures = [json.loads(x) for x in (ROOT / "metadata-fixtures.jsonl").read_text().splitlines()]
    metadata_rows = [json.loads(x) for x in (ROOT / "metadata-rows.jsonl").read_text().splitlines()]
    for f in metadata_fixtures:
        fixture_semantics(f)
    for row in metadata_rows:
        VALIDATOR.validate(row)
        assert row["primary_metadata"]["stratum"] == "1"
        assert row["primary_metadata"]["id"] == metadata_fixtures[0]["id"]
        assert row["primary_category"] == row["primary_metadata"]["category"]
        assert row["contrast_eligible"] == row["primary_metadata"]["contrast_eligible"]
    assert metadata_fixtures[1]["stratum"] == "17-64"
    assert metadata_rows[1]["fixture_id"] != metadata_rows[1]["primary_metadata"]["id"]

    diagnostic_cases = json.loads((ROOT / "generator-examples.json").read_text())["cases"]
    for case in diagnostic_cases:
        diagnostic = case["diagnostic"]
        VALIDATOR.validate(diagnostic)
        assert "gold_id" not in diagnostic  # Grader owns the expected label.
        assert diagnostic["kept_count"] == len(diagnostic["kept_ids"])
        if diagnostic["status"] != "ok":
            outcome = "unavailable"
            assert diagnostic["uncapped_ids"] is None
            assert diagnostic["uncapped_count"] is None
            assert diagnostic["probe_lex_queries"] == 250000
        else:
            assert diagnostic["uncapped_count"] == len(diagnostic["uncapped_ids"])
            assert diagnostic["kept_ids"] == diagnostic["uncapped_ids"][:256]
            outcome = "cap_loss" if case["gold_id"] in diagnostic["uncapped_ids"] else "generator_miss"
            assert case["gold_id"] not in diagnostic["kept_ids"]
        assert outcome == case["name"]
    assert diagnostic_cases[0]["diagnostic"]["kept_ids"] == diagnostic_cases[1]["diagnostic"]["kept_ids"]

    sparse = n["sparse_bootstrap"]
    rng = random.Random(271828)
    values, zero_baselines = [], 0
    for _ in range(10000):
        indices = [rng.randrange(4) for _ in range(4)]
        errors = sum(sparse["baseline_errors"][i] for i in indices)
        zero_baselines += errors == 0
        values.append(errors / 4)  # All new errors are zero; absolute reduction.
    values.sort()
    def quantile(q):
        index = (len(values) - 1) * q
        lo, hi = math.floor(index), math.ceil(index)
        return values[lo] + (values[hi] - values[lo]) * (index - lo)
    assert zero_baselines == sparse["baseline_zero_replicates"] > 0
    assert len(values) == sparse["valid_replicates"]
    for got, expected in zip([quantile(.025), quantile(.975)],
                             sparse["absolute_error_reduction_ci95"]):
        close(got, expected)
    assert quantile(.025) == 0  # Positive point estimate is insufficient evidence.

    print(f"Design checks passed: {len(fixtures)} fixtures, 3 rejected cases, analytic arithmetic, 8-condition/10-comparison hypothetical report.")
    print("Canonical bytes/hash, bank labels, primary cohorts, seen exclusion, query occurrences, unavailable safety, prelude lifecycle and sparse-error bootstrap verified.")
    print("Runtime/training separation, handoff shape, primary/variant strata and generator cap-loss versus search-miss/exhaustion evidence verified.")
    print("No public-dictionary quality replay, model fitting or product change was performed.")


if __name__ == "__main__":
    main()
