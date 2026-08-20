# Merge chunk files into a final pack translation file.
# by 兔兔 — 短跑兔战术的收网器：chunks/<pack>_*.json → out/<pack>.json
#
# Usage: python tools/zh/merge_chunks.py <pack> [pack ...]

import json
import os
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
ZH = os.path.join(ROOT, "tools", "zh")
CHUNK_DIR = os.path.join(ZH, "out", "chunks")
OUT_DIR = os.path.join(ZH, "out")
WORK_DIR = os.path.join(ZH, "work")


def merge_pack(pack: str) -> None:
    chunks = sorted(
        f for f in os.listdir(CHUNK_DIR)
        if f.startswith(pack + "_") and f.endswith(".json")
    )
    merged = {}
    for ch in chunks:
        path = os.path.join(CHUNK_DIR, ch)
        try:
            with open(path, "r", encoding="utf-8-sig") as f:
                data = json.load(f)
        except Exception as e:
            print(f"  SKIP {ch}: {e}")
            continue
        merged.update(data)
        print(f"  {ch}: {len(data)} cards")
    work_path = os.path.join(WORK_DIR, pack + ".json")
    expected = len(json.load(open(work_path, "r", encoding="utf-8")))
    out_path = os.path.join(OUT_DIR, pack + ".json")
    with open(out_path, "w", encoding="utf-8") as f:
        json.dump(merged, f, ensure_ascii=False, indent=1)
    status = "OK" if len(merged) == expected else f"COUNT MISMATCH {len(merged)}/{expected}"
    print(f"{pack}: merged {len(merged)}/{expected} -> {status}")


def main() -> int:
    if len(sys.argv) < 2:
        print(__doc__)
        return 2
    for pack in sys.argv[1:]:
        merge_pack(pack)
    return 0


if __name__ == "__main__":
    sys.exit(main())
