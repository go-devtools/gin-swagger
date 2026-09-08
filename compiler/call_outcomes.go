package compiler

import (
	"go/types"

	core "github.com/openapi-golang/openapi/compiler"
)

// Model binding success and failure as finite alternatives correlating results with response commits.
func bindingOutcomes(c core.CallContext) ([]core.CallOutcome, error) {
	if c.Object == nil || c.Object.Pkg() == nil || c.Object.Pkg().Path() != ginPackage {
		return nil, nil
	}
	signature, ok := c.Object.Type().(*types.Signature)
	if !ok || signature.Recv() == nil || !isContext(signature.Recv().Type()) || len(c.Arguments) == 0 {
		return nil, nil
	}
	name := c.Object.Name()
	var effects []core.Effect
	mandatory := false
	switch name {
	case "ShouldBind", "Bind":
		return automaticBinding(c, name == "Bind", false)
	case "ShouldBindJSON", "ShouldBindBodyWithJSON", "BindJSON":
		effects = bindRequest(c, "json", c.Arguments[0])
		mandatory = name == "BindJSON"
	case "ShouldBindQuery", "BindQuery":
		effects = bindRequest(c, "query", c.Arguments[0])
		mandatory = name == "BindQuery"
	case "ShouldBindUri", "BindUri":
		effects = bindRequest(c, "uri", c.Arguments[0])
		mandatory = name == "BindUri"
	case "ShouldBindHeader", "BindHeader":
		effects = bindRequest(c, "header", c.Arguments[0])
		mandatory = name == "BindHeader"
	case "ShouldBindWith", "ShouldBindBodyWith", "MustBindWith":
		if len(c.Arguments) < 2 {
			return nil, nil
		}
		if name != "ShouldBindBodyWith" && isFormBinder(c.Arguments[1]) {
			return automaticBinding(c, name == "MustBindWith", true)
		}
		effects = explicitBinder(c, c.Arguments[1], c.Arguments[0], name == "ShouldBindBodyWith")
		mandatory = name == "MustBindWith"
	default:
		return nil, nil
	}
	return bindingResults(c, effects, mandatory)
}

// Correlate selected binding facts with success and committed-error results.
func bindingResults(c core.CallContext, effects []core.Effect, mandatory bool) ([]core.CallOutcome, error) {
	name := c.Object.Name()
	// Preserve unresolved binder diagnostics without inventing known success or failure facts.
	for _, effect := range effects {
		if effect.Kind == core.Unresolved {
			return []core.CallOutcome{{Results: []core.Value{{Unknown: true}}, Effects: effects}}, nil
		}
	}
	success := append([]core.Effect(nil), effects...)
	for i := range success {
		// JSON decoding and multipart framing cannot succeed on an empty stream. Form/query fields alone prove no body presence.
		success[i].NonEmptyBody = success[i].Kind == core.RequestBody && (success[i].MediaType == "application/json" || success[i].MediaType == "multipart/form-data")
	}
	outcomes := []core.CallOutcome{{Results: []core.Value{{Nil: true}}, Effects: success}}
	if !mandatory {
		return append(outcomes, core.CallOutcome{Results: []core.Value{{NonNil: true}}, Effects: effects}), nil
	}
	statuses := []string{"400"}
	canLimit := false
	for _, effect := range effects {
		canLimit = canLimit || effect.Kind == core.RequestBody
	}
	// Custom text decoders may also return MaxBytesError; the URI convenience method always commits 400.
	canLimit = canLimit || customBindingError(c.Arguments[0].Type, map[types.Type]bool{})
	if name != "BindUri" && canLimit {
		statuses = append(statuses, "413")
	}
	for _, status := range statuses {
		source := c.Source
		source.Kind, source.Rule = "derived", "gin."+name+".error."+status
		failure := append([]core.Effect(nil), effects...)
		failure = append(failure, core.Effect{Kind: core.ResponseCommit, Status: status, Source: source}, core.Effect{Kind: core.Abort, Source: source})
		outcomes = append(outcomes, core.CallOutcome{Results: []core.Value{{NonNil: true}}, Effects: failure})
	}
	return outcomes, nil
}

// Identify custom text decoders that can introduce arbitrary error types, visiting recursive types once.
func customBindingError(t types.Type, seen map[types.Type]bool) bool {
	if t == nil || seen[t] {
		return false
	}
	seen[t] = true
	t = types.Unalias(t)
	if pointer, ok := t.(*types.Pointer); ok {
		return customBindingError(pointer.Elem(), seen)
	}
	if method, _, _ := types.LookupFieldOrMethod(types.NewPointer(t), true, nil, "UnmarshalParam"); method != nil {
		return true
	}
	switch value := t.Underlying().(type) {
	case *types.Struct:
		for i := 0; i < value.NumFields(); i++ {
			field := value.Field(i)
			if (field.Exported() || field.Embedded()) && customBindingError(field.Type(), seen) {
				return true
			}
		}
	case *types.Slice:
		return customBindingError(value.Elem(), seen)
	case *types.Array:
		return customBindingError(value.Elem(), seen)
	}
	return false
}
