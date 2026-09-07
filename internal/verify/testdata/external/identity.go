// Keep route identity fixtures independent from documentation registration.
package main

import "github.com/gin-gonic/gin"

// Carry state without serialization or documentation tags.
type IdentityReply struct{ Label string }

// Return a plain function response.
func IdentityPlain(c *gin.Context) { c.JSON(200, IdentityReply{Label: "plain"}) }

// Return the same wire shape under another source symbol.
func IdentityRenamed(c *gin.Context) { c.JSON(200, IdentityReply{Label: "renamed"}) }

// Return an unexported handler response.
func identityHidden(c *gin.Context) { c.JSON(200, IdentityReply{Label: "private"}) }

// Capture one label; the project frontend explicitly nominates this constructor.
// @openapi response status=200 mediaType="application/json" type="IdentityReply"
//
//go:noinline
func IdentityFactory(label string) gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(200, IdentityReply{Label: label}) }
}

// Keep receiver state distinct from the shared method implementation.
type IdentityReceiver struct {
	Label  string
	Status int
}

// Copy state when a value method is registered.
func (r IdentityReceiver) Value(c *gin.Context) { c.JSON(200, IdentityReply{Label: r.Label}) }

// Observe later changes through a pointer method.
func (r *IdentityReceiver) Pointer(c *gin.Context) { c.JSON(200, IdentityReply{Label: r.Label}) }

// An unknown receiver status cannot be rescued by a route-to-template binding.
func (r IdentityReceiver) UnknownStatus(c *gin.Context) {
	c.JSON(r.Status, IdentityReply{Label: r.Label})
}

// This contract is a string for every instantiation although runtime symbols differ.
func IdentityGeneric[T any](c *gin.Context) { c.String(200, "%T", *new(T)) }

// Unbound payload types remain unresolved even when the runtime instantiation is known.
func IdentityGenericJSON[T any](c *gin.Context) { var value T; c.JSON(200, value) }

// An explicit generic conversion must not borrow the untyped argument's wire identity.
func IdentityGenericConverted[T ~int](c *gin.Context) { c.JSON(200, T(1)) }

// Deliberately remain unregistered so unrelated models cannot enter selected documents.
func IdentityUnused(c *gin.Context) { c.JSON(200, struct{ UnusedPrivate bool }{true}) }

// Register ordinary aliases, closures, method values and instantiated generic functions.
func identityRouter(prefix string) *gin.Engine {
	engine := gin.New()
	group := engine.Group(prefix)
	group.GET("/direct", IdentityPlain)
	alias := IdentityPlain
	group.GET("/alias", alias)
	group.GET("/private", identityHidden)
	for _, label := range []string{"alpha", "beta"} {
		group.GET("/factory/"+label, IdentityFactory(label))
	}
	a, b := IdentityReceiver{"value-alpha", 201}, IdentityReceiver{"value-beta", 202}
	group.GET("/value/alpha", a.Value)
	group.GET("/value/beta", b.Value)
	group.GET("/pointer/alpha", a.Pointer)
	group.GET("/pointer/beta", b.Pointer)
	a.Label, b.Label = "pointer-alpha", "pointer-beta"
	group.GET("/generic/int", IdentityGeneric[int])
	group.GET("/generic/string", IdentityGeneric[string])
	group.GET("/unknown/status", a.UnknownStatus)
	group.GET("/unknown/generic", IdentityGenericJSON[int])
	group.GET("/unknown/converted", IdentityGenericConverted[int])
	return engine
}
