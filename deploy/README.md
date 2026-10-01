# 腾讯云单机部署

适用于一台没有现有网站和数据库的 Ubuntu/Debian 服务器，通过 `http://公网IP` 访问。建议初期使用 2 核、4 GB 内存和至少 40 GB 磁盘；这是起步配置建议。

服务器运行三个容器：`web`（Nginx 和已构建的前端）、`api`（Go 后端与扫描循环）、`mysql`（MySQL 8.4）。Dockerfile 在构建阶段运行 Go 测试、`go vet` 和前端构建，服务器不需要另外安装 Go 或 Node.js。模型和搜索调用外部 API；SMTP 可省略，仅在启用找回密码时使用，不提供监控邮件推送。

## 新增文件

| 文件 | 作用 |
| --- | --- |
| `Dockerfile` | 多阶段构建；`api` 目标生成带 CA 证书的非 root 后端镜像，`web` 目标生成前端静态文件与 Nginx 镜像。 |
| `.dockerignore` | 从镜像构建上下文排除私有配置、密钥、备份、Git 和本地构建产物。 |
| `compose.yaml` | 管理三个容器、数据库持久卷、启动依赖、健康检查、日志轮转和自动重启，只映射前端 HTTP 端口。 |
| `deploy/.env.example` | 服务器配置模板；复制到仓库根目录 `.env`，填写 IP、数据库密码、JWT 密钥和雷达加密密钥，SMTP 默认留空。 |
| `deploy/nginx.conf.template` | 提供静态文件、React 路由刷新、保留路径的 API 代理和后端健康检查；从容器 DNS 动态解析后端。 |
| `deploy/backup.sh` | 手动导出数据库，并同时保存 `.env`、代码版本和镜像信息，便于恢复或回退。 |
| `deploy/README.md` | 本部署、更新、备份和恢复说明。 |

## 1. 准备服务器与网络

准备 SSH 登录方式和公网 IP。默认无需邮箱或 SMTP 服务；注册、登录、修改密码和雷达扫描都能使用，申请找回密码会提示“找回密码暂不可用”。

以后需要找回密码时，再填写支持 SMTP 的邮箱账号和授权密码。SMTP 的 TLS 独立于网站 HTTP，通常使用 465 + `implicit` 或 587 + `starttls`。腾讯云默认限制 25 端口，参见[端口说明](https://cloud.tencent.com/document/product/213/2502)。

腾讯云 CVM 在安全组中放通 TCP 80，轻量服务器在其防火墙中放通 TCP 80；SSH 22 允许管理来源。出站需要访问 HTTPS API、镜像与软件仓库；仅在启用 SMTP 时需要所选邮件端口。后端 8080、数据库 3306 通过容器网络连接，不需要映射到服务器公网。

本方案默认占用 HTTP 80 端口。若要改为 8081，需要同时将 `.env` 中的 `INFORADAR_HTTP_PORT` 改为 `8081`、`INFORADAR_SERVER_FRONTEND_URL` 改为 `http://公网IP:8081`，并放通 TCP 8081。

## 2. 安装 Docker 与 Compose

以下命令在服务器执行。支持 Ubuntu 22.04/24.04/26.04、Debian 12/13；根据实际系统选择对应的官方 apt 仓库。系统安装了其他容器运行时或旧 Docker 时，先参考官方文档处理包冲突：[Ubuntu](https://docs.docker.com/engine/install/ubuntu/)、[Debian](https://docs.docker.com/engine/install/debian/)。

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl git openssl
. /etc/os-release
case "$ID" in
  ubuntu|debian) ;;
  *) printf '请按 Docker 官方文档安装当前发行版。\n' >&2; exit 1 ;;
esac
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL "https://download.docker.com/linux/$ID/gpg" -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc
printf 'deb [arch=%s signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/%s %s stable\n' \
  "$(dpkg --print-architecture)" "$ID" "${UBUNTU_CODENAME:-$VERSION_CODENAME}" \
  | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo systemctl enable --now docker
