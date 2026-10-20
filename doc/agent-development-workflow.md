# Nightingale AI 开发工作流

一个可验收任务对应一个分支和一个 Paseo worktree，由一个主要 Agent 负责实现。
人负责目标、产品取舍和体验验收；Agent 负责调查、实现、工程检查和可追溯预览。

## 开始任务

先读 [README](../README_zh.md)、[AGENTS.md](../AGENTS.md)、
[架构地图](../CLAUDE.md)，再读目标实现和 `doc/api/` 文档。
新主机按[环境准备](development-environment.md)操作。
本次设施实施的实际检查、基线失败和未验证范围见[验证记录](agent-workflow-validation.md)。

```bash
git status --short
git branch --show-current
git log -5 --oneline
go version
python3 --version
```

任务描述包含场景、期望行为、范围和可观察的验收步骤。简单修改直接实施；跨模块、
跨仓库或迁移任务先明确顺序和验收点。保留已有修改归属，记录基线失败。
Paseo setup 只配置 hooks，install 下载 Go 模块，预览按需构建。

## 实施与检查

沿调用链确认 UI/API、models、memsto、center/edge 和数据源边界。公开入口可能被
n9e-plus 使用；优先扩展字段或 setter，保持签名兼容。

| 范围 | 验证重点 |
| --- | --- |
| 路由/models | 鉴权、业务组权限、失败路径；edge/alert 远程读取 |
| Schema | gorm、初始化/迁移 SQL；MySQL/PostgreSQL/SQLite 差异 |
| 告警/通知 | 判断、恢复、屏蔽、派发；测试接收端，避免真实通知 |
| 集成模板 | action 检查、全部 en_US 渲染检查 |
| MCP/A2A/AI | 身份作用域、读写边界、流式取消；真实模型单独验收 |
| Skill 脚本 | 真实隔离引擎及专项执行验证 |
| 前端 | 指定 n9e/fe revision 和产物；浏览器成功、失败、恢复路径 |

先跑相关行为测试，再执行 `make verify`：改动 Go 文件格式、预览设施测试、
`go build ./...`、`go test ./...`，按变更范围执行集成检查。
`make verify-all` 另含 `go vet ./...` 和两个完整集成检查。CI、Paseo 和 pre-push
复用同一入口。部分 dskit 测试需要外部服务，失败要记录缺失环境和具体结果；
SQLite 单测不代表其他数据库验证通过，不能跳过门禁后称为成功。

## 预览与验收

首次 `make preview`，修改后停止编辑并 `make preview-refresh`。preview 会复用
已运行实例，不自动重建。完整命令见[运行手册](worktree-preview.md)。
worktree 使用十个槽位：`https://n9e-dev0.cubbber.dev` 至
`https://n9e-dev9.cubbber.dev`，端口 30300–30309。地址用 `make preview-url` 获取，
不同 worktree 不抢占已有绑定。`https://n9e.cubbber.dev` 是独立的稳定主版本入口，
端口 30299，不参加 worktree 槽位分配；分支预览、刷新和归档不会更新或清理它。

稳定实例用 `make stable` / `make stable-refresh` 显式管理，从配置的主 checkout
读取应用源码，要求位于预期的 main 分支且没有未提交修改。它有独立数据库、账号和
版本，不把当前任务 worktree 的源码发布到稳定地址。infra 尚未合并进主分支时，也
可以从有管理脚本的 worktree 执行 stable 命令；网关由管理脚本构建，应用仍来自主 checkout。

每个实例有独立 SQLite、miniredis、TSDB、Skill、日志和随机 root 密码；不共享日常
服务数据。Go 模块与编译缓存共享。预览记录 commit、branch、未提交源码指纹、前端
摘要及 build ID；commit 不能单独代表脏工作区。

交付前 status 应为 running、本地/公网通过、sourceChanged=false；外部前端还需
frontendChanged=false。从准备入口打开会清理该域名旧登录状态和缓存，再点击登录。
凭据文件路径由 status 提供，只在本机读取，不提交或贴入公共日志。

Ready 验证版本、页面、TSDB 查询、首次账号登录及鉴权接口，不替代业务功能、真实
模型和人工验收。[PR 模板](../.github/PULL_REQUEST_TEMPLATE.md)要求人在运行实例上
验证；Agent 浏览器结果单独记录，不能代填人工确认。

## 合并、清理与交接

按用户授权提交、推送、PR、合并和发布。基建进入基础分支后，新 worktree 才能继承，
已有工作区需同步。暂停用 stop；保留需要的数据与证据后 destroy。Paseo teardown
在归档前执行 destroy，失败时查日志重试，不能仅凭归档 UI 判断资源已释放。

```text
目标与范围：场景、结果、已有产品选择。
验收：触发步骤、可观察结果、失败恢复及平台。
代码：分支、commit、本任务修改与其他已有修改。
检查：命令及通过/失败/跳过，基线问题。
预览：地址、build ID、源码/前端匹配、验收路径。
人工验收：已完成或待完成，不能用自动检查代填。
数据影响：迁移、外部服务和需要保留的数据。
下一步：未验证事项及明确动作。
```
