# 整合验证记录

## 正式发行 v0.4.2（2026-10-30）

- [私有正式版](https://github.com/tianba777/ARTEX-LoneHero/releases/tag/v0.4.2) 的标签固定在 `1912e2d12fa25fd686cc318a69d5817c3870b17a`；[同一提交完整 CI](https://github.com/tianba777/ARTEX-LoneHero/actions/runs/41041247480) 通过。
- [发布任务](https://github.com/tianba777/ARTEX-LoneHero/actions/runs/41041250930) 一次通过，未使用恢复任务：五个平台 ZIP 与 `SHA256SUMS` 完整生成，原生 AMD64、ARM64 镜像均完成启动、健康检查、内嵌前端、工具链和私有可见性检查；镜像内 Skills 数断言自本版起为 10 个，覆盖新增的 php2shell 与 edr-bypass-re 技能。
- 六个 ZIP（五个平台程序和独立外部技能）的 GitHub 资产 SHA256 摘要与 `SHA256SUMS` 逐项核对一致；php2shell 的 PoC 脚本留在源码树 `scripts/poc/php2shell`，不进入发行镜像与 ZIP，与供体打包方式一致。
- 镜像统一提供 `ghcr.io/tianba777/artex-lonehero:v0.4.2` 与 `latest`，支持 Linux AMD64/ARM64。仓库与镜像保持私有；原 v0.4.0、v0.4.1 标签、程序包和版本镜像保留。
- 本版仅包含技能移植、文档与断言更新；没有数据库 Schema 变更，没有部署到现有服务，没有运行真实目标扫描。

## 正式发行 v0.4.1（2026-10-30）

- [私有正式版](https://github.com/tianba777/ARTEX-LoneHero/releases/tag/v0.4.1) 的标签固定在 `cb1eeef3f10b2bd5cd4b37d2fe65dae0e565115a`；[同一提交完整 CI](https://github.com/tianba777/ARTEX-LoneHero/actions/runs/41027785813) 通过。
- [原始发布任务](https://github.com/tianba777/ARTEX-LoneHero/actions/runs/41028188602) 完成五个平台 ZIP，以及原生 AMD64、ARM64 镜像的启动、内嵌前端、工具和私有可见性检查。最后步骤长期等待执行资源，通过[恢复发布任务](https://github.com/tianba777/ARTEX-LoneHero/actions/runs/41031235186)复用同一批产物完成；原任务已取消，没有重建程序或移动版本标签。
- 六个 ZIP（五个平台程序和独立外部技能）对应的 GitHub 资产 SHA256 摘要全部与 `SHA256SUMS` 一致；实际下载的技能 ZIP、独立校验文件和总校验清单也已核对，技能包含 10 个来源文件和许可证。
- 镜像统一提供 `ghcr.io/tianba777/artex-lonehero:v0.4.1` 与 `latest`，支持 Linux AMD64/ARM64。仓库与镜像保持私有，匿名访问实际无法获取正式版或拉取镜像；原 v0.4.0 标签、程序包和版本镜像保留。
- 实际调用与 v0.4.0 相同的自更新实现，以旧版本身份查询并下载私有 v0.4.1 的 macOS ARM64 包，完成 SHA256、解压、可执行文件试运行和暂存检查。全部在独立临时目录完成，没有替换现有程序。GitHub 元数据查询曾出现短暂连接失败，有限重试后通过；完整程序包下载和暂存成功。
- 用该校验通过的正式可执行文件、独立发行的外部技能和临时 PostgreSQL 启动实例：技能客户端实际登录、会话文件权限 600、两个上游 API、内嵌设置页面和本项目私有更新查询均通过，两项远端查询均首次成功。RuoJi6 返回快照之后有新增提交，BaaRai 与参考快照一致；未自动整合上游。
- 外部技能分发流程从本版起自动生成 ZIP 和独立校验文件。没有部署到现有服务，没有运行真实目标扫描。

## 版本页上游动态（v0.4.1）

- 新增登录后的上游查询 API 与版本页组件，读取同一份来源快照；区分正式版缺失、查询失败、提交一致/新增/落后/分叉。10 项定向测试覆盖公开请求不携带私有凭据、固定 GitHub 链接、部分失败、缓存/代理切换、并发查询、取消与认证边界，定向 race 检查通过。
- 本地完整 Go 构建、vet、PostgreSQL 隔离数据库测试通过；前端 TypeScript 与静态构建通过；评测脚本 13 项与外部技能客户端 7 项本地测试通过，接口目录校验通过。
- 首轮远端后端检查暴露原有备注测试的后台日历竞争：其他 API 测试遗留的调度循环可修改共用数据库中的计划状态。改为独立数据库和已取消的服务上下文，保留所有日历/任务状态断言；本地连续 10 次通过。没有修改生产调度逻辑。
- 临时原生实例在浏览器实际加载并手动刷新，两个来源均显示正式版、最新提交、快照与查询时间；自己的私有 Release 查询仍正常。未调用更新安装、上游合并或真实目标任务。
- 外部技能 ZIP 与独立 SHA256 首次附加到私有 v0.4.0 Release，GitHub 资产摘要已核对。v0.4.0 程序包、镜像和原 SHA256SUMS 未替换；新增上游界面随 v0.4.1 正式程序包和镜像提供。

## 外部 Agent 驱动技能

- 新增独立的 [artex-operator](../agent-skills/artex-operator/SKILL.md)，供外部 Agent 使用，适配现有 v0.4.0 API；不绑定给 ARTEX 内部 Worker。
- 客户端 7 项本地模拟 HTTP 测试通过，包括登录与私有会话存储、跨地址跳转保护、不重复写入、Basic 门禁与 JWT 共用、待审批监控，以及上传/私有导出；Skill 官方格式校验通过。
- [完整 CI](https://github.com/tianba777/ARTEX-LoneHero/actions/runs/40997726212) 和 [双架构镜像检查](https://github.com/tianba777/ARTEX-LoneHero/actions/runs/40997758873) 通过。检查对应功能提交 `2eb9f188cba8c92acc24c3e451647ce775d6eea1`，后续仅补充说明与许可证。
- 本地原生测试包已核对包含外部技能与客户端，内部运行时 Skills 仍为 8 个。已发布的 v0.4.0 标签和安装文件保持原样；外部技能可独立获取，v0.4.1 起的安装包和镜像也附带它。
- 源码接口目录由脚本生成，CI 校验同步；上游 API 变更后须核对请求契约并重新生成。模拟测试不等于真实目标任务实测，本次没有运行真实目标扫描。

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