sudo docker compose version
```

后续 Docker 命令假设当前 SSH 用户有 Docker 访问权限；否则在 `docker` 前加 `sudo`。运行备份脚本时可使用 `sudo sh deploy/backup.sh`。普通账号需要无 sudo 管理 Docker 时，可按[官方安装后说明](https://docs.docker.com/engine/install/linux-postinstall/)加入 `docker` 组，并重新登录。

## 3. 上传项目并填写配置

确保部署文件已包含在服务器上的项目中：先提交并推送这些文件再通过 Git 拉取，或直接上传包含这些文件的源码。不要上传开发环境的 `config.yaml`、`.env` 或 `node_modules`。

```bash
git clone https://github.com/wyya666/inforadar.git
cd inforadar
cp deploy/.env.example .env
chmod 600 .env
nano .env
```

填写以下配置：

| 配置 | 内容 |
| --- | --- |
| `INFORADAR_SERVER_FRONTEND_URL` | 浏览器实际访问的地址，例如 `http://你的公网IP`，不带路径。 |
| `MYSQL_ROOT_PASSWORD` | MySQL 管理密码，仅用于初始化、手工 SQL 与备份。 |
| `INFORADAR_DATABASE_PASSWORD` | `inforadar` 应用账号的数据库密码。 |
| `INFORADAR_AUTH_JWT_SECRET` | 独立生成的 JWT 密钥。 |
| `INFORADAR_RADAR_ENCRYPTION_KEY` | 独立生成的 64 位十六进制加密密钥，后续更新保留原值。 |
| `INFORADAR_SMTP_*` | 可选；保持 `INFORADAR_SMTP_HOST` 为空即可禁用找回密码邮件，需要时再填写完整邮箱参数。 |
| `INFORADAR_IMAGE_TAG` | 本次应用镜像版本，例如 `20261001-1`，更新时换一个新版本。 |

运行下面的命令四次，将四个不同的结果分别填入两个数据库密码、JWT 密钥和雷达加密密钥：

```bash
openssl rand -hex 32
```

SMTP 参数默认保持模板值即可，其中 `INFORADAR_SMTP_HOST` 为空，程序不会创建邮件客户端。需要启用时填写 host、port、username、password、from 和 tls_mode；发件人可填写 `InfoRadar <你的邮箱地址>`。SMTP 授权密码中如果有 `$` 或 `#`，在 `.env` 中用单引号包围整个值。`.env` 自动由 Compose 读取，配置转换成现有 `INFORADAR_` 环境变量；容器不读取开发用 `config.yaml`。不要执行 `source .env`。

容器默认使用专用子网 `172.30.90.0/24`，Nginx 固定为 `172.30.90.10`，后端只信任这个代理地址。如果它与服务器的现有网络路由冲突，同时修改 `.env` 中的 `INFORADAR_DOCKER_SUBNET` 与 `INFORADAR_PROXY_IP`，代理 IP 必须位于该子网。

## 4. 启动 MySQL，手工建表

以下命令均在仓库根目录执行。先检查配置，再仅启动数据库：

```bash
docker compose config --quiet
docker compose up -d --wait --wait-timeout 180 mysql
docker compose exec mysql mysql --user=root --password
```

在提示处输入 `.env` 中的 `MYSQL_ROOT_PASSWORD`。进入 MySQL 控制台后依次执行：

```sql
SOURCE /schema/001_init.sql;
SOURCE /schema/002_radar.sql;
REVOKE ALL PRIVILEGES ON inforadar.* FROM 'inforadar'@'%';
GRANT SELECT, INSERT, UPDATE, DELETE ON inforadar.* TO 'inforadar'@'%';
SHOW TABLES FROM inforadar;
EXIT;
```

应看到八张表。镜像首次运行会创建数据库和独立应用账号，表结构仍由你执行 SQL 创建；SQL 只读挂载到 `/schema`，不会在启动时自动执行。后端启动时检查已有表，不执行迁移。已有旧版数据库不能通过重复执行 `CREATE TABLE IF NOT EXISTS` 更新。

MySQL 数据保存在 `inforadar_mysql-data` 持久卷。重新创建容器不会删除数据；`docker compose down -v` 会删除数据库卷，日常停止和更新不要使用 `-v`。MySQL 已有数据后，修改 `.env` 密码不会自动修改数据库账号密码，需要由管理账号执行 `ALTER USER` 后同步配置。

## 5. 构建并启动应用

```bash
docker compose build --pull api web
docker compose up -d --no-build --wait --wait-timeout 180 api web
docker compose ps
```

浏览器访问 `http://你的公网IP`。Compose 等待 MySQL 可连接再启动后端，等待后端健康再启动 Nginx。Nginx 保留 `/api/v1` 路径并转发请求，React 的登录、设置、重置密码等页面支持直接刷新。超出 1 MiB 的 API 请求由后端返回统一 JSON/413 响应。

登录后在“监控设置”填写用户自己的厂商 Key。最简单的组合为智谱模型与智谱搜索，共用一份 Key；无需在服务器上运行模型。未配置 Key 时可以创建主题，配置完整后才开始扫描。

基础检查与日志命令：

```bash
curl -fsS http://127.0.0.1/healthz
docker compose logs --tail=100 api web mysql
docker compose logs -f api
```

