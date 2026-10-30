# 添加自己的扩展

如果要让外部 Agent 操作 ARTEX 控制台，使用独立的 [artex-operator 技能](../agent-skills/artex-operator/SKILL.md)。它不是 ARTEX 内部执行器的 Skill，不要上传进下面的运行时 Skill 目录。外部技能可直接配合 v0.4.0 API 使用。

已打包的技能可从私有 Release 下载：[artex-operator-skill.zip](https://github.com/tianba777/ARTEX-LoneHero/releases/download/v0.4.2/artex-operator-skill.zip) 和[独立 SHA256 校验文件](https://github.com/tianba777/ARTEX-LoneHero/releases/download/v0.4.2/artex-operator-skill.zip.sha256)。解压后将 `artex-operator` 目录放入外部 Agent 支持的技能目录。v0.4.2 程序包和镜像也包含同版外部技能；上游动态接口需要 v0.4.1，其余基础操作可配合 v0.4.0 使用。

优先使用已有的 Skill、自定义工具和 MCP 入口。添加普通扩展不需要修改任务调度、模型池或新增一套插件加载器。

| 你要添加什么 | 最短路径 |
| --- | --- |
| 操作方法、工作流程、参考资料 | 系统 → Skill → 新建 Skill，或上传压缩包 |
| 已有程序、本地脚本或普通 API | 系统 → 工具 → 自定义工具 → 新建自定义工具 |
| 已提供 MCP 接口的服务 | 系统 → MCP → 添加 MCP |

## Skill

1. 在“系统 → Skill”创建，或者上传包含 `SKILL.md` 的 ZIP。建议每个 ZIP 只放一个 Skill。
2. 选中安装后的 Skill，核对名称、描述与文件；为需要使用它的 Agent 授予可见权限。
3. 如果依赖 MCP，在 Skill 的关联 MCP 中选择已经配置好的服务。
4. 用一个小任务确认能发现和加载它。安装 Skill 不会自动安装其外部程序或 MCP 服务。

最小目录示例：

```text
my-data-format/
  SKILL.md
  references/        # 可选参考资料
  scripts/           # 可选辅助脚本
```

`SKILL.md` 示例：

```markdown
---
name: my-data-format
description: 整理用户提供的表格数据，统一列名与日期格式。
---

先确认输入列和期望格式；转换后检查行数与空值，保留原文件。
```

名称建议使用小写字母、数字和连字符，并与目录名一致。上传同名 Skill 会要求确认覆盖；自己的扩展使用独立名称，便于保留修改和同步上游。

## 自定义工具

按现有页面选择类型，并配置工具描述和 Agent 绑定：

- `shell`：声明执行环境中已经存在的程序，供 Bash 调用；这个选项本身不安装程序。
- `command`：配置本地命令模板。
- `script`：配置 Python 脚本。
- `http`：配置普通 API 请求及参数 JSON Schema。

Docker 部署中，程序实际在 ARTEX 容器里执行。只安装在宿主机上的程序不会自动出现在容器中。
自己的普通脚本或兼容的可执行文件可放在宿主机 `data/local-tools/`，现有 Compose 挂载会使它们出现在容器 `/app/data/local-tools/`；命令模板使用相应绝对路径。该目录不会自动加入 PATH，程序所需依赖仍须存在于容器环境。
依赖额外系统包的扩展应使用自己的派生镜像保存安装步骤，避免仅修改一个随后会被重建的运行容器。

## MCP

在“系统 → MCP”按服务提供方的接口填写 `stdio`、`http` 或 `sse` 配置，保存后刷新工具列表，再设置相应 Agent 可见权限或 Skill 关联。
本地进程路径、网络地址和请求头均从 ARTEX 服务端的运行环境解释；Docker 中的路径及 `localhost` 指容器自身。

## 升级与同步时保留扩展

- 现有 Compose 已持久化 `skills/` 和 `data/`；工具定义、绑定和 MCP 配置保存在 PostgreSQL。
- 升级前运行现有 `scripts/backup.sh --with-data`，同时备份数据库、文件、Skills 与配置。
- 需要随项目分发的普通扩展，作为独立提交维护，记录来源、版本与依赖；凭据通过部署配置提供。
- 上游同步前先保存文件改动，使工作区干净，再运行 `scripts/sync-upstream.sh`。同步后检查自己的扩展及绑定，不用上游目录整体覆盖本地 Skills。
