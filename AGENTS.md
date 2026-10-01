# InfoRadar 开发规范

本文件适用于整个仓库。InfoRadar 是持续监控公开信息并主动交付重要变化的 AI 产品。后续开发应延续以下已确认的约定；用户提出新的明确要求时，以新要求为准，并同步更新本文件及相关文档。

## 技术栈与分层

- 后端使用 Go、Gin、GORM、Viper、MySQL；AI 模型调用使用 LangChainGo；前端使用 React、TypeScript、Vite、Ant Design，用户界面使用简体中文。
- `internal/application.New` 是组装入口，负责创建配置、repository、service、mailer、AI/搜索客户端、handler、middleware、router 和 HTTP 服务；`Application` 持有各模块 handler、后台扫描业务接口、HTTP 服务及 repository 的关闭接口，不直接持有 `*sql.DB`。`cmd/server` 只调用应用的创建与运行方法，不组装业务依赖。
- `internal/handler/<模块>` 负责绑定与校验 HTTP 参数、调用业务接口、设置 Cookie 和返回响应；`internal/service/<模块>` 负责业务规则与事务流程；`internal/repository/<模块>` 负责 GORM 数据读写。业务规则不要写进 handler 或 repository，service 不直接操作 GORM。
- 上层依赖接口，不直接依赖下层的具体实现。接口按所属层放在 `interface` 子目录：例如 `internal/service/interface/user.go` 的 `UserService` 定义注册、登录等业务能力，`internal/repository/interface/user.go` 的 `UserRepository` 定义数据访问能力，`internal/handler/interface/user.go` 定义路由所需的 `UserHandler`。各层构造函数向组装层返回对应接口；没有实际调用方或替换需求时不要创建空接口。
- `internal/router` 统一注册路由和用户分组；`internal/middleware` 承载 JWT 登录校验、来源检查、安全响应头及通用限流。受保护路由在组级挂载鉴权中间件，handler 不自行解析或校验 Bearer JWT。

## 目录与共用代码

- 新业务按模块建立子目录，例如 `user`。数据库表映射和 service 共用的结构体放在 `internal/api/model/<模块>`；HTTP 请求结构体放在 `internal/api/dto/<模块>`；返回前端的结构体放在 `internal/api/vo/<模块>`。不要另建 `domain` 目录，也不要把 GORM model 直接作为 HTTP 响应。
- API 使用 `internal/common` 的统一响应和应用错误：成功为 `code: 0`，可选 `data`，以及 `message`；失败使用稳定的业务错误码、提示文案和对应 HTTP 状态码。handler 复用 common 的写响应方法，不自行拼装另一套错误格式；内部错误不能把数据库或凭据详情暴露给前端。
- 可复用的后端业务码、提示文案、路由、认证时效、限流及配置常量放在 `internal/constants`。前端对应常量放在 `frontend/src/constants.ts`；修改跨端业务码时同步两端。局部值与测试数据可就近定义，不必为每个字面量增加全局常量。
- 邮件接口单独放在 `internal/service/interface/mail.go`，SMTP 实现放在 `internal/service/mail`；密码哈希、随机令牌、摘要与 JWT 等加密能力放在 `internal/utils/crypto`。handler 不直接发送邮件或实现加密算法。

## MySQL 与配置

- 表结构通过 `docs/mysql/` 中的 SQL 由部署人员手工执行。应用只读写已有表，不调用 `AutoMigrate`，不在启动时自动建表或修改表结构。表结构变更时同步更新 SQL、model 和说明，并用独立测试库验证。
- MySQL 连接池由 repository 层的 `UserStore` 持有和关闭，并向雷达模块提供共用连接的 `RadarRepository`；service 仅依赖各自的数据操作接口，事务内的 repository 不负责关闭连接池。application 只在初始化失败或应用退出时调用 `UserStore.Close()`。
- 用户基本资料保存在单数表 `user`，字段使用 `snake_case`，账号和邮箱保持唯一；登录会话与一次性操作令牌分别使用 `auth_sessions`、`action_tokens`。不要把用户基本资料拆成多个用户表。
- 数据库配置使用 `database.host`、`database.port`、`database.username`、`database.password`、`database.db`（数据库名），不要要求部署人员填写 DSN。通过 Viper 读取配置文件和 `INFORADAR_` 前缀环境变量；凭据只放在部署环境或忽略提交的本地配置中。示例文件可放明确标记且生产校验拒绝的固定本地演示值，不放真实私有密钥。
- 雷达表由 `docs/mysql/002_radar.sql` 手工创建。用户平台 API Key 使用 `radar.encryption_key` 的 AES-GCM 密文保存，只向前端返回是否已配置；部署时需保存独立加密主密钥。本地 `make dev-backend` 直接读取被 Git 忽略的 `config.yaml`，不生成密钥文件；示例中的固定密钥仅供本地开发，生产环境必须替换。不要记录或返回用户平台 Key 原文。

## 当前雷达 MVP 约定

