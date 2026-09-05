# 验证记录

## 环境与仓库

- 实际平台：Darwin 27.0.0 arm64。
- 使用隔离下载并校验的精确 Go 1.27.1；Gin 固定 v1.12.0。
- 两个目标目录各自是 Git 根和独立 module；父目录未初始化 Git。
- SSH 成功认证为 Rainer-Yu；两个 private 远端创建后均通过 ls-remote，并已非强制推送首次提交。
- 核心首批提交：c28ea4b52b58b60732b80b65968ad6c7d6d9035e；适配器首批提交：86bcb463014574551129cf19ce5e1d6d0f57250f。
- 当前本地还有未提交实现增量，因此上述 SHA 不代表最终交付版本。

## 已执行的产品检查

- 核心 `GOWORK=off make dev` 退出 0：根包、CLI、compiler、contracttest、comment、validate、spec、swaggerui 测试及完整构建。
- 适配器专用 workspace 的 `make dev` 退出 0：首次或重复生成、运行时、CLI、前端、路径与完整示例测试及构建。
- `TestReadableSchemaTitles`：先观察到缺少 title 的失败，修复后退出 0。输入输出组件仍独立，展示标题相同。
- `TestFlowWriteOrdering`：内联条件读取、pending 状态、switch break、先提交状态后写 body，退出 0。
- `TestRequestContracts`：有效 ASCII/中文、字段缺失/null、太短/太长、非法 JSON/空 body，分别校验真实状态和响应 Schema；比较挂载前后完整响应，退出 0。
- `TestAllFieldTypesContract`：真实 GET 类型样本通过独立引擎；非法字符串和数字枚举被拒绝，退出 0。
- `node --test swaggerui/display-names.test.cjs swaggerui/startup.test.cjs`：Node v22.22.3，9 项通过、0 skip；验证名称隐藏、零值/复杂示例、枚举及注释、安全文本和分类恢复，拒绝任意查询配置覆盖。

## 实际浏览器

本机 `/docs/` 加载固定 swagger-ui-dist 5.32.15，展示 OAS 3.2、中文接口和 10 个示例操作（7 个标准路径）。通过内置浏览器逐次点击 Schema 标签、字段展开与模型列表，核对以下行为：

- 列表及响应模型显示 User、APIError、CreateUserRequest，不显示内部后缀。
- 字段 ID 展示紧凑 Example 1024，不显示带 #0 的 Schema 数组。
- 复制图标为高对比度双页图标，与展开箭头独立占位。
- 默认不存在 Try it out/Execute；浏览器重载网络记录仅有本机文档、脚本、CSS、图标和 data URI，没有外部 validator/CDN 请求。
- 旧深链接曾触发上游关于下划线转义的弃用日志；尚不能将浏览器控制台宣称为全历史零错误。
- Swagger UI 依赖 JavaScript Number，超过安全整数范围的显示存在上游精度限制。教学示例的六十四位数选择能准确显示且超过三十二位范围的值；核心独立精度测试仍使用 9007199254740993。

## 方法、鉴权与整体分类

- `TestHTTPMethodExamples`：13 个真实 HTTP 样本通过，包括五种方法、404、非法输入、PATCH 的 false/null、DELETE 204 空响应及 Deprecated。请求与响应使用独立 Schema 引擎。
- `TestExampleDocumentDefinitions`：all/resources/types/auth/legacy 分别包含 10/6/2/1/1 个操作，标签范围同步匹配。
- `TestDocumentGroups`、`TestInvalidGroupsLeaveRoutesUnchanged`、`TestGroupScopeIntersectionAndCache`：独立文档、ETag、HEAD、304、未知分类 404、错误配置无路由修改、全局范围求交，均通过。
- `TestExampleGroupsAndSecurity` 与 `TestEnumRequestExamples`：Bearer 的缺失/错误/正确凭据、三组命名请求及非法枚举，均通过；API Key 路由已按用户要求移除，并验证返回 404。
- 浏览器：分类下拉、刷新保留选择、Legacy 删除线及 Warning: Deprecated、Authorize 弹窗仅包含 Bearer、枚举 editor/1 和 viewer/2 实际切换均已观察。1280×900 和 390×844 页面没有横向溢出；当前独立验收页 error/warn 记录为空。未将浏览器窗口的历史日志概括为零错误。

