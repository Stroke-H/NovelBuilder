# 从 TestCenter_Vue3 迁移说明

## 目标

把 `TestCenter_Vue3` 中的小说智能生成模块完整剥离到 `NovelGenerater`，保留真实生成能力，而不是做一个裁剪后的演示版。

## 本次迁移保留的核心能力

1. 素材提取
2. 文风分析
3. 大纲首批生成
4. 后续章节大纲后台续生成
5. 章节正文生成
6. 审计
7. 修订
8. 确认章节并写回长期记忆

## 已迁入的关键资产

### 前端

- `apps/web/src/views/NovelWriter/index.vue`
- `apps/web/src/components/NovelWriter/*`
- `apps/web/src/api/novelWriter.ts`
- `apps/web/src/api/request.ts`
- `apps/web/src/stores/auth.ts`
- `apps/web/src/views/Login/index.vue`
- `apps/web/src/views/Register/index.vue`

### 后端

- `apps/api/services/novel_writer_service.go`
- `apps/api/services/novel_writer_prompts.go`
- `apps/api/services/sql_json_store.go`
- `apps/api/services/auth_service.go`
- `apps/api/services/session_service.go`
- `apps/api/services/database_handlers.go`
- `apps/api/database/*`
- `apps/api/model/ai_config.go`

### 配置与规则

- `.cursorrules`
- `.agents/skills/*`
- `apps/api/data/ai_config.example.json`
- `apps/api/data/ai_config.json`（本地保留，不可提交）
- `apps/api/data/database_config.json`（本地保留，不可提交）

## 新项目结构

```text
NovelGenerater/
  .agents/skills/
  apps/
    api/
      data/
      database/
      model/
      services/
      main.go
    web/
      src/
        api/
        components/NovelWriter/
        router/
        stores/
        views/
  docs/
```

## 迁移中的适配项

1. Go module 从 `testcenter-server` 改为 `novel-generater-api`
2. 后端默认端口改为 `19081`
3. 前端默认开发端口改为 `5174`
4. 登录 cookie 改为 `novel_generater_session`
5. 本地登录提示 key 改为 `novel_generater_session_hint`
6. `.cursorrules` 改为独立小说应用语境

## 当前验证结果

- `apps/api`: `go test ./...`
- `apps/web`: `npm run build`

## 后续建议

1. 把 `apps/web/node_modules` 从临时本地依赖挂载切换成独立安装
2. 继续把后端从 `services` 单层拆到更清晰的 `application / repository / workflow`
3. 增加大纲与章节生成的任务表和 SSE 进度推送
