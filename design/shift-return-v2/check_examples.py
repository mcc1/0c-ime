#!/usr/bin/env python3
"""Check design schemas, raw ownership and analytic arithmetic; no IME scoring."""
import copy
import json
import math
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

    report = json.loads((ROOT / "worked-report.json").read_text())
    VALIDATOR.validate(report)
    assert len(report["conditions"]) == 8
    ec = report["conditions"]["EC-revise"]
    assert ec["return_boundary"]["errors"] == 0
    assert ec["literal_only"]["errors"] == 1
    assert ec["literal_only"]["wilson95"][1] > .01
    assert report["outcome"] == "INCONCLUSIVE"
    print(f"Design checks passed: {len(fixtures)} fixtures, 3 rejected cases, analytic arithmetic, 8-condition hypothetical report.")
    print("No public-dictionary quality replay, model fitting or product change was performed.")


if __name__ == "__main__":
    main()
