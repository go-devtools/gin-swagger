# 当前状态

总体目标尚未完成验收。两个独立 Git 仓库及 Go module 均使用 main 主分支，远端保持 private。GitHub 描述和 README.md 使用英文，配套 README.zh-cn.md；自有代码采用 MIT，上游资源保留原许可证与通知。

自有代码注释已同时提供简体中文和英文，包括 UI 扩展、构建脚本、静态契约生成文件及临时引导文件。最新 AST 清单核对两仓库的 1889 行自然语言 Go 注释，没有缺少对应翻译；56 行 UI 和 Makefile 注释也保持双语。实际导出的 152 处示例 OpenAPI 说明仍为英文，13 个业务及路由函数体与原始 AST 一致。历史中文提交说明已按用户明确要求改为对应英文，改写前验证了完整 Git bundle 备份，逐条保留代码树、作者和时间，并使用精确远端版本保护推送。

零 tag 源码生成 Bundle、Gin 启动层挂载、五种 HTTP 方法、Deprecated、Bearer 授权、枚举说明、完整字段示例及五份文档分类已有实现和实际测试。共享离线 Swagger UI 使用业务类型名称展示 Schema，内部引用身份仍可区分；标签筛选框按要求关闭。

核心依赖固定为真实远端版本 v0.0.0-20260905203249-f0f660a9475e，Gin 固定 v1.12.0。核心包括独立 Schema 资源作用域导出、显式依赖嵌入、方言与预算，以及响应头提交快照和明确的网络表示 Schema。新的外部 module 对该远端核心版本运行九项公开 SDK race 测试和实际远端安装 CLI 的导出及实例校验，均通过且无 replace。

Gin 的文本、原始字节、读取器、标准显式 Renderer、状态提交与响应头规则已通过 20 组真实响应检查；204/304 Reader 的附加头另有两组回归。未知 Renderer、连续完整 body 写入、直接 Writer 调用、尚未建模的临时响应和保留状态的 Reader 序列明确诊断。Gin 专有规则只在适配器中，通用控制流仍通过公开核心 SDK。

专用 internal/integration 和 internal/verify 包现已实现并运行。独立消费者验证首次生成、重复生成确定性、新鲜度检查、真实请求契约、裁剪符号构建与运行时文档导出。开发测试临时 replace 仅指向当前适配器；核心始终使用真实远端固定版本。设置 GIN_SWAGGER_TEST_VERSION 可切换为远端适配器版本并禁止替换。当前构建的 CLI 与远端安装的 CLI 分别记录，不能混称。

关闭 workspace 的 make dev、完整 race、vet、模块校验与示例生成新鲜度检查均通过。本阶段使用已有任务缓存。此前独立空缓存验证的版本及首次下载超时记录保留在 verification.md，不能当作当前版本的冷缓存验收。冷环境应先运行 go mod download，避免首次下载占用生成器默认一分钟预算。

这不代表完整交付。全部 binder/render 与运行时身份矩阵、自定义方言与完整 Schema 组合矩阵、注释及编解码边界、专用验收包的完整矩阵、GitHub CI、完整文档和最终版本冷缓存复验仍需完成。未来 Fiber/Echo 仅作为经过测试的公开扩展方向，不新增产品仓库。

显式请求绑定已扩展到 Query、URI、Header、JSON、FormPost、Multipart 以及可传播别名的 ShouldBindWith。普通文本字段与 multipart 的类型规则由公开 BindingCodec 提供；核心复用注释、枚举、组件及参数展开。真实正例、错误分支、嵌入字段与集中自定义 UnmarshalParam mapper 已通过；同一 DTO 的文本输入与 JSON 输出分别验证。新增独立消费者包括查询字节数组及错误输入，关闭 workspace 的 dev/race/vet/模块校验通过。完整表单读取和编解码矩阵仍未完成，不能将本阶段称为全部请求绑定支持。

强制 BindJSON、BindQuery、BindHeader、BindUri 和已支持的 MustBindWith 现已通过公开 CallOutcomes 关联返回值与隐含提交。18 组真实请求涵盖成功、无效输入及请求体上限；立即返回、忽略错误和试图覆盖状态的行为与挂载前一致。独立消费者新增强制绑定契约用例。核心已固定到真实远端 a978fefd3c30，关闭 workspace 的 dev 已通过；完整目标仍未完成。

有限请求条件阶段已贯通自动 ShouldBind/Bind 和显式 Form，前端版本为 gin-v1.12-front-v5。配置通过 DefaultRequestMediaTypes 和按原始 METHOD /Gin/path 的 RequestMediaTypes 集中声明，仅用于选择文档契约。17 组成功请求和八组错误路径验证真实 Gin 的方法/媒体选择、重复值优先顺序及 400/413/422。条件诊断不污染其他媒体，跨位置 required 无法准确表达时明确失败。

本阶段固定核心 f0f660a9475e，在关闭 workspace 后通过 dev、全量 race、vet、模块校验和生成新鲜度检查；新建核心消费者的九项公开 SDK race 测试及固定远端 CLI Schema 导出与实例校验通过。Gin 独立消费者新增六组自动绑定真实请求，开发验收已通过；本次适配器发布后的实际远端 CLI 验证单独留存。13 个原有业务和路由函数体仍与原始 AST 一致，最新导出的 152 处说明均为英文。完整 Goal 和最终冷缓存验收保持未完成。
