package compiler

import (
	"go/types"

	"github.com/openapi-golang/openapi"
	core "github.com/openapi-golang/openapi/compiler"
)

// Keep frontend binder choices local; only neutral conditions and projected contracts enter the Bundle.
type bindingDecision struct {
	when openapi.RequestCondition
	mode string
}

// Identify the explicit Form binder by package-scope identity, including propagated local aliases.
func isFormBinder(value core.Value) bool {
	object, ok := value.Object.(*types.Var)
	return ok && object.Pkg() != nil && object.Pkg().Path() == ginPackage+"/binding" && object.Parent() == object.Pkg().Scope() && object.Name() == "Form"
}

// Describe Gin Form's actual decision table for query, URL-encoded bodies, and multipart text fields.
func formDecisions() []bindingDecision {
	return []bindingDecision{
		{openapi.RequestCondition{MediaTypes: []string{"multipart/form-data"}}, "form-multipart"},
		{openapi.RequestCondition{Methods: []string{"POST", "PUT", "PATCH"}, MediaTypes: []string{"application/x-www-form-urlencoded"}}, "form-urlencoded"},
		{openapi.RequestCondition{ExceptMethods: []string{"POST", "PUT", "PATCH"}, MediaTypes: []string{"application/x-www-form-urlencoded"}}, "form-query"},
		{openapi.RequestCondition{ExceptMediaTypes: []string{"multipart/form-data", "application/x-www-form-urlencoded"}}, "form-query"},
	}
}

// Generate finite automatic-binding conditions, separating GET's Form rules from non-GET codec selection.
func automaticBinding(c core.CallContext, mandatory, formOnly bool) ([]core.CallOutcome, error) {
	var decisions []bindingDecision
	if formOnly {
		decisions = formDecisions()
	} else {
		unsupported := []string{"application/xml", "text/xml", "application/x-protobuf", "application/x-msgpack", "application/msgpack", "application/x-yaml", "application/yaml", "application/toml", "application/bson"}
		decisions = []bindingDecision{
			{openapi.RequestCondition{ExceptMethods: []string{"GET"}, MediaTypes: []string{"application/json"}}, "json"},
			{openapi.RequestCondition{ExceptMethods: []string{"GET"}, MediaTypes: []string{"multipart/form-data"}}, "multipart"},
			{openapi.RequestCondition{ExceptMethods: []string{"GET"}, MediaTypes: unsupported}, "unsupported"},
		}
		excluded := append([]string{"application/json", "multipart/form-data"}, unsupported...)
		scopes := []openapi.RequestCondition{{Methods: []string{"GET"}}, {ExceptMethods: []string{"GET"}, ExceptMediaTypes: excluded}}
		for _, scope := range scopes {
			for _, decision := range formDecisions() {
				when, ok, err := scope.Intersect(decision.when)
				if err != nil {
					return nil, err
				}
				if ok {
					decisions = append(decisions, bindingDecision{when, decision.mode})
				}
			}
		}
	}
	var outcomes []core.CallOutcome
	for _, decision := range decisions {
		effects := bindRequest(c, decision.mode, c.Arguments[0])
		if decision.mode == "unsupported" {
			effects = unresolved(c, "this automatically selected non-JSON codec requires an explicit wire mapping")
		}
		alternatives, err := bindingResults(c, effects, mandatory)
		if err != nil {
			return nil, err
		}
		for _, alternative := range alternatives {
			alternative.When = decision.when
			outcomes = append(outcomes, alternative)
		}
	}
	return outcomes, nil
}
