# NovelGenerater

独立小说智能生成工作台。

当前目标是把既有小说生成流程完整沉淀到这里，保留：

- 素材图谱与文风分析
- 大纲分批生成与后台续写
- 章节正文生成
- 审计、修订、确认
- 项目级 `skills`、规则文件与 AI provider 配置

目录约定：

- `apps/web`：Vue 3 独立前端
- `apps/api`：Go 独立后端
- `.agents/skills`：项目级 skills
- `docs`：迁移与架构文档

本地开发：

```bash
cd /Users/apple/NovelGenerater
./start.sh
```

如需临时换端口：

```bash
NOVEL_GENERATER_BACKEND_PORT=19081 NOVEL_GENERATER_WEB_PORT=15174 ./start.sh
```

如果默认端口被占用，启动脚本会自动选择备用端口，并在终端与 `logs/start-*.log` 中输出最终访问地址。
当前实际端口也会写入 `.tmp/current-dev.env`，浏览器地址以这里的 `NOVEL_GENERATER_WEB_URL` 为准。

当前本地开发默认隐藏登录模块；如需恢复登录校验：

```bash
NOVEL_GENERATER_AUTH_DISABLED=0 VITE_NOVEL_GENERATER_AUTH_DISABLED=0 ./start.sh
```

也可以分别启动：

```bash
cd /Users/apple/NovelGenerater/apps/api
NOVEL_GENERATER_BACKEND_PORT=19081 go run .

cd /Users/apple/NovelGenerater/apps/web
NOVEL_GENERATER_BACKEND_PORT=19081 NOVEL_GENERATER_WEB_PORT=5174 npm run dev
```

常用验证：

```bash
cd /Users/apple/NovelGenerater/apps/api
go test ./...

cd /Users/apple/NovelGenerater/apps/web
npm run build
```
