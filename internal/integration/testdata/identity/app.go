// Keep route identity fixtures independent from documentation registration.
package identity

import "github.com/gin-gonic/gin"

// Carry state without serialization or documentation tags.
type Reply struct{ Label string }

// Return a plain function response.
func Plain(c *gin.Context) { c.JSON(200, Reply{Label: "plain"}) }

// Return the same wire shape under another source symbol.
func Renamed(c *gin.Context) { c.JSON(200, Reply{Label: "renamed"}) }

// Return an unexported handler response.
func hidden(c *gin.Context) { c.JSON(200, Reply{Label: "private"}) }

// Capture one label; the project frontend explicitly nominates this constructor.
// @openapi response status=200 mediaType="application/json" type="Reply"
//
//go:noinline
func Factory(label string) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(200, Reply{Label: label}) }
}

// Keep receiver state distinct from the shared method implementation.
type Receiver struct {
	Label  string
	Status int
}

// Copy state when a value method is registered.
func (r Receiver) Value(c *gin.Context) { c.JSON(200, Reply{Label: r.Label}) }

// Observe later changes through a pointer method.
func (r *Receiver) Pointer(c *gin.Context) { c.JSON(200, Reply{Label: r.Label}) }

// An unknown receiver status cannot be rescued by a route-to-template binding.
func (r Receiver) UnknownStatus(c *gin.Context) { c.JSON(r.Status, Reply{Label: r.Label}) }

// This contract is a string for every instantiation although runtime symbols differ.
func Generic[T any](c *gin.Context) { c.String(200, "%T", *new(T)) }

// Unbound payload types remain unresolved even when the runtime instantiation is known.
func GenericJSON[T any](c *gin.Context) { var value T; c.JSON(200, value) }

// An explicit generic conversion must not borrow the untyped argument's wire identity.
func GenericConverted[T ~int](c *gin.Context) { c.JSON(200, T(1)) }

// Deliberately remain unregistered so unrelated models cannot enter selected documents.
func Unused(c *gin.Context) { c.JSON(200, struct{ UnusedPrivate bool }{true}) }

// Register ordinary aliases, closures, method values and instantiated generic functions.
func Router(prefix string) *gin.Engine {
	engine := gin.New()
	group := engine.Group(prefix)
	group.GET("/direct", Plain)
	alias := Plain
	group.GET("/alias", alias)
	group.GET("/private", hidden)
	for _, label := range []string{"alpha", "beta"} {
		group.GET("/factory/"+label, Factory(label))
	}
	a, b := Receiver{"value-alpha", 201}, Receiver{"value-beta", 202}
	group.GET("/value/alpha", a.Value)
	group.GET("/value/beta", b.Value)
	group.GET("/pointer/alpha", a.Pointer)
	group.GET("/pointer/beta", b.Pointer)
	a.Label, b.Label = "pointer-alpha", "pointer-beta"
	group.GET("/generic/int", Generic[int])
	group.GET("/generic/string", Generic[string])
	group.GET("/unknown/status", a.UnknownStatus)
	group.GET("/unknown/generic", GenericJSON[int])
	group.GET("/unknown/converted", GenericConverted[int])
	return engine
}
