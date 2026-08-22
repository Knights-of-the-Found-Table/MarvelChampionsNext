#!/usr/bin/env python3
"""Normalize double-sided card face imagesrc in the embedded pack data.

This repo standardizes on the image-file convention: {base}a.png is the A
face and {base}b.png the B face. marvelcdb's own storage is inconsistent:

  case-1 (24 main schemes, incl. the whole core set): the parent record's
      imagesrc {base}.png holds the B-face image, backimagesrc {base}b.png
      holds the A-face image, and {base}a.png does not exist.
  case-2 (44 main schemes + heroes/villains): {base}a.{png,jpg} holds the
      A face and {base}b.{png,jpg} the B face, matching the convention.

The face-split records in packs/*.json were generated mechanically
(parent.backimagesrc -> b face), so for case-1 cards the faces are
mislabeled: the {base}b record points at what is actually the A-face image.

The script rewrites every face record's imagesrc to the path where marvelcdb
really stores that face, so the default (English) image chain can keep
fetching from marvelcdb directly. The zh chain does not use imagesrc at all
(it requests {code}.png by convention against a mirror laid out that way).

  case-1: a-face.imagesrc = parent.backimagesrc  (marvelcdb's A-face file)
          b-face.imagesrc = parent.imagesrc      (marvelcdb's B-face file)
  case-2: a-face.imagesrc = parent.imagesrc
          b-face.imagesrc = parent.backimagesrc

Faces without a parent record (heroes, villains) are only verified. Files are
re-serialized byte-identically to their input format (round-trip asserted
against several escape profiles) except for the changed fields.

Usage:
  python tools/normalize_faces.py            # report planned changes only
  python tools/normalize_faces.py --write    # apply them
"""

import itertools
import json
import os
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
PACKS = os.path.join(ROOT, "internal", "engine", "data", "packs")


def side_suffix(code):
    if len(code) == 6 and code[5] in "abcd":
        return code[5]
    return ""


def stem_of(imagesrc):
    """01097b.png -> 01097b for a /bundles/cards/... path."""
    name = imagesrc.rsplit("/", 1)[-1]
    dot = name.rfind(".")
    return name[:dot] if dot > 0 else name


def reserialize(data, ensure_ascii, esc_slash, esc_html, esc_amp):
    out = json.dumps(data, ensure_ascii=ensure_ascii, indent=2)
    if esc_html:
        out = out.replace("<", "\\u003c").replace(">", "\\u003e")
    if esc_amp:
        out = out.replace("&", "\\u0026")
    if esc_slash:
        out = out.replace("/", "\\/")
    return out


def match_profile(data, text):
    """Find the serialization profile that reproduces text byte-for-byte."""
    newline = text.endswith("\n")
    body = text[:-1] if newline else text
    for combo in itertools.product((True, False), repeat=4):
        if reserialize(data, *combo) == body:
            return combo, newline
    return None, newline


def collect_records(cards):
    """Index top-level records and nested linked_card objects by code."""
    by_code = {}
    for rec in cards:
        code = rec.get("code")
        if code:
            by_code[code] = rec
        linked = rec.get("linked_card")
        if isinstance(linked, dict) and linked.get("code"):
            by_code.setdefault(linked["code"], linked)
    return by_code


def normalize_pack(path, write):
    text = open(path, encoding="utf-8").read()
    cards = json.loads(text)
    combo, newline = match_profile(cards, text)
    if combo is None:
        return None, None, f"round-trip mismatch: no escape profile reproduces the file"

    changes = []
    by_code = collect_records(cards)
    for code, rec in by_code.items():
        if side_suffix(code):
            continue  # faces are handled via their parent below
        front, back = rec.get("imagesrc"), rec.get("backimagesrc")
        if not front or not back:
            continue
        fstem = stem_of(front)
        if fstem == code:
            case = 1  # parent imagesrc holds the B face
        elif fstem == code + "a":
            case = 2  # parent imagesrc holds the A face
        else:
            changes.append((code, "parent", front, "UNEXPECTED parent imagesrc stem"))
            continue
        # parent.text belongs to the b face in case-1, to the a face in
        # case-2; assign each face the marvelcdb file that truly depicts it.
        a_src = back if case == 1 else front
        b_src = front if case == 1 else back
        for face, src in ((code + "a", a_src), (code + "b", b_src)):
            frec = by_code.get(face)
            if frec is None:
                continue
            old = frec.get("imagesrc")
            if src and old != src:
                changes.append((face, old, src, f"case-{case}"))
                if write:
                    set_image_key(frec, src)
    if write and changes:
        out = reserialize(cards, *combo) + ("\n" if newline else "")
        with open(path, "w", encoding="utf-8", newline="") as f:
            f.write(out)
    return changes, combo, None


def set_image_key(rec, src):
    """Set imagesrc, placing a newly added key after "url" (the field's
    position in the original data) instead of at the end of the object."""
    if "imagesrc" in rec:
        rec["imagesrc"] = src
        return
    out = {}
    for k, v in rec.items():
        out[k] = v
        if k == "url":
            out["imagesrc"] = src
    if "imagesrc" not in out:
        out["imagesrc"] = src
    rec.clear()
    rec.update(out)


def main():
    write = "--write" in sys.argv
    total, failed = 0, 0
    for fn in sorted(os.listdir(PACKS)):
        if not fn.endswith(".json") or fn == "packs.json":
            continue
        changes, _, err = normalize_pack(os.path.join(PACKS, fn), write)
        if err:
            print(f"{fn}: ERROR {err}")
            failed += 1
            continue
        if changes:
            print(f"{fn}:")
            for c in changes:
                print(f"  {c[0]}: imagesrc {c[1]!r} -> {c[2]!r}  ({c[3]})")
            total += len(changes)
    mode = "written" if write else "planned"
    print(f"\n{total} imagesrc changes {mode}, {failed} files failed")
    sys.exit(1 if failed else 0)


if __name__ == "__main__":
    main()
