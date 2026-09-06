# 当前状态

完整 Goal 仍未完成最终验收。两个独立 Git 仓库及 Go module 均使用 main，远端保持 private。GitHub 描述和 README.md 为英文，README.zh-cn.md 提供中文版本；自有代码采用 MIT。Go 固定 1.27.1，Gin 固定 v1.12.0，核心依赖由 go.mod 固定为真实远端版本，没有本地 replace。

## 已实现与验证

从零 tag 业务源码生成 Bundle，在启动层通过 Build/Mount 读取真实 Gin 路由并链接共享核心。运行时不导入编译器，不读取业务源码，不将文档采集逻辑插入业务请求。适配器只使用核心公开 SDK，Gin 规则保留在本仓库。

示例具有 GET/POST/PUT/PATCH/DELETE、Deprecated、Bearer 授权、带含义的枚举、字段类型及五份整体分类。共享离线 Swagger UI 使用业务名称显示模型，保留内部引用身份；标签筛选关闭。示例 OpenAPI 展示为英文，自有代码注释为中英双语。最近核验的 13 个业务与路由函数体保持原始 AST，导出包含 152 处英文说明，且仅有 BearerAuth。

显式 JSON/Query/URI/Header/FormPost/Multipart、已支持的 ShouldBindWith/MustBindWith、强制绑定错误提交，以及自动 ShouldBind/Bind 与显式 Form 已有真实请求测试。有限方法/媒体条件、query/body 优先顺序与诊断隔离通过公开核心支持；DefaultRequestMediaTypes 和 RequestMediaTypes 只集中解析文档条件，不改变实际请求处理。17 组成功自动绑定请求、八组错误路径和独立消费者的六组请求分别验证。

文本、原始字节、读取器、明确标准 Renderer、状态提交和响应头已有真实响应矩阵。未知 renderer、未建模的临时响应/文件行为、歧义和不能准确表达的输入关系应明确失败，不靠猜测填补契约。支持范围见[请求](requests.md)和[响应](responses.md)指南。

核心最新构建输入清单记录真实加载目标、模块/工作区源码、overlay、构建条件和配置摘要。集中 TypeMapper 编译调用现在显式声明 Configuration。固定新远端核心后，关闭 workspace 的 dev、全量 race、vet、模块校验、生成新鲜度和示例构建均通过。独立核心消费者的 25 项公开 SDK race 测试及实际安装 CLI 的 Schema 导出验证通过。

内部 integration/verify 包验证实际源码生成、首次无 apidoc、重复生成、真实 HTTP 与独立契约、符号裁剪构建和运行时导出。开发消费者临时替换当前适配器时，核心仍固定真实远端；实际安装远端 CLI 的证据独立记录，不能混称。当前阶段复用已有任务缓存。

## 语言与历史

自有代码和生成器注释同时使用简体中文与英文；最新审计覆盖 2341 行自然语言 Go 注释，无缺少对应翻译。第三方资源、许可证与编译指令保留原文。已授权的历史提交说明翻译保留代码树、作者、提交者和时间；之后使用普通快进英文提交。

## 未完成范围

完整 tag/codec/Schema、helper/闭包/receiver、身份与来源矩阵；剩余原始表单/文件边界、完整流式与文件响应；剩余运行时构建条件与工具链组合；原生 OpenAPI 3.2 完整正反例；UI 外部资源及完整浏览器黑盒链路；GitHub CI、路由基准、完整文档和最终单仓库源码的双模块冷缓存验收。

未来 Fiber/Echo 仅为公开扩展方向，不新增这些产品仓库。各阶段精确版本、失败原因及验证结果见[验证记录](verification.md)。

本轮最终固定核心 `v0.0.0-20260905221805-0a1bf9fc1369`，Gin Build/Mount 通过公开 SDK 默认检查运行时构建条件。不同平台在文档路由注册前失败；独立消费者用不同 tags 构建真实裁剪符号程序后，启动明确拒绝旧 Bundle 且不输出文档。GOWORK=off dev、完整 race、vet、模块校验、新鲜度检查和示例构建全部通过。原有 13 个业务/路由函数体不变，导出 152 处英文说明及枚举含义，只有 BearerAuth。

本轮原始表单、重复查询、字典、文件读取和保存已通过实际请求验证，含动态字段名与 nil 文件保存诊断、路由隔离和文件写入副作用检查。核心已固定为 v0.0.0-20260905231240-d92a618c2829；GOWORK=off dev、完整 race、vet、模块校验、新鲜度检查和示例构建通过。新远端核心的 30 项外部 SDK race 测试与实际安装 CLI 导出通过，适配器发布后的远端 CLI 消费结果单独记录。

本轮新增方法与头条件化 Redirect，基于核心公开响应状态快照区分待提交状态、当前头存储与已提交网络头。HEAD 去掉正文但保留元数据及同一 handler 的 GET 表示。72 组真实 HTTP 重定向请求、后续写入、晚修改媒体头、Location 空白和无侵入比较通过；核心固定 v0.0.0-20260906001422-3d48d1ee95ce，无本地 replace。完整 dev、全量 race、最终 HTTP 矩阵 race、vet、新鲜度和示例构建通过。实际远端核心的 33 项 SDK race 测试及 CLI 验证通过；本适配器最终远端 CLI 验收独立记录。

核心依赖现固定为 `v0.0.0-20260906005919-bc9182815e96`，提供经公开外部 SDK 验证的 ResponseItem、内层 codec 和编译期 Schema 包装。该固定远端核心的 40 项 SDK race 测试与实际 CLI 安装/导出通过。适配器 GOWORK=off dev、完整 race、vet、模块校验、新鲜度和示例构建通过；13 个业务/路由函数体、152 处英文说明和 Bearer-only 配置保持一致。此依赖更新尚未增加 Gin SSEvent/Stream 自动推导，完整流式验收继续保留。

Gin SSEvent 与标准 sse.Event 已通过公开核心 SDK 推导原生 itemSchema；108 组真实 HTTP 样本验证文本/JSON、nil、元数据、状态及提交头。关闭 workspace、固定真实远端核心后的 dev 已通过，原有业务与路由函数体不变。完整 race、vet、依赖校验、新鲜度检查和示例构建也已通过，实际导出保留 152 处英文说明、三点二版本与仅 Bearer 授权。真实远端适配器消费者结果在同步后另行记录；Stream 回调、任意 Writer、NDJSON 生成及完整 Goal 的剩余矩阵继续保留。
