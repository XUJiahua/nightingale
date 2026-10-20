# AI 开发设施验证记录

验证日期：2026-10-20。Linux amd64，Go 1.25.7，Python 3.12.3。
基础提交：97fe92976009b12a206d145e59a1ad4fddf5fbc3，实施分支 elegant-mole。
验证基于实施时的工作区快照，实际运行版本以 `make preview-status` 的 commit、源码指纹与 build ID 为准。

## 已通过

| 检查 | 结果 |
| --- | --- |
| `python3 -B -m unittest discover -s scripts -p 'test_*.py'` | 11 项通过，另覆盖十个 dev 域名、稳定域名保留、槽位池分离、稳定源码须为干净 main；保留原槽位/端口、PID、密码、指纹和探测测试 |
| `go test ./cmd/preview-gateway -race -count=1` | 通过；启动门禁、后端身份/健康、敏感路径拦截、API 参数/鉴权传递、SSE 首帧流式输出 |
| `go build ./...` | 通过 |
| `go vet ./...` | 通过 |
| 两项集成模板检查 | action、全量 en_US 渲染通过 |
| `python3 -B scripts/preview-smoke.py` | 真实双 worktree、多次构建的完整生命周期通过 |
| 本地与公网身份探测 | 同一 build ID，源码匹配；独立 named tunnel 和用户服务已运行 |
| 公网浏览器 | 准备入口、随机 root 账号登录、指标查询页面及 `vector(1)` 查询通过，返回一条值为 1 的结果 |
| 公网本机接口边界 | 直接访问 `/prometheus/...`、`/dumper/sync` 返回 403；页面通过鉴权代理查询正常 |

生命周期验证使用独立登记目录，检查两实例密码及 SQLite 互相独立；公网槽位不可
抢占；未提交源码变化被检测；前端被复制而不是实时链接；前端源产物变化可检测；
刷新更换 build ID、保留账号；停止不影响另一实例；销毁幂等且槽位能立即重用。
临时 worktree、进程、数据和绑定已清理，日常公网实例保留用于验收。

## 稳定入口与开发入口

`n9e.cubbber.dev` 独立绑定主 checkout（main），使用 30299，不属于 worktree 槽位池。
`n9e-dev0.cubbber.dev` 至 `n9e-dev9.cubbber.dev` 为开发槽位，使用 30300–30309。
当前 elegant-mole 分支迁到 dev0，原实例数据库和账号保留；主版本另建独立实例。
DNS 与专用 Tunnel 已配置全部 11 个域名，空闲 dev 槽位无实例时不返回应用页面。

主实例通过 `--stable` 显式启动/刷新；要求主 checkout 位于 main 且工作区干净。
普通 preview/refresh/归档只操作当前 worktree，不会替换或清理稳定实例。

## 全量门禁的已有失败

`make verify-all` 如实返回失败，原因是 `go test ./...` 的以下 7 个已有包失败：

| 包 | 观察到的原因 |
| --- | --- |
| `dskit/clickhouse` | 未配置测试连接，empty shards |
| `dskit/mysql` | 未配置测试连接，empty shards/empty pgsql shards |
| `dskit/victorialogs` | 127.0.0.1:9428 没有测试服务 |
| `models/migrate` | 连接 127.0.0.1:3306 失败，原测试继续使用空连接后 panic |
| `pkg/ormx` | 原数据库测试使用固定连接信息，连接/认证失败；部分 DSN 与测试调用不匹配 |
| `pkg/promql` | 原变量解析与二元运算相关期望不匹配 |
| `pushgw/writer` | 原条件 relabel 测试期望不匹配 |

已另建**未覆盖本次改动的干净基线 worktree**，在同一基础提交对这 7 个包运行
`go test -buildvcs=false`，全部复现失败。没有跳过测试、放宽断言或修改无关业务代码。
新 CI/pre-push 会继续暴露这些基线问题，预览通过不等于全量门禁通过。

## 尚未执行

- full profile 的 Docker sandbox 资源构建、实际隔离脚本执行和 QA 语料功能验收。
- 真实 LLM 调用、外部 MySQL/PostgreSQL/ClickHouse/VictoriaLogs 的完整功能矩阵。
- 跨架构发布、真实人工产品验收、推送、PR、合并或发布。

浏览器检查由 Agent 执行，不代替 PR 模板要求的人工确认。主机上的私有验证日志、
截图和账号文件位于预览状态目录及任务日志中，不提交仓库。
