# 英雄引擎实现计划 / Hero Engine Implementation Plan

> 状态：2026-08-21。引擎已实现 21 个英雄面，未实现 51 个（约 46 个独立英雄）。
> 本文档协调骑士团内的实现分工，动手前先认领、动手后打勾，避免撞车。

## 已实现（21）

核心盒：蜘蛛侠、惊奇队长、女绿巨人、钢铁侠、黑豹；美国队长、惊奇少女、奇异博士、格鲁特、火箭浣熊、钢力士、幻影猫、电索、多米诺、天使/大天使、夜行者、虎女、浩克小子、夜魔侠、回声。

## 未实现清单（51 个英雄面）

| 波次 | 卡号 | 英雄 | 包 | 机制要点 | 状态 |
|---|---|---|---|---|---|
| W1 | 06001a | 索尔 Thor | thor | 妙尔尼尔检索，简单数值 | ✅ kaguya+shantu |
| W1 | 10001a | 浩克 Hulk | hlk | 变身增伤，简单数值 | 🔨 kaguya |
| W1 | 04001a | 鹰眼 Hawkeye | trors | 弓/箭矢体系（箭矢=attack/thwart 事件） | ✅ zhuque |
| W1 | 04031a | 蜘蛛女侠 Spider-Woman | trors | 双派系构筑（构筑期规则，引擎内可简化） | ⬜ |
| W2 | 08001a | 黑寡妇 Black Widow | bkw | Preparation 准备异能体系 | ⬜ |
| W2 | 14001a | 快银 Quicksilver | qsv | 抽牌/弃牌循环 | ⬜ |
| W2 | 15001a | 绯红女巫 Scarlet Witch | scw | 混沌控制 | ⬜ |
| W2 | 23001a | 战争机器 War Machine | warm | 弹药标记管理 | ⬜ |
| W2 | 28001a | 新星 Nova | nova | 超新星头盔形态切换 | ⬜ |
| W3 | 50001a | 玛丽亚·希尔 Maria Hill | aos | 战衣形态升级（suit form）体系 | ✅ morgan |
| W3 | 50034a | 尼克·弗瑞 Nick Fury | aos | 战衣形态（突击/潜行切换） | ✅ morgan |
| W3 | 12001a/c | 蚁人 Ant-Man | ant | 多形态（巨人/巨化） | ⬜ |
| W3 | 13001a/c | 黄蜂女 Wasp | wsp | 多形态 | ⬜ |
| W4 | 17001a | 星爵 Star-Lord | stld | 元素枪 | ⬜ |
| W4 | 18001a | 卡魔拉 Gamora | gam | 攻击/化解事件计数 | ⬜ |
| W4 | 19001a | 德拉克斯 Drax | drax | 复仇标记 | ⬜ |
| W4 | 20001a | 毒液 Venom | vnm | 武器限制数+1 | ⬜ |
| W4 | 21001a | 光谱 Spectrum | mts | 能量形态三卡翻转 | ⬜ |
| W4 | 21031a | 亚当术士 Adam Warlock | mts | 构筑限制（四派系均等） | ⬜ |
| W4 | 22001a | 星云 Nebula | nebu | 技巧升级结算体系 | ⬜ |
| W5 | 25001a | 女武神 Valkyrie | valk | 死亡之光 | ⬜ |
| W5 | 26001a | 幻视 Vision | vision | 质量形态（致密/无形） | ⬜ |
| W5 | 27001a | 幽灵蜘蛛 Ghost-Spider | sm | 多元宇宙门票 | ⬜ |
| W5 | 27030a | 蜘蛛侠 Spider-Man | sm | 蛛网发射器体系 | ⬜ |
| W5 | 29001a-29003a | 钢铁之心 Ironheart | ironheart | 三身份+进度标记升级 | ⬜ |
| W5 | 30001a | 蜘蛛侠汉姆 Spider-Ham | spiderham | 卡通标记 | ⬜ |
| W5 | 31001a | SP//dr战甲 | spdr | 佩妮分离/合体 | ⬜ |
| W6 | 33001a | 镭射眼 Cyclops | cyclops | 战术升级+跨派系盟友 | ✅ kaguya |
| W6 | 34001a | 凤凰 Phoenix | phoenix | 凤凰之力力量标记 | ✅ kaguya |
| W6 | 35001a | 金刚狼 Wolverine | wolv | 爪+自愈 | ✅ kaguya |
| W6 | 36001a | 暴风女 Storm | storm | 天气牌堆 | ✅ morgan |
| W6 | 37001a | 牌皇 Gambit | gambit | 盗贼检视 | ⬜ |
| W6 | 38001a | 罗刹女 Rogue | rogue | 触碰升级 | ⬜ |
| W7 | 41001a | 灵蝶 Psylocke | psylocke | 灵能双升级翻转 | ⬜ |
| W7 | 43001a | X-23 | x23 | 爪+洗回循环 | ⬜ |
| W7 | 44001a | 死侍 Deadpool | deadpool | 第四面墙检索 | ⬜ |
| W7 | 45001a | 毕肖普 Bishop | aoa | 时间牌堆 | ✅ morgan |
| W7 | 45030a | 秘客 Magik | aoa | 法术牌堆 | ✅ morgan |
| W8 | 46001a | 冰人 Iceman | iceman | 冻伤升级 | ✅ morgan |
| W8 | 47001a | 千欢 Jubilee | jubilee | 购物密谋 | ✅ morgan |
| W8 | 49001a | 万磁王 Magneto | magneto | 磁力体系 | ✅ morgan |
| W8 | 51001a | 黑豹 Black Panther | bp | 发明家检索 | ✅ zhuque |
| W8 | 52001a | 丝 Silk | silk | 塞卡机制 | ⬜ |
| W8 | 53001a | 猎鹰 Falcon | falcon | 鸟卡体系 | ⬜ |
| W8 | 54001a | 冬日战士 Winter Soldier | winter | 机械臂 | ⬜ |
| W8 | 58001a | 奇迹人 Wonder Man | wonder_man | 离子生理 | ⬜ |
| W8 | 59001a | 赫拉克勒斯 Hercules | hercules | 试炼/礼物双牌堆 | ⬜ |

