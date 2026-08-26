# 以外部 OCR 对照表为准修订中文卡名，生成 repair 补丁与差异报告。
# 对照表格式：{"<code>": "<中文名>"}；来源带 ✦ 前缀（独有标记），一律剥离。
#
# 规则：
#   1. 忽略「·」和空格后与现有 name（或 name+subname）相同 → 视为一致，不动。
#   2. 有 subname 的卡只换主名、保留 subname（对照表不携带副标题）。
#   3. 同英文名的多处印本在表内互证：多数派一致而个别错字 → 统一为多数派。
#   4. 无法确证的 OCR 噪声（括号不平衡 / 边缘标点 / 粘连数字等）不落盘，进报告。
# 输出：
#   out/patches/repair_code2name_names.json
#   out/reports/code2name_report.md
#
# Usage: python tools/zh/import_code2name.py <path/to/code2name.json>

import difflib
import glob
import json
import os
import re
import sys
from collections import Counter, defaultdict

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
ZH = os.path.join(ROOT, "tools", "zh")
OUT_DIR = os.path.join(ZH, "out")
PACKS_DIR = os.path.join(ROOT, "internal", "engine", "data", "packs")

# 人工确证的 OCR 噪声修正（key 为 code，value 为修后名称）
CURATED_FIXES = {
    "01099": "冲锋",            # 表内为「冲锋。」多句号
    "07024": "出逃囚犯",        # 形近错字；07010/07038 等同卡印本均作「出逃囚犯」
    "07053": "出逃囚犯",        # 第三副犯人牌组的同卡印本，表内误作「出逃计划」
    "11005": "康（猩红百夫长）",  # 缺右括号；11038 同名完整
    "21165b": "洛基王万岁",     # 粘连「122」；21165a 同卡 A 面正确
    "32088b": "李千欢",         # 表内误作「李干欢」；35003 同角色作「李千欢」
    # 以下 3 条：表内同基码 a/b 面本应同名却漂移，按 a 面（主面）对齐
    "45085b": "天启四骑士来袭",  # 表内作「…来袭！」多感叹号
    "45183b": "米哈伊尔·拉斯普廷",  # 表内误作「拉斯普京」（姓氏译字漂移）
    "32125b": "兄弟会强袭！",    # 表内丢句尾感叹号
}


def norm(s: str) -> str:
    return s.replace("·", "").replace(" ", "").strip()


def load_json(path):
    with open(path, encoding="utf-8-sig") as f:
        return json.load(f)