标签筛选框已按用户要求在示例配置中关闭，页面介绍也移除对应说明；整体文档分类继续通过顶部选择器提供。

## 枚举注释

`TestEnumDescriptionsFromSource` 从真实常量的普通注释生成 `x-enum-descriptions`，与排序后的 enum 数组对应。字符串、数字/iota、单独常量及别名、浮点枚举经过测试。共享 UI 直接展示 `"admin" - 管理员`、`"editor" - 编辑者`、`"viewer" - 查看者`，以及 `0 - 待处理`、`1 - 执行中`、`2 - 已完成`；浏览器已在请求 Schema 页实际观察，当前验收页 error/warn 为空。说明缺失时仅显示原值；说明始终作为 React 文本，不插入 HTML。

## 独立 Schema 导出

`TestStandalonePreservesValuesAndDataReferences` 和 `TestStandaloneRootNumberAndDefinitionConflict` 先暴露已有 $defs 被覆盖及根级 9007199254740993 被舍入的问题；修复后核心 `GOWORK=off make dev` 通过。导出保留根与组件数值、既有 $defs 和示例中的业务 $ref，仅重写标准 Schema 位置的组件引用，重名定义返回诊断。

## 官方规范独立检查

已下载固定官方 OAS 3.2 `schema/2025-11-23`。独立 jsonschema/v6 v6.0.3 对实际导出文档的首次结构校验失败：`/security` 为 null、规范要求数组。

修复默认 Optional 复制后，新增“省略 / 显式 [] / 拒绝 null”回归通过；重新导出的真实文档再次通过官方结构 Schema 校验，退出 0。此项覆盖结构，不等于已经完成全部 OAS 3.2 语义、Schema 方言和高级功能矩阵。

固定四份官方 schema/schema-base/dialect/meta 资源和 Apache 2.0 许可证保存在 `contracttest/testdata/oas32`，来源与 SHA-256 见 PROVENANCE.md。`TestOfficialOpenAPI32Matrix` 用 jsonschema/v6 离线加载 schema-base，完整 fixture 正例与 14 个反例通过。`TestNative32AndPresence` 类型化完整往返及 `TestFullNative32Fixture` 自有检查通过。官方 Schema 不覆盖所有跨对象语义，不能据此宣称标准验收完成。

`TestNoFrameworkDependencies`、`TestRuntimeDependencyBoundary`、`TestExternalFrontend` 真实通过：外部临时 module 仅调用公开 SDK、执行编译/构建/契约/非 HTTP 资源消费。开发模式的临时 replace 明确记录，不作为远端固定版本证据；最终可通过 OPENAPI_TEST_CORE_VERSION 指定真实版本且禁止 replace。

## 尚未执行完毕

完整 race/vet/fuzz/benchmark、自动化浏览器跨环境回归、完整 3.2 语义矩阵、远端外部 SDK 模块、GitHub CI 和最终冷缓存远端固定版本验收尚未完成。未执行或受阻项目不记为通过。

## 示例英文文档

示例四个源文件的 OpenAPI 注释、枚举常量说明和文档配置改为英文，并重新生成 Bundle。适配器专用 workspace 下 `make dev` 退出 0；首次英文生成的指纹为 `13a98b4b4ffbf842052c738cbad98bd46f4d8a0ed741117fffc3f4ec2acf365b`；英文示例阶段后续 dev 回归指纹为 `6625363b0ebb4d44e61ae32a1069a683a053ea11477f1a00902f1b20375bfa4c`。导出的规范检查了 144 处标题、摘要、描述和枚举说明，没有中文说明残留；13 个业务及路由函数体经 Go AST 对比一致。浏览器实际显示五个英文分类、英文接口说明、`"admin" - Administrator`、`"editor" - Editor`、`"viewer" - Viewer`，以及 `0 - Pending`、`1 - Running`、`2 - Completed`。本节为当前英文版本的证据，前文中文截图观察仅代表此前版本。

## 显式资源与模型闭包

`TestCheckExplicitOfflineResources` 覆盖官方多文档用法中的检索 URI、$self 与 $id；`TestCheckNeverFetchesResources` 使用真实 HTTP 服务计数，默认拒绝与显式预载两种检查都没有发出请求。外部原始示例、重复键、尾随 JSON、资源身份冲突、预算边界、内嵌 $id 和累计 JSON 节点均有正反例。`TestBuildPrunesWithSchemaResourceSemantics` 与 `TestBuildOfflineResourceBackReference` 验证资源 URI、锚点、子节点目标、discriminator、示例数据隔离和跨资源回指的模型闭包。

