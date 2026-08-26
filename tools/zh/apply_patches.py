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


def detect_indent(raw_text: str):
    """保持各文件既有格式：单行紧凑返回 None，否则取首个条目行的缩进宽度。"""
    stripped = raw_text.strip()
    if "\n" not in stripped:
        return None
    for line in stripped.splitlines()[1:]:
        if line.strip():
            return len(line) - len(line.lstrip(" "))
    return None


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
            raw = open(out_path, "r", encoding="utf-8-sig").read()
            data = json.loads(raw)
            indent = detect_indent(raw)
            had_trailing_newline = raw.endswith("\n")
            n = 0
            for code, fields in codes.items():
                if code not in data:
                    print(f"  WARN {pack}/{code}: not in output")
                    continue
                for fld, text in fields.items():
                    data[code][fld] = text
                    n += 1
            if n == 0:
                continue  # 无可写条目：不重写文件，避免无谓格式变更
            with open(out_path, "w", encoding="utf-8", newline="\n") as f:
                json.dump(data, f, ensure_ascii=False, indent=indent)
                if had_trailing_newline:
                    f.write("\n")
            print(f"{os.path.basename(pf)} -> {pack}: {n} fields patched")
            applied += n
    print(f"total applied: {applied}, skipped files: {skipped}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