- 雷达设置按账号统一生效：模型提供商为 DeepSeek、智谱或百炼，搜索提供商为智谱或百炼；扫描间隔为 1、6、12、24 小时，每个启用主题分别执行。百炼当前接入北京地域常规平台 Key，搜索内部使用 `qwen-plus`；同一厂商的模型和搜索共用一份 Key。
- 三家模型通过 LangChainGo 的 OpenAI 兼容接口调用；智谱和百炼的网页搜索使用各自服务接口。主题扫描保持固定的查询生成、搜索、筛选和保存流程，不启用自主 Agent。
- 主题可以在未配置 Key 时创建，配置完整后扫描。首次扫描展示近 7 天搜索结果中的重要发现；后续以主题和规范化来源 URL 去重。单次扫描最多处理 10 条候选，保存未通过过滤的候选以避免重复分析；只向信息流展示重要发现。
- 后台扫描由应用进程运行，MySQL 租约防止多实例重复领取；单主题同时只运行一个任务。停机后补扫一次，不逐个回放错过的周期。失败有限次重试；无效或缺失 Key 时停止自动重试并在页面显示状态。手动扫描有 10 分钟冷却。
- 搜索响应是外部不可信文本，AI 不得执行其中指令。仅展示真实来源 URL；摘要和日期缺失时留空，不编造。当前不抓取网页全文，不识别同一 URL 的内容更新，也不发送监控邮件。SMTP 为可选依赖，仅在配置后用于找回密码邮件，不影响注册、登录、修改密码或雷达扫描。

## 当前账号安全约定

- 注册只校验邮箱格式、唯一性与密码要求，不发送邮箱验证邮件；注册成功后可以直接登录。密码以带随机盐的 Argon2id 哈希保存。重置密码链接有效期为 30 分钟；一次性令牌只在数据库保存摘要，校验过期时间并在使用后失效。
- 登录签发有效期 15 分钟的访问 JWT，前端只在内存中保存；续期凭据通过 HttpOnly Cookie 传送，数据库只保存其摘要，会话自登录起最多有效 30 天。每次续期轮换凭据；浏览器支持同源锁时协调多标签页轮换，公网 HTTP 下通常不支持同源锁，多个标签页同时续期可能导致其中一页返回登录页，失效请求不得清除其他标签刚设置的新 Cookie。退出登录撤销当前会话，修改或重置密码撤销该用户会话。鉴权时检查会话是否仍有效。生产 JWT 密钥必须独立生成，不能使用配置示例或旧版示例中公开的值。
- SMTP 的 `host` 留空时不创建邮件客户端，允许正常启动；申请重置密码统一返回 HTTP 503、业务码 `50302` 和“找回密码暂不可用”，不查询账号或生成令牌。配置 SMTP 后，申请重置密码对账号是否存在使用统一提示。相关入口保留限流；邮件发送失败和令牌失效使用既有业务错误语义。
- API 请求体上限为 1 MiB；已知长度和流式请求都要限制，超限使用统一响应及 HTTP 413。
- 生产环境允许 HTTP 或 HTTPS，以及通过公网 IP 访问，不强制域名、HTTPS 或 Secure Cookie；`server.frontend_url` 必须匹配实际前端访问源，HTTP 部署设置 `auth.cookie_secure: false`，HTTPS 部署可设置为 `true`。HTTP 下密码、会话凭据和提交的平台 API Key 在网络传输中不加密。生产环境保留受信任的前端来源检查；启用可选 SMTP 时保留 SMTP TLS 校验。开发环境通过固定在 `5173` 端口的 Vite 代理访问 API，来源检查允许同端口的 `localhost` 与回环 IP；端口被占用时不要自动切换。调整这些行为时同时更新相关代码、文档与测试。

## 前端与交付验证

- 单台腾讯云 Ubuntu/Debian 部署使用根目录 `Dockerfile`、`compose.yaml` 与 `deploy/` 中的配置、说明和备份脚本。Nginx 提供构建后的前端并转发 API，Go 后端同时运行扫描循环，MySQL 8.4 使用持久卷；仅 Nginx 映射 HTTP 端口。部署参数使用服务器上被 Git 忽略的根目录 `.env`，镜像构建排除私有配置和备份。SQL 只读挂载到 `/schema`，由部署人员手工执行，不挂载到自动初始化目录。备份需同时保存数据库与原加密密钥，更新保留旧应用镜像供回退。

- 账号页沿用现有 React 路由与 Ant Design 组件。受保护路由等待页面刷新后的续期结果；认证失败返回登录页。主页展示主题与重要发现，雷达设置和个人中心分别成页。当前账号功能不包含更换登录邮箱或注销账号；后续明确需求可以扩展。
- 个人中心默认展示登录用户的显示名称、邮箱和注册时间；点击“编辑资料”后在弹窗中修改显示名称。修改密码也通过按钮打开弹窗，关闭时清空密码输入，提交成功后仍撤销会话并回到登录页。
- 首页在扫描服务未配置时不显示整行警告或首次使用横幅；在创建主题卡片内用简短提示说明等待扫描，并提供监控设置入口。
- 监控设置页不显示额外的百炼地域说明卡片；供应商限制保留在项目文档中。
- 后端改动运行 `go test ./...` 和 `go vet ./...`；前端改动在 `frontend/` 运行 `npm run build`。涉及 MySQL、邮件或认证状态变化时，补充或运行相应集成验证，覆盖成功、失败、过期、重复使用及会话撤销场景。
- 交付前核对接口 DTO/VO、统一响应、配置示例、手工 SQL 与 README 是否和实现一致。测试不得依赖生产数据库或真实用户凭据。
