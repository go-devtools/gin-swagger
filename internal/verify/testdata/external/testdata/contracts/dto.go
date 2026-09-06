// Keep DTO metadata in an imported package excluded from source-root patterns.
package contracts

import "github.com/gin-gonic/gin"

// A role with an explicitly closed value set.
// @openapi enum
type Role string

// Declare the known role values and their labels.
const (

	// Administrator.
	Admin Role = "admin"

	// Editor.
	Editor Role = "editor"
)

// Known constants do not close this ordinary named string.
type State string

// A known state remains an ordinary constant.
const Ready State = "ready"

// A tag-free request shared by actual binders.
type Request struct {

	// Display name from the imported DTO.
	// @openapi required minLength=3 examples=["Alice"]
	Name string

	// Selected role.
	Role Role

	// An open state.
	State State
}

// A generic envelope whose fields retain their declaration metadata.
type Envelope[T any] struct {

	// The actual payload from the imported envelope.
	// @openapi required
	Data T
}

// Malformed metadata is reported only when this dependency type is projected.
type Malformed struct {
	// @openapi examples=[
	Value string
}

// An incompatible source constraint retains a structured field location.
type Incompatible struct {
	// @openapi minimum=1
	Value string
}

// This dependency handler must not enter application candidate discovery.
func LibraryHandler(c *gin.Context) { c.Status(204) }
