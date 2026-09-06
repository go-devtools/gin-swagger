package main

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
)

// The closed set of supported user roles.
// @openapi enum examples=["admin","editor","viewer"]
type Role string

// Demonstrates enum values with descriptions from source comments.
const (

	// Administrator
	RoleAdmin Role = "admin"

	// Editor
	RoleEditor Role = "editor"

	// Viewer
	RoleViewer Role = "viewer"
)

// Numeric enumeration of task states.
// @openapi enum examples=[0,1,2]
type TaskState int

// The complete set of task states.
const (

	// Pending
	StatePending TaskState = 0

	// Running
	StateRunning TaskState = 1

	// Completed
	StateDone TaskState = 2
)

// A business-specific string alias.
type Label = string

// Demonstrates Go numeric types that can be represented in JSON.
type NumberFields struct {

	// Signed machine-sized integer.
	// @openapi examples=[-42]
	Int int

	// Signed 8-bit integer.
	// @openapi examples=[-8]
	Int8 int8

	// Signed 16-bit integer.
	// @openapi examples=[-1600]
	Int16 int16

	// Signed 32-bit integer.
	// @openapi examples=[-32000]
	Int32 int32

	// Signed 64-bit integer.
	// @openapi examples=[1099511627776]
	Int64 int64

	// Unsigned machine-sized integer.
	// @openapi examples=[42]
	Uint uint

	// Unsigned 8-bit integer.
	// @openapi examples=[255]
	Uint8 uint8

	// Unsigned 16-bit integer.
	// @openapi examples=[65535]
	Uint16 uint16

	// Unsigned 32-bit integer.
	// @openapi examples=[4294967295]
	Uint32 uint32

	// Unsigned 64-bit integer.
	// @openapi examples=[1099511627776]
	Uint64 uint64

	// Single-precision floating-point number.
	// @openapi examples=[1.5]
	Float32 float32

	// Double-precision floating-point number.
	// @openapi examples=[3.141592653589793]
	Float64 float64

	// Preserves JSON number semantics.
	// @openapi examples=[12.75]
	Number json.Number
}

// A nested address object.
type Address struct {

	// City.
	// @openapi examples=["Hangzhou","Shanghai"]
	City string

	// Postal code.
	// @openapi examples=["310000"]
	PostalCode string
}

// Embedded fields are flattened according to the actual JSON encoding.
type AuditFields struct {

	// Creation time.
	// @openapi examples=["2026-01-02T03:04:05Z"]
	CreatedAt time.Time
}

// Demonstrates element projection for a concrete generic type.
type Page[T any] struct {

	// Total number of items.
	// @openapi examples=[1]
	Total int

	// Items on the current page.
	Items []T
}

// A complete example of encodable fields, collections, nullable values, aliases, enums, and generics.
type TypeExamples struct {
	AuditFields

	// Plain string.
	// @openapi examples=["hello","world"]
	Text string

	// Boolean value; false examples are preserved even though false is the zero value.
	// @openapi examples=[false,true]
	Enabled bool

	// String alias.
	// @openapi examples=["example-label"]
	Label Label

	// String enumeration.
	// @openapi examples=["admin","viewer"]
	Role Role

	// Numeric enumeration, including zero.
	// @openapi examples=[0,2]
	State TaskState

	// Fields covering the supported numeric types.
	Numbers NumberFields

	// Nested structure.
	Address Address

	// Fixed-length array.
	// @openapi examples=[[1,2,3]]
	Coordinates [3]int

	// String slice.
	// @openapi examples=[["Go","OpenAPI"]]
	Tags []string

	// An empty slice is sent separately from a nil slice.
	// @openapi examples=[[]]
	EmptyTags []string

	// Nil slice.
	// @openapi examples=[null]
	NilTags []string

	// Object with string keys.
	// @openapi examples=[{"read":3,"write":1}]
	Counts map[string]int

	// Integer keys are converted to strings in JSON.
	// @openapi examples=[{"1":"first","2":"second"}]
	Indexed map[int]string

	// Empty map.
	// @openapi examples=[{}]
	EmptyMap map[string]string

	// Nil map.
	// @openapi examples=[null]
	NilMap map[string]string

	// Pointer containing a value.
	// @openapi examples=["Optional note"]
	Note *string

	// Pointer explicitly represented as null.
	// @openapi examples=[null]
	MissingNote *string

	// Byte slices use base64 when encoded as JSON.
	// @openapi examples=["aGVsbG8="]
	Bytes []byte

	// Raw JSON preserves its object or array structure.
	// @openapi examples=[{"nested":{"value":1}}]
	Raw json.RawMessage

	// Dynamic JSON value; examples do not imply a closed set of possible types.
	// @openapi examples=[{"kind":"dynamic","valid":true}]
	Dynamic any

	// This JSON codec encodes durations as integer nanoseconds.
	// @openapi examples=[1500000000]
	Elapsed time.Duration

	// Generic paginated result.
	Users Page[User]
}

// Get field type examples
//
// Demonstrates encodable fields, string and numeric enums, nested structures, and generics without persisting data.
// @openapi tags=["Field types"]
func GetTypeExamples(c *gin.Context) {
	note := "Optional note"
	c.JSON(200, TypeExamples{
		AuditFields: AuditFields{CreatedAt: time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)},
		Text:        "Hello", Enabled: false, Label: "Example label", Role: RoleAdmin, State: StatePending,
		Numbers: NumberFields{Int: -42, Int8: -8, Int16: -1600, Int32: -32000, Int64: 1099511627776, Uint: 42, Uint8: 255, Uint16: 65535, Uint32: 4294967295, Uint64: 1099511627776, Float32: 1.5, Float64: 3.141592653589793, Number: json.Number("12.75")},
		Address: Address{City: "Hangzhou", PostalCode: "310000"}, Coordinates: [3]int{1, 2, 3}, Tags: []string{"Go", "OpenAPI"}, EmptyTags: []string{}, NilTags: nil,
		Counts: map[string]int{"read": 3, "write": 1}, Indexed: map[int]string{1: "first", 2: "second"}, EmptyMap: map[string]string{}, NilMap: nil,
		Note: &note, MissingNote: nil, Bytes: []byte("hello"), Raw: json.RawMessage(`{"nested":{"value":1}}`), Dynamic: map[string]any{"kind": "dynamic", "valid": true}, Elapsed: 1500 * time.Millisecond,
		Users: Page[User]{Total: 1, Items: []User{{ID: 1024, Name: "alice"}}},
	})
}
