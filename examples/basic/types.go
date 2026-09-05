package main

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
)

// 用户角色的封闭枚举。

// The closed set of supported user roles.
// @openapi enum examples=["admin","editor","viewer"]
type Role string

// 演示枚举值与源码注释。

// Demonstrates enum values with descriptions from source comments.
const (
	// 管理员

	// Administrator
	RoleAdmin Role = "admin"
	// 编辑者

	// Editor
	RoleEditor Role = "editor"
	// 查看者

	// Viewer
	RoleViewer Role = "viewer"
)

// 任务状态的数字枚举。

// Numeric enumeration of task states.
// @openapi enum examples=[0,1,2]
type TaskState int

// 完整的任务状态集合。

// The complete set of task states.
const (
	// 待处理

	// Pending
	StatePending TaskState = 0
	// 执行中

	// Running
	StateRunning TaskState = 1
	// 已完成

	// Completed
	StateDone TaskState = 2
)

// 业务字符串别名。

// A business-specific string alias.
type Label = string

// 展示 JSON 能表达的各类 Go 数字。

// Demonstrates Go numeric types that can be represented in JSON.
type NumberFields struct {
	// 有符号机器整数。

	// Signed machine-sized integer.
	// @openapi examples=[-42]
	Int int
	// 八位有符号数。

	// Signed 8-bit integer.
	// @openapi examples=[-8]
	Int8 int8
	// 十六位有符号数。

	// Signed 16-bit integer.
	// @openapi examples=[-1600]
	Int16 int16
	// 三十二位有符号数。

	// Signed 32-bit integer.
	// @openapi examples=[-32000]
	Int32 int32
	// 六十四位有符号数。

	// Signed 64-bit integer.
	// @openapi examples=[1099511627776]
	Int64 int64
	// 无符号机器整数。

	// Unsigned machine-sized integer.
	// @openapi examples=[42]
	Uint uint
	// 八位无符号数。

	// Unsigned 8-bit integer.
	// @openapi examples=[255]
	Uint8 uint8
	// 十六位无符号数。

	// Unsigned 16-bit integer.
	// @openapi examples=[65535]
	Uint16 uint16
	// 三十二位无符号数。

	// Unsigned 32-bit integer.
	// @openapi examples=[4294967295]
	Uint32 uint32
	// 六十四位无符号数。

	// Unsigned 64-bit integer.
	// @openapi examples=[1099511627776]
	Uint64 uint64
	// 单精度数。

	// Single-precision floating-point number.
	// @openapi examples=[1.5]
	Float32 float32
	// 双精度数。

	// Double-precision floating-point number.
	// @openapi examples=[3.141592653589793]
	Float64 float64
	// 保留 JSON 数字语义。

	// Preserves JSON number semantics.
	// @openapi examples=[12.75]
	Number json.Number
}

// 嵌套地址对象。

// A nested address object.
type Address struct {
	// 城市。

	// City.
	// @openapi examples=["Hangzhou","Shanghai"]
	City string
	// 邮政编码。

	// Postal code.
	// @openapi examples=["310000"]
	PostalCode string
}

// 匿名嵌入后的字段依照真实 JSON 编码展开。

// Embedded fields are flattened according to the actual JSON encoding.
type AuditFields struct {
	// 创建时间。

	// Creation time.
	// @openapi examples=["2026-01-02T03:04:05Z"]
	CreatedAt time.Time
}

// 展示泛型实例的元素投影。

// Demonstrates element projection for a concrete generic type.
type Page[T any] struct {
	// 总数。

	// Total number of items.
	// @openapi examples=[1]
	Total int
	// 当前页元素。

	// Items on the current page.
	Items []T
}

// 可编码字段、集合、可空值、别名、枚举与泛型的完整示例。

