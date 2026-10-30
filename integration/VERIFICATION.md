# 整合验证记录

## 首个正式发行 v0.4.0（2026-10-30）

- [正式版](https://github.com/tianba777/ARTEX-LoneHero/releases/tag/v0.4.0) 已发布，源码标签对应 `31ecd4d30d768295dec60c08e07edf31f6fe2d1b`。
- [同一提交的完整 CI](https://github.com/tianba777/ARTEX-LoneHero/actions/runs/40994643616) 和 [正式发布流程](https://github.com/tianba777/ARTEX-LoneHero/actions/runs/40994647101) 均通过。
- 五个平台 ZIP 和 `SHA256SUMS` 已上传；下载校验清单后，与 GitHub 为五个发行资产计算的 SHA256 摘要逐项比对，全部一致。
- Linux AMD64、ARM64 镜像均实际连接临时 PostgreSQL 启动，检查 `0.4.0` 版本、内嵌前端、工具与 8 个 Skills，然后上传私有 GHCR 包；统一 `v0.4.0` 和 `latest` 标签已生成。
- 发布工作流已确认 GHCR 包 `visibility=private`；仓库也保持私有。实际匿名请求无法访问正式版或拉取镜像。
- 使用真实只读访问，在临时目录调用程序自身的私有更新流程：查询 `v0.4.0`、下载 macOS ARM64 安装包、校验 SHA256、解压并试运行新程序、暂存升级文件，全部通过。首次 CDN 连接中断，重试后成功；未替换现有应用。
- 公开 Demo 未启用，发布流程跳过了 Demo 部署。没有部署到现有服务。

安装及配置见 [INSTALL.md](INSTALL.md)。后续文档和发布流程的修订不移动已有正式版本标签。

## 整合阶段记录

2026-10-30，本地 macOS，Go 1.27.2、Node 24.13.1、独立 PostgreSQL 16.15。

- 完整 Go 测试通过：15 个有测试的包，无失败；使用按包隔离的临时数据库。
- Go vet、前端 TypeScript 检查与静态构建通过。
- 新增设置、经验、规划与模型切换相关的定向 Go race 检查通过。
- 评测脚本的 13 个本地单元测试通过；未启动真实测试目标或调用外部模型。
- 新页面已在浏览器加载，设置保存与恢复默认通过，见[页面截图](../screenshots/lonehero-settings.png)。
- [GitHub 完整 CI](https://github.com/tianba777/ARTEX-LoneHero/actions/runs/40992150847) 已通过：Ubuntu 24.04 后端构建、vet、完整 PostgreSQL 测试、前端类型/静态构建和脚本检查。
- 上游同步工作流已实测通过“当前分支包含上游最新提交”的路径；尚无新上游变更可验证真实冲突处理与 PR 路径。
- [完整工具镜像检查](https://github.com/tianba777/ARTEX-LoneHero/actions/runs/40992181123) 已在 AMD64、ARM64 两种原生 runner 通过：完整镜像构建、程序帮助命令、工具命令、8 个 Skills，以及扩展、报告复核脚本和许可证的存在性检查。本机未安装 Docker，这项结果来自 GitHub 的实际构建。
- 原生 macOS ARM64 ZIP 已实际构建并校验，包含 8 个 Skills、扩展、报告复核脚本与许可证。

CI 固定 Ubuntu 24.04，与镜像发布检查使用同一 runner 系列，避免 ubuntu-latest 漂移。
首次 Ubuntu 26.04 runner 的完整测试中，上游未改动的 `TestStaticSedGeneratedInputAndBounds`
失败：虚拟脚本链被保守标为不完整。没有放宽生产审批/系统工具可信路径检查，也没有跳过该测试。
Ubuntu 26.04 的这条路径尚不在本项目验证通过的兼容范围内。

上述独立检查对应功能提交 `95a6b701848abcaafe56d68e1e671ae5788ace1e`；后续提交仅修订文档。
初次 AMD64 工具镜像在导入时遇到 runner 磁盘不足，已通过移除该一次性 runner 的无关预装 SDK 解决，未裁减工具。
该整合阶段尚未发布正式 Release 或推送发行镜像；之后的首版结果见上方记录。测试构建不等于已部署。

这些检查不证明真实漏洞发现率、模型成本收益或所有第三方服务的可用性。
没有部署到现有服务，没有运行真实目标扫描。
