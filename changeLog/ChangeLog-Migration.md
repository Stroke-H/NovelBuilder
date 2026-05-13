# ChangeLog - Migration

## 2026-05-13

### 新增

- 新建 `NovelGenerater` 独立项目骨架，开始从 `TestCenter_Vue3` 剥离小说智能生成流程。
- 迁入小说前端工作台、素材图谱、章节生成、审计和修订相关页面与组件。
- 迁入小说后端的真实提示词、AI provider 选择逻辑、SQL JSON 存储、分批大纲生成与后台续生成逻辑。
- 迁入项目级 `.agents/skills` 与 `open-novel-writing` 相关能力说明。
- 新增独立登录/注册入口与新项目会话命名。

### 调整

- 将项目规则文件改写为面向 `NovelGenerater` 的独立协作规则。
- 将前端默认开发端口调整为 `5174`。
- 将后端开发端口调整为更少冲突的 `19081`，并同步前端代理配置。
- 将本地真实 `ai_config.json`、`database_config.json` 迁入 `apps/api/data/`，并通过 `.gitignore` 阻止提交。

### 验证

- `cd /Users/apple/NovelGenerater/apps/api && go test ./...`
- `cd /Users/apple/NovelGenerater/apps/web && npm run build`