// A complete example of encodable fields, collections, nullable values, aliases, enums, and generics.
type TypeExamples struct {
	AuditFields
	// 普通字符串。

	// Plain string.
	// @openapi examples=["hello","world"]
	Text string
	// 布尔值，false 示例不会因零值而丢失。

	// Boolean value; false examples are preserved even though false is the zero value.
	// @openapi examples=[false,true]
	Enabled bool
	// 字符串别名。

	// String alias.
	// @openapi examples=["example-label"]
	Label Label
	// 字符串枚举。

	// String enumeration.
	// @openapi examples=["admin","viewer"]
	Role Role
	// 数字枚举，包含零值。

	// Numeric enumeration, including zero.
	// @openapi examples=[0,2]
	State TaskState
	// 各类数字字段。

	// Fields covering the supported numeric types.
	Numbers NumberFields
	// 嵌套结构。

	// Nested structure.
	Address Address
	// 固定长度数组。

	// Fixed-length array.
	// @openapi examples=[[1,2,3]]
	Coordinates [3]int
	// 字符串切片。

	// String slice.
	// @openapi examples=[["Go","OpenAPI"]]
	Tags []string
	// 空切片与 nil 切片分别发送。

	// An empty slice is sent separately from a nil slice.
	// @openapi examples=[[]]
	EmptyTags []string
	// nil 切片。

	// Nil slice.
	// @openapi examples=[null]
	NilTags []string
	// 字符串键对象。

	// Object with string keys.
	// @openapi examples=[{"read":3,"write":1}]
	Counts map[string]int
	// 整数键在 JSON 中转换成字符串。

	// Integer keys are converted to strings in JSON.
	// @openapi examples=[{"1":"first","2":"second"}]
	Indexed map[int]string
	// 空映射。

	// Empty map.
	// @openapi examples=[{}]
	EmptyMap map[string]string
	// nil 映射。

	// Nil map.
	// @openapi examples=[null]
	NilMap map[string]string
	// 有值的指针。

	// Pointer containing a value.
	// @openapi examples=["Optional note"]
	Note *string
	// 显式 null 指针。

	// Pointer explicitly represented as null.
	// @openapi examples=[null]
	MissingNote *string
	// JSON 编码的字节切片采用 base64。

	// Byte slices use base64 when encoded as JSON.
	// @openapi examples=["aGVsbG8="]
	Bytes []byte
	// 原始 JSON 保持对象或数组语义。

	// Raw JSON preserves its object or array structure.
	// @openapi examples=[{"nested":{"value":1}}]
	Raw json.RawMessage
	// 动态 JSON 值，示例不被当作封闭类型证明。

	// Dynamic JSON value; examples do not imply a closed set of possible types.
	// @openapi examples=[{"kind":"dynamic","valid":true}]
	Dynamic any
	// 时间间隔在此 JSON codec 中按纳秒整数发送。

	// This JSON codec encodes durations as integer nanoseconds.
	// @openapi examples=[1500000000]
	Elapsed time.Duration
	// 泛型分页结果。

	// Generic paginated result.
	Users Page[User]
}

// 获取字段类型示例
// 展示各种可编码字段、字符串与数字枚举、嵌套结构及泛型，不执行持久化操作。

// Get field type examples
//
// Demonstrates encodable fields, string and numeric enums, nested structures, and generics without persisting data.
// @openapi tags=["Field types"]
func GetTypeExamples(c *gin.Context) {
	note := "可选备注"
	c.JSON(200, TypeExamples{
		AuditFields: AuditFields{CreatedAt: time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)},
		Text:        "你好", Enabled: false, Label: "示例标签", Role: RoleAdmin, State: StatePending,
		Numbers: NumberFields{Int: -42, Int8: -8, Int16: -1600, Int32: -32000, Int64: 1099511627776, Uint: 42, Uint8: 255, Uint16: 65535, Uint32: 4294967295, Uint64: 1099511627776, Float32: 1.5, Float64: 3.141592653589793, Number: json.Number("12.75")},
		Address: Address{City: "杭州", PostalCode: "310000"}, Coordinates: [3]int{1, 2, 3}, Tags: []string{"Go", "OpenAPI"}, EmptyTags: []string{}, NilTags: nil,
		Counts: map[string]int{"read": 3, "write": 1}, Indexed: map[int]string{1: "first", 2: "second"}, EmptyMap: map[string]string{}, NilMap: nil,
		Note: &note, MissingNote: nil, Bytes: []byte("hello"), Raw: json.RawMessage(`{"nested":{"value":1}}`), Dynamic: map[string]any{"kind": "dynamic", "valid": true}, Elapsed: 1500 * time.Millisecond,
		Users: Page[User]{Total: 1, Items: []User{{ID: 1024, Name: "alice"}}},
	})
}
