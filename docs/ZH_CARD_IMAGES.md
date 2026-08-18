# 中文卡图接入指南 / Chinese Card Images

MarvelChampionsNext 默认从 marvelcdb.com 拉取**英文**卡图。本仓库附带一个
播种脚本，可以把社区懒人包里的**简体中文卡面**灌进图片缓存，让游戏直接
显示中文卡面（缓存命中的卡不再请求 marvelcdb）。

## 中文卡图来源

来自中文社区分享的「漫威网页版懒人包（打开即玩）」网盘压缩包（约 3.4GB，
irefrixs Marvel Champions: Digital Edition 的中文社区发行版）。包内：

```
漫威网页版/
  assets/pics/     ← 3521 张简中卡图，按 MarvelCDB 卡号命名（01001a.jpg 等）
  data/cards.json  ← irefrixs 格式全量卡数据（英文文本）
  data/translate_cn.json ← 游戏界面/日志简中翻译（UI 字符串，非卡牌文本）
  data/scenarios/  ← 剧本定义
```

卡图为粉丝重制的简中卡面（FAN ARTIFACT DESIGN），卡号与 marvelcdb 完全
对齐。约 3489 张可被本项目识别（其余为图标等杂图）。

## 如何播种

解压懒人包后：

```sh
python tools/seed_zh_images.py <懒人包>/漫威网页版/assets/pics
# 默认写入 ./cache/images（与 cmd/server/main.go 的缓存路径一致）
# 也可指定：python tools/seed_zh_images.py <pics_dir> <cache_dir>
```

脚本按 `internal/api/images.go` 的缓存格式写入 `{code}.img` +
`manifest.json`（sha256 前 16 位）。重复运行幂等，只覆盖内容变化的文件。

## 覆盖情况

- 已覆盖：3489 个卡号（核心盒～约 2024 年扩展的大部分卡牌）
- 未覆盖的卡号：服务器按原逻辑回落到 marvelcdb 英文图，不影响游玩
- 卡牌**文本数据**仍是英文（marvelcdb 快照）；卡名/效果文本的中文翻译
  是独立的后续工作（见 AI 翻译计划）

## 部署注意

- 缓存目录不进 git（已忽略），服务器上需单独播种或用挂载卷
- Docker 部署时建议把播种好的 `cache/images` 挂到容器 `MC_CACHE_DIR`
