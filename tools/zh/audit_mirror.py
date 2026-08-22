#!/usr/bin/env python3
"""Audit the zh image mirror against the {code}.png face convention.

The zh chain requests /bundles/cards/{code}.png for every card code
(single-sided cards by their plain code, double-sided faces by {base}a /
{base}b). This script probes ZH_IMAGE_MIRROR for each code, classifies the
gaps, and writes an actionable fix list for the mirror's maintainer:

  - REQUIRED codes are reachable at runtime (scheme b faces, hero/villain
    faces, deck cards); a miss means the player sees the English image via
    the default-chain fallback.
  - OPTIONAL codes are only requested by prewarm (content-addressed URLs);
    a miss is harmless.
  - Codes already seeded in the local zh cache (cache/images/zh) are served
    from disk and do not need a mirror copy.

For each missing file the script also probes common sibling names (other
extensions, the unsuffixed base) as rename hints.

Usage:
  python tools/zh/audit_mirror.py            # write tools/zh/zh_mirror_audit.txt
  python tools/zh/audit_mirror.py --verbose  # also print every missing code
"""

import concurrent.futures
import datetime
import json
import os
import sys
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
PACKS = ROOT / "internal" / "engine" / "data" / "packs"
SEED_MANIFEST = ROOT / "cache" / "images" / "zh" / "manifest.json"
REPORT = ROOT / "tools" / "zh" / "zh_mirror_audit.txt"

FACE_REQUIRED_TYPES = {"hero", "alter_ego", "villain"}
B_REQUIRED_TYPES = FACE_REQUIRED_TYPES | {"main_scheme"}


def side_of(code):
    return code[5] if len(code) == 6 and code[5] in "abcd" else ""


def load_cards():
    """All card codes with (type, base, is_parent), linked faces included."""
    cards = {}
    for pack_file in sorted(PACKS.glob("*.json")):
        if pack_file.name == "packs.json":
            continue
        for rec in json.loads(pack_file.read_text(encoding="utf-8")):
            code = rec.get("code", "")
            if code:
                cards.setdefault(code, rec)
            lc = rec.get("linked_card")
            if isinstance(lc, dict) and lc.get("code"):
                cards.setdefault(lc["code"], lc)
    return cards


def classify(cards):
    """Split codes into runtime-reachable (required) vs prewarm-only."""
    required, optional = [], []
    for code, rec in cards.items():
        side = side_of(code)
        base = code[:5] if side else code
        if side == "":
            # Plain codes are deck/entity cards; parent records of
            # double-sided cards are no longer requested at runtime.
            is_parent = bool(rec.get("double_sided")) or bool(rec.get("backimagesrc")) or (base + "a" in cards)
            (optional if is_parent else required).append(code)
        elif side == "a":
            required.append(code)  # display face: decks ship a-sides
        elif side == "b":
            typ = rec.get("type_code", "")
            (required if typ in B_REQUIRED_TYPES else optional).append(code)
        else:
            (required if rec.get("type_code") in FACE_REQUIRED_TYPES else optional).append(code)
    return sorted(required), sorted(optional)


def load_seed():
    if SEED_MANIFEST.exists():
        try:
            return set(json.loads(SEED_MANIFEST.read_text(encoding="utf-8-sig")))
        except (OSError, ValueError):
            pass
    return set()


def mirror_root():
    url = os.environ.get("ZH_IMAGE_MIRROR")
    if not url:
        env = ROOT / ".env"
        if env.exists():
            for line in env.read_text(encoding="utf-8-sig").splitlines():
                if line.startswith("ZH_IMAGE_MIRROR="):
                    url = line.split("=", 1)[1].strip()
                    break
    if not url:
        sys.exit("ZH_IMAGE_MIRROR not set (env or .env)")
    return url.rstrip("/")


def head(url, timeout=15):
    req = urllib.request.Request(url, method="HEAD",
                                 headers={"User-Agent": "marvelchampionsnext/audit"})
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status == 200
    except Exception:
        return False


def probe_many(root, codes, workers=8):
    """HEAD-check /bundles/cards/{code}.png for every code; return misses."""
    missing = set()

    def check(code):
        if not head(f"{root}/bundles/cards/{code}.png"):
            missing.add(code)

    with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as pool:
        list(pool.map(check, codes))
    return missing


def rename_hints(root, code):
    """Sibling object names that might hold this face, probed on demand."""
    side = side_of(code)
    base = code[:5] if side else code
    candidates = [f"{code}.jpg", f"{code}.webp"]
    if side:
        candidates += [f"{base}.png", f"{base}.jpg"]
    return [c for c in candidates if head(f"{root}/bundles/cards/{c}")]


def main():
    verbose = "--verbose" in sys.argv
    cards = load_cards()
    required, optional = classify(cards)
    seeded = load_seed()
    root = mirror_root()

    print(f"probing {root} for {len(cards)} card codes "
          f"({len(required)} required, {len(optional)} optional)...")
    missing = probe_many(root, sorted(cards))

    req_missing = [c for c in required if c in missing and c not in seeded]
    opt_missing = [c for c in optional if c in missing and c not in seeded]
    req_local = [c for c in required if c in missing and c in seeded]

    lines = [
        f"# zh 镜像审计 — {datetime.date.today()}",
        f"# 镜像: {root}",
        f"# 必需缺失 {len(req_missing)}（运行时回退英文图） | "
        f"本地缓存已种 {len(req_local)} | 可选缺失 {len(opt_missing)}（仅影响预热/哈希 URL）",
        "",
        "## 必需缺失（建议按提示补齐或改名）",
    ]
    for code in req_missing:
        hints = rename_hints(root, code)
        hint = f"  <- 疑似存在于 {', '.join(hints)}（改名即可）" if hints else ""
        lines.append(f"{code}.png{hint}")
    lines += ["", "## 可选缺失（运行时会回退英文图，无需处理）"]
    lines += [f"{c}.png" for c in opt_missing] if verbose \
        else [f"# 共 {len(opt_missing)} 个，--verbose 查看"]

    REPORT.write_text("\n".join(lines) + "\n", encoding="utf-8")
    print(f"required missing: {len(req_missing)} (of {len(required)}), "
          f"covered by local seed: {len(req_local)}, optional missing: {len(opt_missing)}")
    print(f"report: {REPORT}")


if __name__ == "__main__":
    main()
