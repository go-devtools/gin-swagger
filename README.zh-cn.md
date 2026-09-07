# gin-swagger

[English](README.md)

无业务侵入的 Gin 源码契约生成与原生 OpenAPI 3.2 运行时适配器。

当前 SDK 尚处于 1.0 之前，接口可能随固定版本更新而变化。请使用公开 API，并先核对各功能指南中的支持范围。

## 环境要求

- 开发和验证使用 Go 1.27.1。
- 最低版本验收使用 Gin v1.12.0。
- 核心依赖固定为 `github.com/openapi-golang/openapi v0.0.0-20260907083400-2a27bf547b5e`，由 Go 工具从真实远端提交解析。

## 快速开始

在本仓库根目录使用 Go 1.27.1 和 `go.mod` 中固定的依赖执行：

```sh
GOWORK=off go mod download
GOWORK=off go run ./cmd/gin-swagger generate --dir ./examples/basic --output ./internal/apidoc
GOWORK=off go run ./cmd/gin-swagger check --dir ./examples/basic --output ./internal/apidoc
GOWORK=off make dev
LISTEN_ADDR=127.0.0.1:8080 GOWORK=off go run ./examples/basic
```

打开[本地 Swagger UI](http://127.0.0.1:8080/docs/)或[原生文档](http://127.0.0.1:8080/docs/openapi.json)，按 Ctrl-C 停止示例。DTO 不使用 tag，生成器只写入 `examples/basic/internal/apidoc/zz_openapi.gen.go`。另开终端发送真实请求，可得到 HTTP 201 和创建的用户：

```sh
curl -i -H 'Content-Type: application/json' -d '{"Name":"alice"}' http://127.0.0.1:8080/users
```

接入既有应用时，保留业务 handler 和路由注册方式，增加生成包导入，在所有业务路由注册后、启动服务前挂载一次：

```go
// r already contains the application's business routes.
document, err := ginswagger.Mount(r, apidoc.Bundle(), ginswagger.Config{
    OpenAPI: openapi.Config{Title: "User service", Version: "1.0.0"},
    Path: "/docs",
})
if err != nil {
    return err
}
_ = document
```

片段应置于返回 error 的启动函数中。导入 `github.com/openapi-golang/openapi`，将 `github.com/openapi-golang/gin-swagger` 别名设为 `ginswagger`，并导入**自己模块**生成的 `internal/apidoc` 包。完整可编译应用见[基础示例](examples/basic/main.go)。CLI 与应用使用同一固定适配器版本；首次生成允许生成目录尚不存在。

`Build(r, bundle, config)` 只构建文档，不注册路由；`Mount` 还会准备并挂载共享 UI。上述最小配置默认关闭实际 API 提交，演示应用则显式启用选定的方法。通过 `Document.JSON()`、`Report()` 和 `WriteFile(path)` 获取缓存结果与诊断。源码新鲜度检查无法获知仅在运行时确定的路由，还应从真实 Router 导出文档并执行 `gin-swagger check --spec`。

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

开发检查使用 `GOWORK=off make dev`、`GOWORK=off go test -race ./...`、`GOWORK=off go vet ./...` 和 `GOWORK=off go mod verify`，保持固定核心版本且不设置本地 replace。独立消费者测试可通过 `GIN_SWAGGER_TEST_VERSION` 选择真实远端固定版本。

自有源码注释、OpenAPI 描述、诊断、CLI 帮助、示例文字和提交信息使用英文。多语言编码测试保留其真实输入数据，上游资源保留原文，本文件提供对应的中文使用说明。

显式 JSON、Query、URI、Header、FormPost、Multipart 与集中自定义解码规则见[请求绑定指南](docs/requests.md)。

请求绑定指南同时说明强制 Bind 调用、关联错误返回值、已提交的 400/413 响应，以及自动 Bind/ShouldBind 和显式 Form 的有限方法与媒体条件。通过 Config 集中声明默认或单路由请求媒体范围，解析文档条件而不改变请求处理。

构建目标、依赖源码、overlay 和声明的集中映射配置均参与生成新鲜度。包含 TypeMapper 的编译调用需要通过 Options.Configuration 提供具名 JSON 输入；Bundle 仅保留配置摘要。

Gin 的 `Build` 和 `Mount` 会校验当前程序与生成 Bundle 的构建条件。已知不匹配时在挂载文档路由前失败，缺少元数据时在 `Document.Report()` 保留警告。这不能替代 CI 的源码新鲜度检查。

原始 PostForm、数组/字典读取、FormFile 和 SaveUploadedFile 已保留正文与查询位置、编码和业务错误分支。已验证行为及剩余边界见[请求指南](docs/requests.md)。

HEAD 和重定向响应语义已通过真实 HTTP 服务验证，详见[响应指南](docs/responses.md)。

Gin SSEvent 和标准 sse.Event 渲染器生成原生事件 itemSchema，保留实际文本/JSON 载荷及已发送元数据。真实 HTTP 矩阵与尚待完成的流回调边界见[响应指南](docs/responses.md)。

Stream 回调与面向 Gin 响应 Writer 的 JSON Encoder 通过公开核心回调 SDK 推导，涵盖明确的 NDJSON 分帧及稳定长连接 SSE。已测行为与剩余边界见[响应指南](docs/responses.md)。

## AI 辅助接入

从 [llms.txt](llms.txt) 查看精简文档索引，再阅读 [AI 接入指南](docs/ai-integration.md)，获取真实可执行的命令、结构化诊断说明及公开 API 边界。生成的 JSON 和来源信息可用于核对接入判断。

函数注释可以通过共享核心声明请求和响应类型。当前包类型和完整模块路径的泛型类型保留真实 Go 类型身份；同一位置的声明必须与已推导 Schema 一致。未知行为仍需集中规则补充，详见[请求与响应类型声明](docs/requests.md#explicit-request-and-response-types)。

参见[性能指南](docs/performance.md)，运行可复现的 100／1000 路由生成、启动 Build、文档读取与内存分配基准。

参见[独立 CI 指南](docs/ci.md)，了解固定工具链、私有模块访问、离线浏览器检查、真实平台任务和固定远端版本消费验证。

[来源解释命令](docs/ai-integration.md#explain-a-field-or-response) 可查询字段和响应来源、实际投影规则及声明，并明确区分契约与业务实施证据。

当前固定核心版本使用 `spec.Optional[bool]` 表示可选标准布尔字段。通过 `spec.Set(false)` 保留显式假值，判断标志时读取 `.Value`；详见[原生对象迁移](https://github.com/openapi-golang/openapi/blob/main/docs/native-objects.md)。

固定核心版本会校验原生 HTTP 对象结构与引用解析后的参数上下文，包括 Path Item 继承、操作级覆盖、整段查询冲突及离线文档间的 Link 操作身份。`spec.Parameter.Name` 保留原生查询参数的显式空名称。这些检查不会改变 Gin 路由或业务 handler；详见 [HTTP 验证边界](https://github.com/openapi-golang/openapi/blob/2a27bf547b5e4397bde6289df65caba482b6cd7f/docs/native-objects.md#http-objects-and-parameter-contexts)。

固定核心版本同时校验原生元数据字段类型、必填项、组件名称及许可证互斥字段。参阅[元数据规则](https://github.com/openapi-golang/openapi/blob/2a27bf547b5e4397bde6289df65caba482b6cd7f/docs/native-objects.md#document-metadata-and-component-names)，其中明确说明了空请求体 content 的处理策略。

Gin 路径编码遵循 Engine 的实际配置。请在 Gin 初始化前构建文档；含转义静态冒号的路由若需在初始化后构建，应提前保存 `Engine.Routes()` 到 `Config.RegisteredRoutes`。参阅[路径编码与路由快照](docs/paths.md)，了解 raw 路径条件、过期快照诊断及挂载边界。

闭包、接收者方法和泛型 handler 使用运行时证据或集中绑定来关联契约。参阅 [handler 身份](docs/identity.md)，了解已验证的共用契约，以及普通、trimpath 和符号裁剪构建；未知泛型载荷仍会被拒绝。

挂载文档复用核心原生兼容性面板，提示未呈现的扩展方法和标签元数据，同时保留原始文档。参见 [UI 展示与提交边界](https://github.com/openapi-golang/openapi/blob/2a27bf547b5e4397bde6289df65caba482b6cd7f/docs/swaggerui-compatibility.md)。

共享 UI 将请求与响应的流式单项 Schema 和完整消息 Schema 分开展示，并保留有限 NDJSON/SSE 的实际字节。由于固定客户端序列化时会省略整段查询参数的值，相关操作保持只读，并提供结构化浏览器诊断说明限制。

[公开适配器 SDK 权威说明](https://github.com/openapi-golang/openapi/blob/2a27bf547b5e4397bde6289df65caba482b6cd7f/docs/adapter-sdk.md)固定到当前模块使用的核心提交。
