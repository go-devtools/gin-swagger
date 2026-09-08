package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	ginswagger "github.com/go-devtools/gin-swagger"
	"github.com/go-devtools/gin-swagger/examples/basic/internal/apidoc"
	"github.com/go-devtools/openapi"
	"github.com/go-devtools/openapi/spec"
	"github.com/go-devtools/openapi/swaggerui"
)

// Demonstrates string and numeric enums in the same request and response.
type EnumSelection struct {

	// User role: admin for administrators, editor for editors, and viewer for viewers.
	// @openapi required nonnull examples=["admin","editor","viewer"]
	Role Role

	// Task state: 0 for pending, 1 for running, and 2 for completed; defaults to 0 when omitted.
	// @openapi default=0 examples=[0,1,2]
	State TaskState
}

// Select a role and task state
//
// Both the request and response contain string and numeric enums. Use the Examples dropdown to choose from three complete requests. Invalid enum values return 400.
// @openapi tags=["Enums"]
func SelectEnums(c *gin.Context) {
	var req EnumSelection
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, APIError{Code: "INVALID_JSON", Message: "Invalid request body format"})
		return
	}
	validRole := req.Role == RoleAdmin || req.Role == RoleEditor || req.Role == RoleViewer
	validState := req.State == StatePending || req.State == StateRunning || req.State == StateDone
	if !validRole || !validState {
		c.JSON(400, APIError{Code: "INVALID_ENUM", Message: "Role or state is outside the allowed enum values"})
		return
	}
	c.JSON(200, req)
}

// A sample result available after authentication.
type AuthorizedExample struct {

	// Current demo user.
	// @openapi examples=["demo-user"]
	User string

	// Whether authentication succeeded.
	// @openapi examples=[true]
	Authorized bool
}

// Read protected content
//
// Click Authorize and enter the demo credentials. Missing or invalid credentials return 401; valid credentials return 200.
// @openapi tags=["Authentication"]
func GetAuthorizedExample(c *gin.Context) {
	c.JSON(200, AuthorizedExample{User: "demo-user", Authorized: true})
}

// Protects the Bearer demo group while preserving the existing user endpoint behavior.
func requireDemoBearer(c *gin.Context) {
	if c.GetHeader("Authorization") != "Bearer demo-token" {
		c.AbortWithStatusJSON(401, APIError{Code: "UNAUTHORIZED", Message: "Provide the demonstration Bearer token"})
		return
	}
	c.Next()
}

// Configures documentation tags and authentication at startup; Swagger tags are independent of Gin route groups.
func documentationConfig() ginswagger.Config {
	return ginswagger.Config{
		Path:         "/docs",
		DefaultGroup: "all",
		Groups: []ginswagger.DocumentGroup{
			{ID: "all", Name: "All endpoints"},
			{ID: "resources", Name: "Users and resources", Include: func(_, path string) bool {
				return path == "/users" || path == "/examples/items" || strings.HasPrefix(path, "/examples/items/")
			}},
			{ID: "types", Name: "Types and enums", Include: func(_, path string) bool { return path == "/examples/types" || path == "/examples/enums" }},
			{ID: "auth", Name: "Authentication", Include: func(_, path string) bool { return strings.HasPrefix(path, "/examples/auth/") }},
			{ID: "legacy", Name: "Legacy · Deprecated", Include: func(_, path string) bool { return strings.HasPrefix(path, "/examples/legacy/") }},
		},
		UI: swaggerui.Config{Title: "Gin Swagger Examples", DocExpansion: "list", OperationsSorter: "method"},
		OpenAPI: openapi.Config{
			Title: "User Service", Version: "1.0.0",
			Description: "Use Select a definition in the top-right corner to switch document groups. Use Authorize to set a Bearer token.",
			Tags: []spec.Tag{
				{Name: "Users", Description: "Basic business endpoints demonstrating source comments, request validation, and response models."},
				{Name: "HTTP methods", Description: "Real GET, POST, PUT, PATCH, and DELETE examples, including 201, 204, 400, and 404 responses."},
				{Name: "Legacy", Description: "Deprecated endpoints and their replacements, with the Deprecated label."},
				{Name: "Enums", Description: "String roles and numeric task states, with complete named request examples and invalid-value validation."},
				{Name: "Field types", Description: "Field types including primitives, arrays, maps, null values, nested objects, bytes, time, and generics."},
				{Name: "Authentication", Description: "Bearer authentication with a Gin route group, middleware, and lock icons. See Authorize for the demo token."},
			},
			Configure: configureExamples,
		},
	}
}

// Uses the public typed API to add middleware contracts and named examples, reusing the generated error model.
func configureExamples(doc *spec.OpenAPI) error {
	doc.Components.SecuritySchemes = map[string]spec.RefOr[spec.SecurityScheme]{
		"BearerAuth": spec.Inline(spec.SecurityScheme{Type: "http", Scheme: "bearer", Description: "For this local demo, enter demo-token without the Bearer prefix."}),
	}
	if item := doc.Paths["/examples/auth/bearer"]; item != nil && item.Get != nil {
		op := item.Get
		op.Security = spec.Set([]spec.SecurityRequirement{{"BearerAuth": []string{}}})
		op.Summary = "Read Bearer-protected content"

		// Reuses the generated APIError contract so each document group can be built independently.
		var unauthorized spec.Response
		for _, template := range apidoc.Bundle().Snapshot().Templates {
			if template.Key == openapi.OperationKey("github.com/go-devtools/gin-swagger/examples/basic.CreateUser") {
				unauthorized = *template.Operation.Responses["400"].Value
			}
		}
		if len(unauthorized.Content) == 0 {
			return fmt.Errorf("example is missing the source-derived error response")
		}
		unauthorized.Description = "Missing or invalid demo credentials."
		op.Responses["401"] = spec.Inline(unauthorized)
	}

	// Shows only tags used by operations in the selected document group.
	used := map[string]bool{}
	for _, path := range doc.Paths {
		for _, op := range []*spec.Operation{path.Get, path.Post, path.Put, path.Delete, path.Patch, path.Head, path.Options, path.Trace, path.Query} {
			if op != nil {

				// Uses standard English status descriptions for generated response placeholders.
				for status, response := range op.Responses {
					if response.Value != nil && response.Value.Description == "Response "+status {
						code, _ := strconv.Atoi(status)
						response.Value.Description = http.StatusText(code)
					}
				}
				for _, tag := range op.Tags {
					used[tag] = true
				}
			}
		}
	}
	var tags []spec.Tag
	for _, tag := range doc.Tags {
		if used[tag.Name] {
			tags = append(tags, tag)
		}
	}
	doc.Tags = tags
	if doc.Paths["/examples/enums"] == nil {
		return nil
	}
	media := doc.Paths["/examples/enums"].Post.RequestBody.Value.Content["application/json"].Value
	media.Examples = map[string]spec.RefOr[spec.Example]{
		"admin-pending":  spec.Inline(spec.Example{Summary: "Administrator · Pending", Value: spec.Set[any](EnumSelection{Role: RoleAdmin, State: StatePending})}),
		"editor-running": spec.Inline(spec.Example{Summary: "Editor · Running", Value: spec.Set[any](EnumSelection{Role: RoleEditor, State: StateRunning})}),
		"viewer-done":    spec.Inline(spec.Example{Summary: "Viewer · Completed", Value: spec.Set[any](EnumSelection{Role: RoleViewer, State: StateDone})}),
	}
	return nil
}