CLI 清单测试验证资源路径相对于清单目录、普通文件读取预算、错误清单、help 退出码和取消状态。新增反例先在旧逻辑下失败，再修复通过。核心 `GOWORK=off make dev` 与适配器专用 workspace 下 `make dev` 均退出 0；本轮最终适配器生成指纹为 `88fc0e996b705c41e6e750e31961a3094c4e61fed47a036d7ed376e551f50e1c`。当前预览仍使用此前已验证的英文文档构建，本轮没有声明重新验收 UI 外部引用。

本轮最后补充并修复了两项组合回归：`externalValue` 指向的已识别 Schema 资源在裁剪后仍保留；discriminator 按组件名映射到带 `$id` 的 Schema 时，按该资源身份解析，不误报跨作用域。最终核心与 Gin 的 `make dev` 均退出 0。两仓库的 `go test -race ./...` 已通过；最后引用修改后的核心根包与 `internal/validate` 另行通过 race。

`FuzzReferenceGraph` 用 `-fuzztime=30s -parallel=2` 完成 1,132,358 次执行，退出 0，验证 URI / 锚点变异输入的有界性与诊断确定性。该 fuzz 不代表所有 Schema 关键字或动态引用实例语义均已覆盖。文档中的显式清单用法另通过真实 CLI 进程执行，输出 `{"diagnostics":[]}` 并退出 0。

## 英文示例与 Schema 检查回归

四个示例源码文件的注释再次核对为英文；13 个业务及路由函数体与翻译前的 Go AST 对比一致。当前运行的 `/docs/openapi.json` 实际返回 User Service，本次检查 152 处说明文本没有中文残留。核心新增 Schema 关键字及源码约束检查后，适配器专用 workspace 的 `make dev` 退出 0，11 个模板的 Bundle 指纹为 `487f2095471be99f75d2a6c751e32c9f59a2c08ae20e3f13328fbaed4b19ce03`。本轮未重启预览服务；原预览继续展示已验证的英文文档。当前仍不是关闭 workspace 后的远端固定版本验收。

## 独立契约资源验证回归

使用本地新核心执行专用 workspace 的 make dev 和 go test -race ./... 均退出 0。最后资源注解数据隔离修复后，dev 再次退出 0；生成 11 个模板，指纹为 `c7dd2f1870492242302e4854b552e22b99ffc201918587086205a7210bc7f589`。真实 Gin 业务样本仍通过新的独立资源验证路径。未修改或重启预览服务，未更新远端依赖伪版本；关闭 workspace 的冷缓存独立验收仍未完成。

## 真实远端固定核心版本

核心阶段提交 `1c9eeff38e45a3085d74dcaa66bea9b6e630a07a` 已推送至 openapi/main。Go 工具从该真实 SHA 解析出 `v0.0.0-20260905135839-1c9eeff38e45`，适配器 go.mod 和 go.sum 已固定该依赖。

- `GOWORK=off make dev`：退出 0，重新生成 11 个模板，指纹 `1821f9ebb6b3d18f7d300acd9308a4cd96066f61ccb6a6e36beca12bab4117ae`。
- `GOWORK=off go test -race ./...`：退出 0。
- `GOWORK=off go vet ./...`：退出 0。
- `GOWORK=off go mod verify`：退出 0，all modules verified。
- `GOWORK=off go list -m -json`：核心为上述固定版本，Gin 为 v1.12.0，核心目录来自下载模块缓存，结果不含 Replace；go.mod 同样没有 replace。
- 核心 `OPENAPI_TEST_CORE_VERSION=v0.0.0-20260905135839-1c9eeff38e45 GOWORK=off go test ./internal/verify -run '^TestExternalFrontend$' -count=1 -v`：退出 0。外部临时 module 的 TestPublicFrontend 和 TestTransportNeutralResources 均通过；该模式断言核心版本精确相等并拒绝 replace。
- 核心本身的 go mod verify 与 go vet ./... 也退出 0。

这些验证使用本任务现有模块缓存，没有把它们称为冷缓存验收。未重启预览服务或重新验证 UI 外部资源映射；最终两个仅含各自源码的冷环境与 GitHub CI 仍待完成。
