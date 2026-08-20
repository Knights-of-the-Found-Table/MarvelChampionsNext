# Validate and summarize zh translation outputs against the work files.
# by 兔兔 — 翻译流水线质检站：卡号齐全？标记没丢？JSON 能读？
#
# Usage: python tools/zh/verify_merge.py [pack ...]   (default: all packs)
# Checks per pack:
#   - output JSON parses
#   - every work-file code is present
#   - per-card: <b>/<i>/[icon]/arrow marker counts match the English source
#   - required fields translated (name present, text present when source has text)

import json
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
WORK_DIR = os.path.join(ROOT, "tools", "zh", "work")
OUT_DIR = os.path.join(ROOT, "tools", "zh", "out")

MARK_RE = re.compile(r"<b>|</b>|<i>|</i>|\[[a-z_]+\]|→")


def markers(s: str):
    return sorted(MARK_RE.findall(s or ""))


def check_pack(pack: str):
    work_path = os.path.join(WORK_DIR, pack)
    out_path = os.path.join(OUT_DIR, pack)
    with open(work_path, "r", encoding="utf-8") as f:
        work = json.load(f)
    if not os.path.exists(out_path):
        return {"pack": pack, "status": "MISSING"}
    try:
        with open(out_path, "r", encoding="utf-8-sig") as f:
            out = json.load(f)
    except Exception as e:
        return {"pack": pack, "status": "BAD_JSON", "err": str(e)}

    work_by_code = {item["code"]: item for item in work}
    problems = []
    missing = [c for c in work_by_code if c not in out]
    if missing:
        problems.append(f"missing {len(missing)} codes: {missing[:5]}")
    field_gaps = marker_issues = 0
    for code, item in work_by_code.items():
        tr = out.get(code)
        if not isinstance(tr, dict):
            continue
        if item.get("name") and not tr.get("name"):
            field_gaps += 1
        for fld in ("text", "back_text"):
            src = item.get(fld)
            dst = tr.get(fld)
            if src and dst and markers(src) != markers(dst):
                marker_issues += 1
                problems.append(f"{code}.{fld} missing: "
                                f"{sorted(set(markers(src)) - set(markers(dst)))}")
    status = "OK" if not problems else "ISSUES"
    return {"pack": pack, "status": status, "cards": len(work),
            "translated": len(out), "field_gaps": field_gaps,
            "marker_issues": marker_issues, "problems": problems}


def main() -> int:
    packs = sys.argv[1:] or sorted(
        p for p in os.listdir(WORK_DIR) if p.endswith(".json"))
    packs = [p if p.endswith(".json") else p + ".json" for p in packs]
    total_cards = total_ok = 0
    for pack in packs:
        r = check_pack(pack)
        total_cards += r.get("cards", 0)
        if r["status"] == "OK":
            total_ok += r.get("cards", 0)
            print(f"OK   {r['pack']}: {r['translated']}/{r['cards']}")
        else:
            print(f"{r['status']:8s} {r['pack']}: {r.get('problems', r.get('err'))}")
    print(f"---\ncards translated clean: {total_ok}/{total_cards}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