若调整了 HTTP 端口，`curl` 地址也加上对应端口。`/healthz` 只确认后端 HTTP 服务能响应；正式验收还需浏览器验证注册、登录、刷新后的续期、退出、修改密码，以及配置 Key 后的一次手动扫描。未配置 SMTP 时，找回密码应提示暂不可用；仅在配置 SMTP 后验证重置邮件与链接。重启应用容器后确认账号与主题保留、扫描继续运行。

HTTP 部署固定使用 `auth.cookie_secure: false`，密码、会话凭据和提交的 API Key 在网络传输中不加密。公网 HTTP 下浏览器通常没有 Web Locks，多标签页同时续期可能使其中一页返回登录页。

## 6. 备份、更新与回退

三个容器均设置 `unless-stopped`，Docker 随系统启动；每个容器的 JSON 日志最多保留三份、每份 10 MiB。后端收到终止信号会停止扫描并关闭服务，数据库使用持久卷。

更新前手动备份：

```bash
sh deploy/backup.sh
```

备份目录位于 `deploy/backups/inforadar-时间-进程号/`，包含 `database.sql`、`deployment.env`、`revision.txt`、`images.txt`。导出使用事务快照，失败时清理未完成的备份。数据库备份必须与原始 `radar.encryption_key` 一起保存；没有它就无法解密已保存的厂商 Key。

该目录包含全部部署凭据，需通过 SSH/SCP 复制到服务器之外的私密存储，并保留最近七份已确认完整的备份。只留在同一台服务器上无法覆盖服务器丢失场景。

更新时记录旧镜像版本，拉取代码，将 `.env` 的 `INFORADAR_IMAGE_TAG` 改为新版本，保留密码和密钥，然后重新构建启动：

```bash
git pull --ff-only
nano .env
docker compose build --pull api web
docker compose up -d --no-build --wait --wait-timeout 180 api web
```

新版本需要表结构变更时，先备份并按其说明手工执行 SQL。回退应用时将 `INFORADAR_IMAGE_TAG` 改回旧版本，再执行：

```bash
docker compose up -d --no-build --pull never --wait --wait-timeout 180 api web
```

保留旧版本镜像直到新版本运行稳定，期间不要清理旧镜像。以上只回退应用镜像；数据库结构变更需要另外检查是否兼容旧版本。

## 7. 在新服务器恢复备份

先安装 Docker、上传相同代码版本，将备份的 `deployment.env` 复制为仓库根目录 `.env` 并设置 `chmod 600`。保留密码与加密密钥，将前端地址改为新服务器 IP，然后启动 MySQL：

```bash
docker compose up -d --wait --wait-timeout 180 mysql
docker compose cp /你的备份目录/database.sql mysql:/tmp/inforadar-restore.sql
docker compose exec mysql mysql --user=root --password
```

在 MySQL 控制台执行：

```sql
SOURCE /tmp/inforadar-restore.sql;
REVOKE ALL PRIVILEGES ON inforadar.* FROM 'inforadar'@'%';
GRANT SELECT, INSERT, UPDATE, DELETE ON inforadar.* TO 'inforadar'@'%';
EXIT;
```

随后按第 5 步构建并启动应用。恢复文件导入到已有库会替换同名表的数据；上述流程用于新服务器，已有服务器覆盖恢复前要先备份并暂停 `web`、`api`。

## 常见启动问题

- `Fill ... in .env`：模板中有必填项未填写。先填写根目录 `.env`，再运行 `docker compose config --quiet`。
- 后端提示表缺失：数据库就绪不代表表已创建，按第 4 步手工执行两份 SQL。
- 浏览器请求返回 403：`INFORADAR_SERVER_FRONTEND_URL` 必须与浏览器地址的协议、IP 和端口一致。
- 找回密码提示暂不可用：默认未配置 SMTP 时的预期行为；需要启用时填写完整 SMTP 参数，并重新创建 API 容器。
- 已配置 SMTP 但找回密码不收到邮件：核对 SMTP 授权码、发件人、端口、TLS 模式与出站规则；不能直接使用本地测试 SMTP 配置。
- API 容器访问 `mysql` 失败：容器内不能使用 `127.0.0.1` 连接另一个容器，Compose 已配置服务名 `mysql`。
- 镜像或依赖下载失败：检查服务器到 Docker Hub、Go 模块仓库和 npm 仓库的网络。配置可信的镜像源后再构建；Dockerfile 的基础镜像可通过 `GO_IMAGE`、`NODE_IMAGE`、`API_IMAGE`、`NGINX_IMAGE` 构建参数替换。
