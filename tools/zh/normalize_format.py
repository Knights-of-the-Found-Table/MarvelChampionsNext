# 统一翻译数据 JSON 的排版：indent=1、LF、无 BOM、ensure_ascii=False、无末尾换行。
# 仅归一化格式，不改任何内容；落盘后做解析级回读断言。
# 与 merge_chunks / apply_patches / export_web 的输出风格保持一致。
#
# Usage: python tools/zh/normalize_format.py

import glob
import json
import os
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
PATTERNS = [
    os.path.join(ROOT, "tools", "zh", "out", "*.json"),
    os.path.join(ROOT, "tools", "zh", "out", "patches", "*.json"),
]


def main() -> int:
    changed = unchanged = 0
    for pat in PATTERNS:
        for path in sorted(glob.glob(pat)):
            with open(path, encoding="utf-8-sig") as f:
                raw = f.read()
            data = json.loads(raw)  # utf-8-sig 兼容带/不带 BOM
            text = json.dumps(data, ensure_ascii=False, indent=1)
            if text == raw:
                unchanged += 1  # 已是标准形态（无 BOM、LF、indent=1）
                continue
            with open(path, "w", encoding="utf-8", newline="\n") as f:
                f.write(text)
            with open(path, encoding="utf-8-sig") as f:
                assert json.load(f) == data, path  # 回读一致
            changed += 1
            print(f"normalized {os.path.relpath(path, ROOT)}")
    print(f"done: {changed} rewritten, {unchanged} already clean")
    return 0


if __name__ == "__main__":
    sys.exit(main())
