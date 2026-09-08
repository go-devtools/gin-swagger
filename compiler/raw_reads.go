package compiler

import (
	"go/constant"
	"go/types"

	"github.com/openapi-golang/openapi"
	core "github.com/openapi-golang/openapi/compiler"
	"github.com/openapi-golang/openapi/spec"
)

// Translate constant field reads into neutral wire representations while preserving actual Go result signatures.
func rawReadOutcomes(c core.CallContext) ([]core.CallOutcome, error) {
	if c.Object == nil || c.Object.Pkg() == nil || c.Object.Pkg().Path() != ginPackage {
		return nil, nil
	}
	signature, ok := c.Object.Type().(*types.Signature)
	if !ok || signature.Recv() == nil || !isContext(signature.Recv().Type()) {
		return nil, nil
	}
	name := c.Object.Name()
	// Application-context reads and file saving are not HTTP writes; propagate only known result shapes.
	if name == "GetString" {
		return []core.CallOutcome{{Results: []core.Value{{}}}}, nil
	}
	if name == "SaveUploadedFile" {
		if len(c.Arguments) > 0 && c.Arguments[0].Nil {
			return []core.CallOutcome{{Results: []core.Value{{Unknown: true}}, Effects: unresolved(c, "Save target file pointer is nil; a normal return contract cannot be inferred")}}, nil
		}
		return []core.CallOutcome{{Results: []core.Value{{Nil: true}}}, {Results: []core.Value{{NonNil: true}}}}, nil
	}
	form, file, array, dictionary := false, false, false, false
	location := "query"
	switch name {
	case "PostForm", "DefaultPostForm", "GetPostForm":
		form = true
	case "PostFormArray", "GetPostFormArray":
		form, array = true, true
	case "PostFormMap", "GetPostFormMap":
		form, dictionary = true, true
	case "FormFile":
		form, file = true, true
	case "Param":
		location = "path"
	case "Query", "GetQuery", "DefaultQuery":
	case "QueryArray", "GetQueryArray":
		array = true
	case "QueryMap", "GetQueryMap":
		dictionary = true
	case "GetHeader":
		location = "header"
	case "Cookie":
		location = "cookie"
	default:
		return nil, nil
	}
	values := make([]core.Value, signature.Results().Len())
	for i := range values {
		values[i].Type = signature.Results().At(i).Type()
	}
	if len(c.Arguments) == 0 || c.Arguments[0].Constant == nil || c.Arguments[0].Constant.Kind() != constant.String || literal(c.Arguments[0]) == "" {
		return []core.CallOutcome{{Results: values, Effects: unresolved(c, "field name must be an evaluable nonempty string constant")}}, nil
	}
	schema := spec.Typed("string")
	style := "form"
	if array {
		schema = spec.Typed("array")
		schema.Items = spec.Typed("string")
	}
	if dictionary {
		schema = spec.Typed("object")
		schema.AdditionalProperties = spec.Typed("string")
		style = "deepObject"
	}
	if (name == "DefaultPostForm" || name == "DefaultQuery") && len(c.Arguments) > 1 && c.Arguments[1].Constant != nil && c.Arguments[1].Constant.Kind() == constant.String {
		schema.Default = spec.Set[any](literal(c.Arguments[1]))
	}
	source := c.Source
	source.Kind, source.Rule = "derived", "gin."+name
	effect := core.Effect{Kind: core.ParameterRead, Name: literal(c.Arguments[0]), In: location, WireSchema: schema, Style: style, Explode: spec.Set(true), Source: source}
	if location == "path" || location == "header" {
		effect.Style = "simple"
		effect.Explode = spec.Set(false)
	}
	if !form {
		return []core.CallOutcome{{Results: values, Effects: []core.Effect{effect}}}, nil
	}
	effect.Kind, effect.In = core.RequestField, ""
	effect.Encoding = &spec.Encoding{Style: style, Explode: spec.Set(true)}
	effect.MediaType = "multipart/form-data"
	if file {
		effect.WireSchema = &spec.Schema{SchemaObject: &spec.SchemaObject{ContentMediaType: "application/octet-stream"}}
		effect.Encoding = &spec.Encoding{ContentType: "application/octet-stream"}
		// File reads do not commit HTTP errors; nil and non-nil results only control subsequent business branches.
		success := effect
		success.NonEmptyBody = true
		return []core.CallOutcome{{Results: []core.Value{{NonNil: true}, {Nil: true}}, Effects: []core.Effect{success}}, {Results: []core.Value{{Nil: true}, {NonNil: true}}, Effects: []core.Effect{effect}}}, nil
	}
	encoded := effect
	encoded.MediaType = "application/x-www-form-urlencoded"
	encoded.Source.Rule += ".body-only.urlencoded"
	effect.Source.Rule += ".body-only.multipart"
	// net/http reads URL-encoded bodies only for POST, PUT, and PATCH; multipart has no such method restriction.
	return []core.CallOutcome{
		{When: openapi.RequestCondition{Methods: []string{"POST", "PUT", "PATCH"}}, Results: values, Effects: []core.Effect{encoded, effect}},
		{When: openapi.RequestCondition{ExceptMethods: []string{"POST", "PUT", "PATCH"}}, Results: values, Effects: []core.Effect{effect}},
	}, nil
}

// Dispatch state-dependent redirects before raw reads and finite binder outcomes.
func requestOutcomes(c core.CallContext) ([]core.CallOutcome, error) {
	results, err := streamWriterOutcomes(c)
	if err != nil || len(results) > 0 {
		return results, err
	}
	results, err = redirectOutcomes(c)
	if err != nil || len(results) > 0 {
		return results, err
	}
	results, err = rawReadOutcomes(c)
	if err != nil || len(results) > 0 {
		return results, err
	}
	return bindingOutcomes(c)
}
