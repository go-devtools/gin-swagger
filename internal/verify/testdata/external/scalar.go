// Exercise scalar request sources without adding tags or replacing business handlers.
package main

import "github.com/gin-gonic/gin"

// ScalarResult preserves actual values and distinguishes missing inputs from empty values.
type ScalarResult struct {
	// ID is read from the registered route.
	ID string
	// Query is the first query value.
	Query string
	// Optional retains the value returned with a presence flag.
	Optional string
	// Present reports query presence independently of the value.
	Present bool
	// Default retains Gin's missing-only fallback.
	Default string
	// Header is the first value from the case-insensitive header lookup.
	Header string
	// Cookie retains Gin's decoded cookie value.
	Cookie string
	// CookieError records the actual lookup error.
	CookieError bool
}

// ScalarReads accepts missing values while exposing the actual cookie lookup result.
func ScalarReads(c *gin.Context) {
	value, present := c.GetQuery("optional")
	cookie, err := c.Cookie("session")
	c.JSON(200, ScalarResult{ID: c.Param("id"), Query: c.Query("q"), Optional: value, Present: present, Default: c.DefaultQuery("alias", "guest"), Header: c.GetHeader("x-probe"), Cookie: cookie, CookieError: err != nil})
}

// CheckedCookie returns only the application-selected status for a missing cookie.
func CheckedCookie(c *gin.Context) {
	value, err := c.Cookie("session")
	if err != nil {
		c.String(401, "missing cookie")
		return
	}
	c.JSON(200, value)
}