## 实现规范（每个英雄）

1. **新包**：`internal/engine/cards/<pack>/`（已存在的包直接加文件）
   - `<hero>.go`：身份行为 `engine.RegisterBehavior("<base>", &engine.Behavior{...})`
   - `signatures.go`：专属卡行为
   - 宿敌/重担按现有包的模式（参考 `msmarvel/`）
2. **Behavior 钩子**（见 `internal/engine/entity.go`）：
   - `HeroAbilities` / `AlterEgoAbilities`：主动异能
   - `React`：响应/打断（监听 engine.Message）
   - `CardCost` / `IdentityStats` / `Resource` 等被动钩子
   - 玩家抉择一律用 `engine.AskQuestion` + `engine.Choice`
3. **注册**：`init()` 里注册；`cmd/server/main.go` 加 blank import
4. **测试**：`<pack>_test.go` 覆盖核心异能触发（参考 `msmarvel/msm_test.go`）
5. **验证**：`go build ./... && go test ./internal/engine/...`，本地起服务确认 `/api/v1/marvel/cards` 无回归
6. **卡牌文本**：翻译层已覆盖（tools/zh/out），行为代码里引用卡名用英文原文（引擎内部），显示层自动中文化

## 分工约定

- 动手前把状态列 ⬜ 改成 🔨（你的名字），完成后改 ✅
- 每个英雄一个独立提交，提交信息 `feat(hero): implement <name> (<code>)`
- 推送前 `git fetch origin && git rebase origin/main`（dynilath 活跃开发中）
- W1-W2 由 shantu + kaguya 并行推进；W3 aos 双英雄优先（玩家点名要玩）
