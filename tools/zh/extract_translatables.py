# Extract translatable strings from embedded card packs into work files.
# by 兔兔 — 翻译流水线第一步：把英文卡牌文本抽成待译工作文件
#
# Usage: python tools/zh/extract_translatables.py
# Output: tools/zh/work/<pack>.json — list of {code, name, subname?, text?,
#         traits?, flavor?, back_name?, back_text?, back_flavor?}
# Fields with empty values are omitted so translation nodes stay lean.

import glob
import json
import os
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
PACKS_DIR = os.path.join(ROOT, "internal", "engine", "data", "packs")
OUT_DIR = os.path.join(ROOT, "tools", "zh", "work")

FIELDS = ["name", "subname", "text", "traits", "flavor",
          "back_name", "back_text", "back_flavor"]


def main() -> int:
    os.makedirs(OUT_DIR, exist_ok=True)
    total_cards = total_fields = 0
    for path in sorted(glob.glob(os.path.join(PACKS_DIR, "*.json"))):
        pack = os.path.basename(path)
        if pack == "packs.json":
            continue
        with open(path, "r", encoding="utf-8") as f:
            data = json.load(f)
        cards = data if isinstance(data, list) else data.get("cards", [])
        items = []
        for c in cards:
            item = {"code": c.get("code", "")}
            for fld in FIELDS:
                val = c.get(fld)
                if isinstance(val, str) and val.strip():
                    item[fld] = val
            # duplicate_of cards share text with their original — still need
            # a translated name, but skip re-translating identical text.
            if c.get("duplicate_of_code"):
                item["_dup_of"] = c["duplicate_of_code"]
            items.append(item)
        out_path = os.path.join(OUT_DIR, pack)
        with open(out_path, "w", encoding="utf-8") as f:
            json.dump(items, f, ensure_ascii=False, indent=1)
        nf = sum(len(i) - 1 - (1 if "_dup_of" in i else 0) for i in items)
        total_cards += len(items)
        total_fields += nf
        print(f"{pack}: {len(items)} cards, {nf} fields")
    print(f"TOTAL: {total_cards} cards, {total_fields} translatable fields")
    return 0


if __name__ == "__main__":
    sys.exit(main())
