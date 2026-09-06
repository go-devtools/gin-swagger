package compiler

import (
	"fmt"
	"go/types"
	"net/textproto"
	"reflect"
	"strings"

	core "github.com/openapi-golang/openapi/compiler"
	"github.com/openapi-golang/openapi/spec"
)

// Select explicit Gin text-binding field rules for reuse by project generation entry points.
type BindingCodec struct {
	// Mode is query, uri, header, form-post, or multipart.
	Mode string
}

// Include the mode in codec identity to isolate JSON and other location projections.
func (c BindingCodec) Name() string { return "gin-v1.12-binding-" + c.Mode + "-v1" }

// Validate the explicit mode and select the tag actually read by Gin.
func (c BindingCodec) fieldTag() (string, error) {
	switch c.Mode {
	case "query", "form", "form-post", "multipart":
		return "form", nil
	case "uri":
		return "uri", nil
	case "header":
		return "header", nil
	}
	return "", fmt.Errorf("gin.codec.mode: unknown binding mode %q", c.Mode)
}

// Follow Gin's recursive field traversal and existing names without imposing JSON field precedence.
func (c BindingCodec) Fields(root *types.Struct) ([]core.WireField, error) {
	tag, err := c.fieldTag()
	if err != nil {
		return nil, err
	}
	var fields []core.WireField
	names := map[string]bool{}
	visiting := map[*types.Struct]bool{}
	var visit func(*types.Struct, int) error
	visit = func(value *types.Struct, depth int) error {
		if depth > 32 || visiting[value] {
			return fmt.Errorf("gin.codec.fields: embedded field recursion exceeds the analysis scope")
		}
		visiting[value] = true
		defer delete(visiting, value)
		for i := 0; i < value.NumFields(); i++ {
			field := value.Field(i)
			tags := reflect.StructTag(value.Tag(i))
			if tags.Get(tag) == "-" {
				continue
			}
			if !field.Exported() && !field.Embedded() {
				continue
			}
			if tags.Get("binding") != "" || tags.Get("validate") != "" {
				return fmt.Errorf("gin.codec.validation: %s existing validation source requires a centralized declaration", field.Name())
			}
			name, options, _ := strings.Cut(tags.Get(tag), ",")
			if options != "" || tags.Get("parser") != "" || tags.Get("collection_format") != "" || tags.Get("time_format") != "" || tags.Get("time_utc") != "" || tags.Get("time_location") != "" {
				return fmt.Errorf("gin.codec.options: %s custom text binding options require a centralized rule", field.Name())
			}
			if name == "" {
				name = field.Name()
			}
			typ := types.Unalias(field.Type())
			for {
				pointer, ok := typ.(*types.Pointer)
				if !ok {
					break
				}
				typ = types.Unalias(pointer.Elem())
			}
			identity := types.TypeString(typ, func(p *types.Package) string { return p.Path() })
			if object, ok := typ.Underlying().(*types.Struct); ok {
				if field.Embedded() {
					if err := visit(object, depth+1); err != nil {
						return err
					}
					continue
				}
				if identity != "time.Time" && identity != "mime/multipart.FileHeader" {
					return fmt.Errorf("gin.codec.nested: %s JSON field and flat fallback require conditional projection", field.Name())
				}
			}
			if c.Mode == "header" || c.Mode == "uri" {
				switch typ.Underlying().(type) {
				case *types.Array, *types.Slice:
					return fmt.Errorf("gin.codec.serialization: %s repeated values cannot be represented as a comma-separated parameter", field.Name())
				}
			}
			if c.Mode == "header" {
				name = textproto.CanonicalMIMEHeaderKey(name)
			}
			if names[name] {
				return fmt.Errorf("gin.codec.collision: multiple fields consume the same wire name %s", name)
			}
			names[name] = true
			fields = append(fields, core.WireField{Name: name, Field: field})
		}
		return nil
	}
	if err := visit(root, 0); err != nil {
		return nil, err
	}
	return fields, nil
}

// Actual Gin text and multipart binding determines input types instead of JSON null or Base64 representations.
func (c BindingCodec) ProjectType(request core.ProjectionRequest, project func(types.Type) (*spec.Schema, error)) (*spec.Schema, bool, error) {
	if _, err := c.fieldTag(); err != nil {
		return nil, true, err
	}
	if request.Direction != core.Input {
		return nil, true, fmt.Errorf("gin.codec.direction: this binder only supports input")
	}
	typ := types.Unalias(request.Type)
	switch value := typ.(type) {
	case *types.Pointer:
		schema, err := project(value.Elem())
		return schema, true, err
	case *types.Named:
		identity := types.TypeString(value, func(p *types.Package) string { return p.Path() })
		switch identity {
		case "time.Time":
			schema := spec.Typed("string")
			schema.Format = "date-time"
			return schema, true, nil
		case "time.Duration":
			schema := spec.Typed("string")
			schema.Pattern = `^[-+]?(?:0|(?:(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:ns|us|µs|μs|ms|s|m|h))+)$`
			return schema, true, nil
		case "mime/multipart.FileHeader":
			if c.Mode != "multipart" {
				return nil, true, fmt.Errorf("gin.codec.file: file field requires a multipart binder")
			}
			return &spec.Schema{SchemaObject: &spec.SchemaObject{ContentMediaType: "application/octet-stream"}}, true, nil
		}
		if method, _, _ := types.LookupFieldOrMethod(types.NewPointer(value), true, nil, "UnmarshalParam"); method != nil {
			if signature, ok := method.Type().(*types.Signature); ok && signature.Params().Len() == 1 && signature.Results().Len() == 1 && types.Identical(signature.Params().At(0).Type(), types.Typ[types.String]) && types.Identical(signature.Results().At(0).Type(), types.Universe.Lookup("error").Type()) {
				return nil, true, fmt.Errorf("gin.codec.custom: %s UnmarshalParam requires a centralized TypeMapper", identity)
			}
		}
	case *types.Slice:
		item, err := project(value.Elem())
		schema := spec.Typed("array")
		schema.Items = item
		return schema, true, err
	case *types.Array:
		item, err := project(value.Elem())
		schema := spec.Typed("array")
		schema.Items = item
		schema.MinItems = spec.Set(uint64(value.Len()))
		schema.MaxItems = spec.Set(uint64(value.Len()))
		return schema, true, err
	case *types.Map, *types.Interface:
		return nil, true, fmt.Errorf("gin.codec.dynamic: dynamic objects and JSON text fields require explicit wire mappings")
	}
	return nil, false, nil
}
