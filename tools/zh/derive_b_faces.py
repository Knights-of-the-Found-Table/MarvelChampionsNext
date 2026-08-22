#!/usr/bin/env python3
"""Derive {base}b zh entries for case-1 double-sided schemes.

The engine registers main schemes by their b-face code (the gameplay side).
The zh translation files (tools/zh/out/<pack>.json) key entries by card code
and were extracted from top-level records only, so they contain the parent
code but not the nested {base}b face. For "case-1" schemes — parents whose
imagesrc stem equals the base code, i.e. whose own text is the b-face text —
the parent's translation IS the b-face translation. This script copies those
entries to {base}b (keeping only the face-relevant fields; back_* describes
the a face and is dropped).

Faces-only cards (heroes, newer packs like 56063) have no parent entry, so
their b faces stay untranslated until new translations are added.

Usage:
  python tools/zh/derive_b_faces.py           # report planned derivations
  python tools/zh/derive_b_faces.py --write   # apply, then rerun export_web.py
"""

import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
PACKS = ROOT / "internal" / "engine" / "data" / "packs"
OUT = ROOT / "tools" / "zh" / "out"

KEEP_FIELDS = ("name", "subname", "text", "traits", "flavor")


def has_side(code):
    return len(code) == 6 and code[5] in "abcd"


def stem_of(imagesrc):
    name = imagesrc.rsplit("/", 1)[-1]
    dot = name.rfind(".")
    return name[:dot] if dot > 0 else name


def case1_bases(cards):
    """Base codes whose parent record's own face is the b face."""
    bases = []
    for rec in cards:
        code = rec.get("code", "")
        if has_side(code):
            continue
        front, back = rec.get("imagesrc"), rec.get("backimagesrc")
        if not front or not back:
            continue
        if stem_of(front) == code:
            bases.append(code)
    return bases


def reserialize(data, indent):
    return json.dumps(data, ensure_ascii=False, indent=indent)


def main():
    write = "--write" in sys.argv
    total = 0
    for pack_file in sorted(PACKS.glob("*.json")):
        if pack_file.name == "packs.json":
            continue
        out_file = OUT / pack_file.name
        if not out_file.exists():
            continue
        bases = case1_bases(json.loads(pack_file.read_text(encoding="utf-8")))
        if not bases:
            continue
        raw = out_file.read_bytes()
        bom = raw.startswith(b"\xef\xbb\xbf")
        text = raw.decode("utf-8-sig")
        entries = json.loads(text)
        # Match the file's serialization exactly (merge_chunks writes
        # 1-space indent, no trailing newline); abort on mismatch.
        matched = next((i for i in (1, 2, 4) if reserialize(entries, i) == text), None)
        if matched is None:
            print(f"{out_file.name}: round-trip mismatch, skipping")
            continue
        added = []
        for base in bases:
            src = entries.get(base)
            if not src or base + "b" in entries:
                continue
            derived = {k: src[k] for k in KEEP_FIELDS if src.get(k)}
            entries[base + "b"] = derived
            added.append(base + "b")
        if not added:
            continue
        print(f"{out_file.name}: derived {added}")
        total += len(added)
        if write:
            body = reserialize(entries, matched)
            out_file.write_bytes(
                (b"\xef\xbb\xbf" if bom else b"") + body.encode("utf-8")
            )
    mode = "written" if write else "planned"
    print(f"\n{total} b-face entries {mode}")
    if write:
        print("rerun: python tools/zh/export_web.py")


if __name__ == "__main__":
    main()
