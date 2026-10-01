# InfoRadar

InfoRadar 是一款持续监控公开信息的 AI 产品。用户可以创建多个监控主题，选择自己的分析模型与搜索服务，按固定间隔扫描，并在个人主页查看 AI 筛选的重要发现。现支持 DeepSeek、智谱、阿里百炼模型，以及智谱和百炼网页搜索。账号模块包含邮箱注册、登录、续期、退出、找回和修改密码。

## 技术栈

- API：Go 1.24、Gin、GORM、Viper、MySQL 8.0+
- AI 模型：LangChainGo，通过兼容 OpenAI 的接口调用 DeepSeek、智谱和阿里百炼
- 前端：React 19、TypeScript、Vite 8、Ant Design 6
- 邮件：通用 SMTP，支持 STARTTLS 或隐式 TLS；本地测试 SMTP 可使用无 TLS 模式

## 启动

1. 在全新 MySQL 数据库中手工创建表。应用不会运行迁移、自动建表或修改表结构；已有旧版 `users` 表不能通过重复执行 `CREATE TABLE IF NOT EXISTS` 更新：

   ```bash
   mysql -u YOUR_MYSQL_USER -p < docs/mysql/001_init.sql
   mysql -u YOUR_MYSQL_USER -p < docs/mysql/002_radar.sql
   ```

2. 执行 `cp config.example.yaml config.yaml`，分别填写 MySQL 的 `host`、`port`、`username`、`password`、`db`（数据库名）和 SMTP 参数。示例中的 `auth.jwt_secret` 与 `radar.encryption_key` 是固定的本地开发值；生产环境须在私有配置或部署环境中替换为各自独立的密钥。`config.yaml` 已被 Git 忽略。配置也可通过 `INFORADAR_` 前缀环境变量覆盖，例如 `INFORADAR_DATABASE_USERNAME`、`INFORADAR_DATABASE_PASSWORD`、`INFORADAR_DATABASE_DB`、`INFORADAR_AUTH_JWT_SECRET`；`INFORADAR_CONFIG` 可指定配置文件路径。
3. 在仓库根目录启动 API：

   ```bash
   make dev-backend
   ```

4. 在另一个终端、仓库根目录启动前端（首次启动会自动安装依赖）：

   ```bash
   make dev-frontend
   ```

   打开 `http://127.0.0.1:5173`。Vite 固定使用 `5173`；如果本项目的前端已经运行，`make dev-frontend` 会提示现有地址并正常退出；如果是其他程序占用端口，会提示占用进程。Vite 将 `/api` 代理到 `http://127.0.0.1:8080`。开发环境也接受同端口的 `localhost` 来源；生产环境只接受 `server.frontend_url` 指定的来源。注册不发邮件；找回密码仍需 SMTP。示例配置中的 SMTP `none` 仅用于本地测试服务器；生产环境必须使用 `starttls` 或 `implicit`。

`make dev-backend` 直接读取 `config.yaml`，不生成密钥文件。生产环境分别生成并保存独立的 `radar.encryption_key` 和 `auth.jwt_secret`（例如用 `openssl rand -hex 32` 生成），公开的开发示例值和旧版 JWT 示例值都会被生产配置校验拒绝。`radar.encryption_key` 用于解密已保存的平台 API Key，更换它后需要重新填写这些 Key。

首次启动时 API 会只读检查八张表是否存在，缺表会提示先手工执行 SQL。生产环境如部署在反向代理后，请将可信代理的 IP 或 CIDR 填入 `server.trusted_proxies`，并使 `server.frontend_url` 与浏览器实际访问源完全一致。

生产环境支持通过公网 IP 使用 HTTP，无需域名或 HTTPS 证书。将 `server.environment` 设为 `production`，`server.frontend_url` 设为实际前端地址（例如 `http://203.0.113.10`，使用非默认端口时需包含端口），并设置 `auth.cookie_secure: false`，使浏览器能在 HTTP 请求中发送续期 Cookie。前端静态文件和 `/api` 应通过同一访问源提供；密码重置邮件也使用该前端地址。使用 HTTPS 时可设置 `auth.cookie_secure: true`。HTTP 下密码、会话凭据和提交的平台 API Key 在网络传输中不加密。

登录后到“监控设置”选择模型、网页搜索服务、扫描间隔，并填写对应厂商的常规平台 API Key；主题可以先创建，配置完整后会自动开始扫描。智谱模型和搜索共用智谱 Key，百炼模型和搜索共用北京地域的百炼 Key；不同厂商 Key 不互通。百炼搜索内部调用 `qwen-plus`，可能同时产生模型与搜索费用。系统不发送监控邮件，找回密码邮件仍使用 SMTP。

“个人中心”展示当前登录用户的显示名称、邮箱和注册时间。点击“编辑资料”可修改显示名称；点击“修改密码”可在弹窗中更新登录密码。

## API

