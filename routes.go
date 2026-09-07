package ginswagger

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
)

// Compare complete public route multisets while allowing only Gin's documented colon unescaping.
// Function pointers corroborate an existing snapshot; they never identify closure or receiver state.
func registeredRoutes(engine *gin.Engine, saved gin.RoutesInfo) (gin.RoutesInfo, error) {
	current := engine.Routes()
	if saved == nil {
		return current, nil
	}
	if len(saved) != len(current) {
		return nil, fmt.Errorf("gin-swagger.routes.stale: registered route count changed; capture Engine.Routes after all business registration and before Run or ServeHTTP")
	}
	// Keep method, path and handler evidence together, including repeated collapsed paths.
	type evidence struct {
		method, path, handler string
		pointer               uintptr
	}
	key := func(route gin.RouteInfo) evidence {
		var pointer uintptr
		if route.HandlerFunc != nil {
			pointer = reflect.ValueOf(route.HandlerFunc).Pointer()
		}
		return evidence{route.Method, strings.ReplaceAll(route.Path, `\:`, ":"), route.Handler, pointer}
	}
	counts := make(map[evidence]int, len(current))
	for _, route := range current {
		counts[key(route)]++
	}
	for _, route := range saved {
		value := key(route)
		if value.pointer == 0 || counts[value] == 0 {
			return nil, fmt.Errorf("gin-swagger.routes.stale: route snapshot no longer matches the Engine: %s %s; capture routes after registration and preserve the original snapshot for initialized engines", route.Method, route.Path)
		}
		counts[value]--
	}
	return append(gin.RoutesInfo(nil), saved...), nil
}