def main() -> int:
    if len(sys.argv) < 2:
        print(__doc__)
        return 1
    src = sys.argv[1]
    raw = load_json(src)

    # ---- 清洗 + 人工修正 ----
    ocr, suspects, curated_applied = {}, [], []
    for code, text in raw.items():
        c = text.replace("✦", "").replace("!", "！").strip()
        if code in CURATED_FIXES:
            curated_applied.append((code, c, CURATED_FIXES[code]))
            c = CURATED_FIXES[code]
        if not c:
            suspects.append((code, text, "空串"))
            continue
        if c.startswith(("。", "，", "、", "！", "？")) or c.endswith(("。", "，", "、")):
            suspects.append((code, text, "边缘标点"))
            continue
        if c.count("（") != c.count("）") or c.count("(") != c.count(")"):
            suspects.append((code, text, "括号不平衡"))
            continue
        m = re.match(r"^([0-9]+)([\u4e00-\u9fff])", c)
        if m and not c[len(m.group(1)):].startswith(("号", "之爪")):
            suspects.append((code, text, "粘连数字"))
            continue
        ocr[code] = c

    # ---- 汇总各翻译文件的出现位置 ----
    occurrences = defaultdict(list)
    total_entries = {}
    for path in sorted(glob.glob(os.path.join(OUT_DIR, "*.json"))):
        pack = os.path.splitext(os.path.basename(path))[0]
        data = load_json(path)
        total_entries[pack] = data
        for code in data:
            occurrences[code].append(pack)

    # ---- 同英文名副本互证 ----
    en_groups = defaultdict(list)
    for path in sorted(glob.glob(os.path.join(PACKS_DIR, "*.json"))):
        data = load_json(path)
        if not isinstance(data, list):
            continue
        for card in data:
            if isinstance(card, dict) and card.get("code") and card.get("name"):
                en_groups[card["name"]].append(card["code"])
                linked = card.get("linked_card")
                if isinstance(linked, dict) and linked.get("code") and linked.get("name"):
                    en_groups[linked["name"]].append(linked["code"])

    unified, unify_rejected = [], []
    for en_name, codes in en_groups.items():
        members = sorted({c for c in codes if c in ocr})
        if len(members) < 3:
            continue
        buckets = defaultdict(list)
        for c in members:
            buckets[norm(ocr[c])].append(c)
        if len(buckets) < 2:
            continue
        ranked = sorted(buckets.values(), key=len, reverse=True)
        majority, rest = ranked[0], ranked[1:]
        if len(majority) < 2 or any(len(b) > 1 for b in rest):
            continue
        rep = Counter(ocr[c] for c in majority).most_common(1)[0][0]
        for c in rest[0]:
            sim = difflib.SequenceMatcher(None, ocr[c], rep).ratio()
            # 只有机械性错字（近乎相同）才采信多数派；翻译取舍差异尊重对照表原值
            if sim >= 0.85:
                unified.append((c, en_name, ocr[c], rep, f"{sim:.2f}", len(majority)))
                ocr[c] = rep
            else:
                unify_rejected.append((c, en_name, ocr[c], rep))

    # ---- 与 out/ 现状比对 ----
    patch, details, sub_kept, noop_norm = defaultdict(dict), [], [], 0
    missing_in_repo, same_before = [], 0
    for code in sorted(ocr):
        target = ocr[code]
        packs = occurrences.get(code)
        if not packs:
            missing_in_repo.append(code)
            continue
        fields = total_entries[packs[0]][code]
        cur_name = fields.get("name", "")
        cur_sub = fields.get("subname", "")
        if cur_name == target:
            same_before += 1
            continue
        if norm(cur_name) == norm(target) or (cur_sub and norm(cur_name + cur_sub) == norm(target)):
            noop_norm += 1  # 仅 · / 空格或拆分方式不同，保留现状
            continue
        if cur_sub:
            sub_kept.append((code, packs[0], cur_name, cur_sub, target))
        for p in packs:  # 所有出现位置都打同一补丁，规避导出顺序遮蔽
            patch[p][code] = {"name": target}
        details.append((packs[0], code, cur_name, cur_sub, target))

    # ---- 写补丁 ----
    patch_path = os.path.join(OUT_DIR, "patches", "repair_code2name_names.json")
    with open(patch_path, "w", encoding="utf-8", newline="\n") as f:
        json.dump(patch, f, ensure_ascii=False, indent=1)
        f.write("\n")

    # ---- 写报告 ----
    report_dir = os.path.join(OUT_DIR, "reports")
    os.makedirs(report_dir, exist_ok=True)
    repo_uncovered = [c for p, d in total_entries.items() if p != "zz_backfaces"
                      for c in d if c not in raw]
    report_path = os.path.join(report_dir, "code2name_report.md")
    with open(report_path, "w", encoding="utf-8", newline="\n") as f:
        w = f.write
        w("# code2name 卡名对齐报告\n\n")
        w(f"- 对照表条目：{len(raw)}\n")
        w(f"- 现状完全一致（逐字）：{same_before}\n")
        w(f"- 归一化后视为一致而跳过（仅 ·/空格/拆分差异）：{noop_norm}\n")
        w(f"- 实际改名：{sum(len(v) for v in patch.values())} 处（{len(details)} 个独立 code，含多文件重复计 {sum(len(v) for v in patch.values()) - len(details)}）\n")
        w(f"- 仓库有而对照表无（未动）：{len(repo_uncovered)}\n")
        w(f"- 对照表有而仓库无（跳过）：{len(missing_in_repo)} {missing_in_repo}\n\n")
        w("## 人工确证的修正\n\n| code | 原 | 修后 |\n|---|---|---|\n")
        for code, before, after in curated_applied:
            w(f"| {code} | {before} | {after} |\n")
        w("\n## 同名卡副本统一\n\n| code | 英文名 | 表内值 | 统一为 | 相似度 | 多数派 |\n|---|---|---|---|---|---|\n")
        for c, en, old, new, sim, n in unified:
            w(f"| {c} | {en} | {old} | {new} | {sim} | {n} |\n")
        if unify_rejected:
            w("\n副本差异过大，未自动统一（需人工判断）：\n\n" +
              "\n".join(f"- {c} ({en}): {old} vs {rep}" for c, en, old, rep in unify_rejected) + "\n")
        w("\n## 跳过的可疑条目\n\n| code | 表内原文 | 原因 |\n|---|---|---|\n")
        for code, text, why in suspects:
            w(f"| {code} | {text} | {why} |\n")
        w("\n## 保留副标题的改名\n\n| code | pack | 主名旧→新 | 保留的 subname |\n|---|---|---|---|\n")
        for code, pack, old, sub, new in sub_kept:
            w(f"| {code} | {pack} | {old} → {new} | {sub} |\n")
        w("\n## 改名明细（按 pack）\n\n")
        cur_pack = None
        for pack, code, old, sub, new in details:
            if pack != cur_pack:
                cur_pack = pack
                w(f"\n### {pack}\n\n| code | 旧名 | 新名 |\n|---|---|---|\n")
            tail = f"（sub 保留：{sub}）" if sub else ""
            w(f"| {code} | {old} | {new}{tail} |\n")
        w("\n## 仓库未覆盖 code 清单\n\n```\n")
        w("\n".join(sorted(repo_uncovered)))
        w("\n```\n")

    print(f"suspects skipped: {len(suspects)}")
    print(f"curated fixes: {len(curated_applied)}, dup-unified: {len(unified)}, unify-rejected: {len(unify_rejected)}")
    print(f"identical: {same_before}, norm-identical skipped: {noop_norm}")
    print(f"renamed codes: {len(details)} across {len(patch)} pack files "
          f"({sum(len(v) for v in patch.values())} field writes)")
    print(f"ocr codes missing in repo: {len(missing_in_repo)} {missing_in_repo}")
    print(f"patch -> {patch_path}")
    print(f"report -> {report_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