所有接口以 `/api/v1` 为前缀，请求和响应使用 JSON。请求体上限为 1 MiB，超过时返回 HTTP 413 和统一错误码 `41300`。成功响应为 `{ "code": 0, "data": ..., "message": "ok" }`；失败响应为 `{ "code": 40000, "message": "请检查表单内容" }` 等形式。`data` 为空时省略该字段，错误同时使用对应的 HTTP 状态码。统一响应和应用错误定义在 `internal/common/`，设计参考 [ai-passage-creator 的 common 实现](https://github.com/yuyuanweb/ai-passage-creator/tree/43347270a4cc319195dfcffeaa9bc9c32557b85b/go-backend/internal/common)。

| 方法 | 路径 | 请求数据 | 结果 |
| --- | --- | --- | --- |
| POST | `/auth/register` | `email`, `password` | 校验邮箱格式并注册，无需邮箱验证 |
| POST | `/auth/login` | `email`, `password` | 返回访问 JWT 和用户资料；设置续期 Cookie |
| POST | `/auth/refresh` | 无 | 轮换续期凭据，返回新访问 JWT |
| POST | `/auth/logout` | 无 | 撤销当前会话并清除 Cookie |
| POST | `/auth/forgot-password` | `email` | 发送重置链接 |
| POST | `/auth/reset-password` | `token`, `newPassword` | 重置密码并撤销全部会话 |
| GET | `/users/me` | Bearer JWT | 读取个人资料 |
| PATCH | `/users/me` | Bearer JWT、`displayName` | 修改显示名称 |
| POST | `/users/me/password` | Bearer JWT、`currentPassword`, `newPassword` | 修改密码并撤销全部会话 |
| GET/PUT | `/radar/settings` | Bearer JWT；PUT 传 `modelProvider`, `modelId`, `searchProvider`, `intervalHours` | 读取/保存账号级监控配置与密钥配置状态 |
| PUT/DELETE | `/radar/keys/:provider` | Bearer JWT；PUT 传 `apiKey` | 保存/删除平台密钥；响应不返回密钥原文 |
| GET/POST | `/radar/topics` | Bearer JWT；POST 传 `description` | 列出/创建主题 |
| PATCH/DELETE | `/radar/topics/:id` | Bearer JWT；PATCH 传 `description` 或 `status` | 编辑、暂停、恢复、删除主题 |
| POST | `/radar/topics/:id/scan` | Bearer JWT | 手动排队扫描，冷却 10 分钟 |
| GET | `/radar/topics/:id/runs` | Bearer JWT | 最近 20 次扫描记录 |
| GET | `/radar/feed` | Bearer JWT；可传 `topicId`, `before`, `limit` | 分页读取重要发现 |

用户与雷达的 model、请求 DTO、响应 VO 分别在 `internal/api/model/<模块>/`、`internal/api/dto/<模块>/`、`internal/api/vo/<模块>/`。`internal/common/` 统一响应结构和业务错误；`internal/constants/` 集中保存后端业务常量；handler 校验参数并返回响应，service 处理业务和扫描，repository 负责 GORM 读写，各层在 `interface/` 定义被上层使用的接口。`internal/repository/user/` 持有共享 MySQL 连接池，并提供雷达 repository；`internal/application.New` 组装所有模块，应用运行时启动后台扫描循环。`internal/router/` 分组注册带 JWT 中间件的接口；`internal/service/provider/` 使用 LangChainGo 调用三家模型，智谱和百炼的网页搜索仍调用各自服务接口；`internal/utils/crypto/` 加密平台密钥。主题扫描保持现有的固定流程，并未启用自主 Agent。

用户资料只存放在一张 `user` 表，字段使用下划线命名，包含账号、邮箱、密码哈希、昵称、头像、简介、角色、时间和删除标记。当前仅支持邮箱注册，因此 `user_account` 写入规范化后的邮箱；`auth_sessions` 和 `action_tokens` 分别保存登录会话与密码重置令牌。密码使用 Argon2id 加盐哈希。访问 JWT 有效期 15 分钟，只保存在浏览器内存；续期凭据以摘要形式保存在数据库，并通过 HttpOnly Cookie 传送，会话自登录起最多有效 30 天。前端在浏览器支持同源锁时协调多标签页续期；公网 HTTP 下浏览器通常不提供该能力，建议使用单个标签页，多个标签页同时续期可能导致其中一页返回登录页。失效的续期请求不会清除可能已轮换的新 Cookie。密码重置链接有效 30 分钟；令牌只保存 SHA-256 摘要，使用后失效。

## 验证

```bash
go test ./...
go vet ./...
cd frontend && npm run build
```

MySQL 完整流程测试需要一个**独立的测试库**。将两份 SQL 中的 `inforadar` 数据库名改为测试库名后手工执行，再设置 `INFORADAR_TEST_DATABASE_HOST`、`INFORADAR_TEST_DATABASE_PORT`、`INFORADAR_TEST_DATABASE_USERNAME`、`INFORADAR_TEST_DATABASE_PASSWORD`、`INFORADAR_TEST_DATABASE_DB`，运行 `go test ./internal/service/user ./internal/repository/radar -run TestMySQL`。未设置测试库名称时集成测试会跳过。普通测试使用模拟厂商 HTTP 服务覆盖模型请求、搜索响应、密钥拒绝、限流、加密及重要发现去重；不会调用真实付费 API。

MVP 只分析搜索服务返回的标题、摘要等字段，不抓取网页全文；相同 URL 的内容更新暂不识别。扫描间隔按账号统一配置，但每个启用主题分别运行。服务停机期间错过的周期不会逐次补跑，重启后对逾期主题补扫一次。
