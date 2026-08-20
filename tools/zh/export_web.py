# 给前端导出简中卡牌名称：tools/zh/out/*.json -> web/src/i18n/zh-cards.json
#
# 页面语言为中文时，前端把服务端英文卡名覆盖为译文。完整译文文件里
# 还有 text/flavor/背面字段，但网页以卡图渲染、从不显示卡牌正文，
# 所以这里只裁剪出界面用到的字段：
#
#   web/src/i18n/zh-cards.json = { "<code>": { "name": "...", "subname": "..." } }
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
TARGET = ROOT / "web" / "src" / "i18n" / "zh-cards.json"


def main() -> int:
    cards = {}
    for pack_file in sorted(OUT_DIR.glob("*.json")):
        # out 文件带 UTF-8 BOM（merge_chunks.py 写入时加了 utf-8-sig）。
        for code, fields in json.loads(pack_file.read_text(encoding="utf-8-sig")).items():
            entry = {"name": fields["name"]}
            if fields.get("subname"):
                entry["subname"] = fields["subname"]
            cards[code] = entry
    TARGET.parent.mkdir(parents=True, exist_ok=True)
    TARGET.write_text(json.dumps(cards, ensure_ascii=False, indent=1), encoding="utf-8")
    print(f"done: {len(cards)} cards -> {TARGET}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
