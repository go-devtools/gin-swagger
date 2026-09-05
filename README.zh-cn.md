# gin-swagger

[English](README.md)

无业务侵入的 Gin 源码契约生成与原生 OpenAPI 3.2 运行时适配器。

本仓库正在实现。完整验收目标见 [GOAL.md](GOAL.md)，实际证据与未完成工作见[状态](docs/status.md)和[验证记录](docs/verification.md)。当前尚未完成产品验收或发布。

## 环境要求

- 最低版本验收使用精确 Go 1.27.1。
- 最低版本验收使用 Gin v1.12.0。
- 核心依赖固定为 `github.com/openapi-golang/openapi v0.0.0-20260905203249-f0f660a9475e`，由 Go 工具从真实远端提交解析。

## 架构

Go 源码与真实编解码提供结构，普通注释提供业务语义。文档生成不修改业务 DTO tag、handler 函数体或签名，也不改变既有路由注册。

核心拥有类型投影、注释、中立效果、Bundle 和 OpenAPI 模型。框架适配器拥有框架调用语义、路径语法、handler 证据和挂载。运行时构建文档不导入编译器、不读取应用源码。

## 能力边界

- **自动推导：** 类型、支持的网络表示和已识别的源码效果，以实际测试为准。
- **显式声明：** 语义约束与高级契约；声明不等于已经证明服务端执行。
- **集中适配：** 自定义 codec 和未支持的项目 helper，通过显式 Go 扩展接入。
- **无法消歧：** 不确定或未支持行为必须返回诊断，不能猜测响应。

Fiber 和 Echo 仅为未来扩展方向，本仓库未交付或宣称支持这些适配器。

## Swagger UI 示例与分组

[基础示例](examples/basic) 包含以下真实路由。资源写入仅用于演示，不持久化数据。

| 方法 | 路由 | 行为 |
| --- | --- | --- |
| GET | `/examples/items/:id` | 读取资源；`missing` 演示 404。 |
| POST | `/examples/items` | 创建样本并返回 201。 |
| PUT | `/examples/items/:id` | 完整替换样本字段。 |
| PATCH | `/examples/items/:id` | 更新非 null 字段；`false` 是明确更新。 |
| DELETE | `/examples/items/:id` | 返回 204，不携带响应体。 |
| GET | `/examples/legacy/items/:id` | 已弃用接口，说明中给出替代路径。 |

`Config.Groups` 配置右上角 **Select a definition** 的整体文档分类。每项包含稳定的 `ID`、展示 `Name`，以及可选的 `Include(method, path)`；path 使用原始 Gin 路由语法。分类范围始终与全局 `Config.Include` 求交集，`Config.DefaultGroup` 指定默认分类。挂载前先构建全部文档，请求只读取 `/docs/groups/<ID>.json` 缓存；`/docs/openapi.json` 保留完整的已配置总览。

标签用于当前文档内部的接口分组。普通 `@openapi tags=[...]` 注释设置操作标签，`OpenAPI.Tags` 设置分组描述和顺序；`UI.Filter`、`UI.DocExpansion`、`UI.TagsSorter`、`UI.OperationsSorter` 控制筛选、展开与排序。示例关闭标签筛选框，提供“All endpoints、Users and resources、Types and enums、Authentication、Legacy · Deprecated”五个分类。

鉴权分类只提供 Bearer 授权接口。**Authorize** 中填写公开演示值 `demo-token`，无需添加 `Bearer` 前缀；该 token 只保护新示例路由，原 `/users` 接口不变。枚举示例提供 `admin/0`、`editor/1`、`viewer/2` 三组完整请求，Schema 中列出所有允许值。

示例项目的 OpenAPI 注释、接口说明、文档分组和字段说明使用英文；枚举显示如 `"admin" - Administrator`、`"editor" - Editor`、`0 - Pending`。

枚举说明直接读取类型常量的普通注释。生成的 `x-enum-descriptions` 与标准 `enum` 数组一一对应，共享 UI 显示为“值 - 含义”；没有注释时只显示原值。

刷新和深链接只恢复已注册的分类，其他 URL 查询配置保持禁用。默认禁止提交 API 请求；需要时通过 `UI.SubmitMethods` 显式启用指定的小写方法。

新 checkout 先运行 `GOWORK=off go mod download`，再运行 `GOWORK=off make dev`，避免首次依赖下载占用生成器默认一分钟预算。直接调用 CLI 时可通过 `--timeout=5m` 设置更长预算。

## 响应渲染

前端从真实 Gin 调用推导文本、原始字节、读取器、明确标准 Renderer 及立即提交状态。已支持范围与未完成边界见[响应及验收指南](docs/responses.md)。

## 许可证

项目新增代码采用 [MIT](LICENSE)。第三方资源保留原许可证与声明。

## 当前模块验证

使用下载到模块缓存的固定核心版本，关闭 workspace 且不设置本地 replace 后，`GOWORK=off make dev`、`go test -race ./...`、`go vet ./...` 和 `go mod verify` 均通过。这证明当前实现可使用真实远端依赖；最终冷缓存、CI 和完整能力验收仍见[验证记录](docs/verification.md)。

自有源码注释同时提供简体中文和英文。编译指令及上游资源保留原文；示例与 Schema 测试数据中的伴随翻译通过空行与 Go 声明注释分开，使生成说明保持原有语言。提交信息使用英文。

显式 JSON、Query、URI、Header、FormPost、Multipart 与集中自定义解码规则见[请求绑定指南](docs/requests.md)。

请求绑定指南同时说明强制 Bind 调用、关联错误返回值、已提交的 400/413 响应，以及自动 Bind/ShouldBind 和显式 Form 的有限方法与媒体条件。通过 Config 集中声明默认或单路由请求媒体范围，解析文档条件而不改变请求处理。
