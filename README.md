# Marvel Champions Next

[Marvel Champions: The Card Game](https://fantasyflightgames.com/en/products/marvel-champions-the-card-game/) 的全栈 Web 实现：Go 游戏引擎 + REST/WebSocket 服务 + React 前端。

- 卡牌数据来自 [marvelcdb.com](https://marvelcdb.com) 公开 API（61 个扩展包 / 4000+ 张卡，快照内置于仓库）
- 卡图按需从 marvelcdb 拉取并缓存到本地，hash URL 永久缓存
- 内置 5 个可玩场景：Rhino、Klaw、Ultron、Green Goblin ×2；未手写实现的卡牌走通用兜底
- 支持最多 4 人对局、观战、撤销、回放

## 快速开始

本地开发：

```bash
go run ./cmd/server                # 后端 :3000
cd web && npm install && npm run dev   # 前端 :8080
```

打开 http://localhost:8080，注册后在 Decks 页粘贴 marvelcdb 卡组链接导入，即可开局。

Docker 部署：

```bash
cd deploy && docker compose up --build -d   # http://localhost:3000
```
