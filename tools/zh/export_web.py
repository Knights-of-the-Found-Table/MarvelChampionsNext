# 给前端导出简中卡牌数据：tools/zh/out/*.json -> 两个前端文件
#
#   web/src/i18n/zh-cards.json   { "<code>": { "name", "subname"? } }
#     卡名映射（小体积），静态打进 JS 包，供对局记录/提示里的卡名本地化。
#
#   web/public/zh-cards-full.json  { "<code>": { "name", "subname"?, "text"?, "traits"? } }
#     完整卡面数据（约 1MB），按需运行时拉取（Ctrl 悬浮信息层首次显示时），
#     不进 JS 包。网页其余场景以卡图渲染，不需要正文。
#
# Usage:
#   python tools/zh/export_web.py
#
# 译文更新后重跑本脚本，并把重新生成的文件一起提交。

import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
OUT_DIR = ROOT / "tools" / "zh" / "out"
NAMES = ROOT / "web" / "src" / "i18n" / "zh-cards.json"
FULL = ROOT / "web" / "public" / "zh-cards-full.json"


def main() -> int:
    names = {}
    full = {}
    for pack_file in sorted(OUT_DIR.glob("*.json")):
        # out 文件带 UTF-8 BOM（merge_chunks.py 写入时加了 utf-8-sig）。
        for code, fields in json.loads(pack_file.read_text(encoding="utf-8-sig")).items():
            entry = {"name": fields["name"]}
            if fields.get("subname"):
                entry["subname"] = fields["subname"]
            names[code] = entry
            rich = dict(entry)
            if fields.get("text"):
                rich["text"] = fields["text"]
            if fields.get("traits"):
                rich["traits"] = fields["traits"]
            full[code] = rich
    def write_lf(path: Path, obj) -> None:
        # 固定 LF 行尾：gitattributes 要求 *.json eol=lf
        with open(path, "w", encoding="utf-8", newline="\n") as f:
            json.dump(obj, f, ensure_ascii=False, indent=1)

    NAMES.parent.mkdir(parents=True, exist_ok=True)
    write_lf(NAMES, names)
    FULL.parent.mkdir(parents=True, exist_ok=True)
    write_lf(FULL, full)
    print(f"done: {len(names)} names -> {NAMES}")
    print(f"done: {len(full)} full cards -> {FULL}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
