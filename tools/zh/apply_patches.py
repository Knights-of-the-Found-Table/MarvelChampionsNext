# Apply repair patches into final pack translation files.
# by 兔兔 — 补丁施加器：out/patches/repair_*.json 打进 out/<pack>.json
#
# Usage: python tools/zh/apply_patches.py
# Patch format: {"<pack>": {"<code>": {"<field>": "<corrected text>"}}}

import glob
import json
import os
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
ZH = os.path.join(ROOT, "tools", "zh")
PATCH_GLOB = os.path.join(ZH, "out", "patches", "repair_*.json")
OUT_DIR = os.path.join(ZH, "out")


def main() -> int:
    patches = sorted(glob.glob(PATCH_GLOB))
    if not patches:
        print("no patch files found")
        return 1
    applied = skipped = 0
    for pf in patches:
        try:
            with open(pf, "r", encoding="utf-8-sig") as f:
                patch = json.load(f)
        except Exception as e:
            print(f"SKIP {os.path.basename(pf)}: {e}")
            skipped += 1
            continue
        for pack, codes in patch.items():
            out_path = os.path.join(OUT_DIR, pack + ".json")
            if not os.path.exists(out_path):
                print(f"SKIP {pack}: no output file")
                continue
            with open(out_path, "r", encoding="utf-8-sig") as f:
                data = json.load(f)
            n = 0
            for code, fields in codes.items():
                if code not in data:
                    print(f"  WARN {pack}/{code}: not in output")
                    continue
                for fld, text in fields.items():
                    data[code][fld] = text
                    n += 1
            with open(out_path, "w", encoding="utf-8") as f:
                json.dump(data, f, ensure_ascii=False, indent=1)
            print(f"{os.path.basename(pf)} -> {pack}: {n} fields patched")
            applied += n
    print(f"total applied: {applied}, skipped files: {skipped}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
