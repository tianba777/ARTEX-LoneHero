# 正式版安装与私有更新

首版版本号为 **v0.4.0**。发布完成后，从本私有仓库的 [Releases](https://github.com/tianba777/ARTEX-LoneHero/releases/latest) 下载，Docker 镜像为 `ghcr.io/tianba777/artex-lonehero:v0.4.0`，另提供 `latest`。

## Docker：包含完整工具链

先用有仓库访问权限的 GitHub 账号克隆项目：

```bash
gh repo clone tianba777/ARTEX-LoneHero
cd ARTEX-LoneHero
cp .env.example .env
```

编辑 `.env`，设置自己的 `POSTGRES_PASSWORD`，首次安装建议将 `ARTEX_TAG` 固定为 `v0.4.0`。模型密钥可稍后在页面配置。

私有镜像需要登录 GHCR。按 [GitHub 官方说明](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)，使用 GitHub classic PAT 的 `read:packages` 权限，令牌在登录提示中输入：

```bash
docker login ghcr.io -u tianba777
docker compose pull
docker compose up -d --no-build
```

镜像同时支持 Linux AMD64、ARM64，Docker 自动选择架构。安装后打开 `http://localhost:8787` 设置管理员账号。

## 原生安装包

| 运行环境 | 文件 |
| --- | --- |
| Apple Silicon Mac | `artex-0.4.0-darwin-arm64.zip` |
| Intel Mac | `artex-0.4.0-darwin-amd64.zip` |
| Linux x86_64 | `artex-0.4.0-linux-amd64.zip` |
| Linux ARM64 | `artex-0.4.0-linux-arm64.zip` |
| Windows x86_64 | `artex-0.4.0-windows-amd64.zip` |

每个包包含内嵌前端的程序、启动脚本、8 个 Skills、浏览器扩展、辅助脚本、说明和许可证。原生包的外部 CLI 程序需在运行机器上另行准备；需要完整预装工具链时选择 Docker。

例如下载 Apple Silicon Mac 安装包：

```bash
gh release download v0.4.0 --repo tianba777/ARTEX-LoneHero \
  --pattern artex-0.4.0-darwin-arm64.zip --pattern SHA256SUMS
```

按 `SHA256SUMS` 校验下载的 ZIP，解压后复制 `config.example.json` 为 `config.json`，填写可用的 PostgreSQL 连接；将示例中的 `skill_dir` 留空或改为 `skills`，使用包内 Skills。数据库需事先准备。
macOS/Linux 用 `./start.sh` 启动；Windows 用 `start.bat`。首次打开页面设置管理员账号。

## 后续更新

**程序在线更新：**在服务端运行环境设置 `ARTEX_UPDATE_TOKEN`，使用本仓库 Contents 只读权限令牌；Docker 可在本地 `.env` 中配置。重启后在系统配置页检查版本。更新源固定为本项目的正式 Release；开发版 `dev` 不支持在线升级。

**完整 Docker 镜像更新：**升级前执行 `scripts/backup.sh --with-data`。在 `.env` 中把 `ARTEX_TAG` 改为新的正式标签，随后运行：

```bash
docker compose pull artex
docker compose up -d --no-deps --no-build artex
```

使用原部署目录、Compose 项目名称和数据卷，保留 `data/`、`skills/` 及数据库。程序在线更新只替换可执行文件；要更新外部工具链应更新镜像。数据库迁移不会随程序回滚而回退。

镜像登录凭据与程序更新令牌用途不同，均只保存在自己的运行环境，不随项目分发。
