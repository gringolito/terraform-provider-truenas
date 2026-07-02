package codegen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"unicode"
)

// Registry is the method registry as returned by core.get_methods.
type Registry map[string]MethodDef

// MethodDef describes a single middleware method.
type MethodDef struct {
	Description string   `json:"description"`
	Accepts     []Schema `json:"accepts"`
	Returns     []Schema `json:"returns"`
}

// Schema is a TrueNAS-flavoured JSON-Schema node.
//
// TrueNAS deviates from standard JSON-Schema in two ways:
//   - "_required_" is a per-property boolean (true = required, false = optional).
//   - "items" is always a list (even when it carries a single type), not a bare schema object.
//
// The standard "required" string list still appears on object schemas and carries the
// same meaning; both forms are checked when deciding whether to emit omitempty.
type Schema struct {
	Type         string            `json:"type"`
	Format       string            `json:"format"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Properties   map[string]Schema `json:"properties"`
	Items        []Schema          `json:"items"` // TrueNAS always emits this as a list
	AnyOf        []Schema          `json:"anyOf"`
	OneOf        []Schema          `json:"oneOf"`
	AllOf        []Schema          `json:"allOf"`
	Enum         []json.RawMessage `json:"enum"`
	Const        json.RawMessage   `json:"const"`
	Required     []string          `json:"required"`   // standard JSON-Schema required list on objects
	RequiredProp *bool             `json:"_required_"` // per-property boolean (TrueNAS extension)
	AttrsOrder   []string          `json:"_attrs_order_"`
}

// BranchSelector picks one branch of a discriminated-union accepts entry (an
// anyOf of objects that share a const-valued discriminator field) by value.
type BranchSelector struct {
	Field string
	Value string
}

// ParseRegistry parses a registry JSON snapshot.
func ParseRegistry(data []byte) (Registry, error) {
	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}
	return reg, nil
}

// FilterByNamespaces returns a new registry containing only methods
// whose namespace exactly matches one of the given namespaces.
func FilterByNamespaces(reg Registry, namespaces []string) Registry {
	allowed := make(map[string]bool, len(namespaces))
	for _, ns := range namespaces {
		allowed[ns] = true
	}
	filtered := make(Registry)
	for method, def := range reg {
		if allowed[methodNamespace(method)] {
			filtered[method] = def
		}
	}
	return filtered
}

// Generate writes one typed Go source file per namespace to outDir.
// branchSelectors maps a fully-qualified method name (e.g. "pool.dataset.create")
// to the discriminator branch to select from a discriminated-union accepts entry.
func Generate(reg Registry, namespaces []string, outDir string, branchSelectors map[string]BranchSelector) error {
	for _, ns := range namespaces {
		src, err := GenerateNamespace(reg, ns, branchSelectors)
		if err != nil {
			return fmt.Errorf("namespace %s: %w", ns, err)
		}
		outPath := filepath.Join(outDir, namespaceToFile(ns))
		if err := os.WriteFile(outPath, src, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", outPath, err)
		}
	}
	return nil
}

// GenerateNamespace generates the Go source for a single namespace.
func GenerateNamespace(reg Registry, ns string, branchSelectors map[string]BranchSelector) ([]byte, error) {
	var methods []string
	for method := range reg {
		if methodNamespace(method) == ns {
			methods = append(methods, method)
		}
	}
	if len(methods) == 0 {
		return nil, fmt.Errorf("no methods found for namespace %q", ns)
	}
	sort.Strings(methods)
	return generateFile(ns, methods, reg, branchSelectors)
}

// ---- naming helpers --------------------------------------------------------

func methodNamespace(method string) string {
	idx := strings.LastIndex(method, ".")
	if idx < 0 {
		return method
	}
	return method[:idx]
}

func methodVerb(method string) string {
	idx := strings.LastIndex(method, ".")
	if idx < 0 {
		return method
	}
	return method[idx+1:]
}

func namespaceToFile(ns string) string {
	return strings.ReplaceAll(ns, ".", "_") + "_gen.go"
}

// namespaceToPrefix converts a namespace to a PascalCase Go identifier prefix.
// "pool.dataset" → "PoolDataset", "sharing.nfs" → "SharingNfs".
func namespaceToPrefix(ns string) string {
	parts := strings.Split(strings.ReplaceAll(ns, ".", "_"), "_")
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(capitalize(p))
	}
	return b.String()
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func snakeToPascal(s string) string {
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, "-", "_") // kebab-case property names (e.g. AES-128-CCM)
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(capitalize(p))
	}
	return b.String()
}

// ---- schema helpers --------------------------------------------------------

func isNullable(s Schema) bool {
	if len(s.AnyOf) != 2 {
		return false
	}
	return s.AnyOf[0].Type == "null" || s.AnyOf[1].Type == "null"
}

func nonNullBranch(s Schema) *Schema {
	if s.AnyOf[0].Type == "null" {
		return &s.AnyOf[1]
	}
	return &s.AnyOf[0]
}

func isPolymorphic(s Schema) bool {
	if len(s.OneOf) > 0 {
		return true
	}
	if len(s.AnyOf) > 2 {
		return true
	}
	if len(s.AnyOf) == 2 && !isNullable(s) {
		return true
	}
	return false
}

func isObjectLike(s Schema) bool {
	return s.Type == "object" || len(s.AllOf) > 0
}

// isIDArg reports whether a schema node can serve as an id-like leading
// positional accepts argument. Most namespaces key on an integer internal id
// (user, group); pool.dataset keys on its string ZFS path instead.
func isIDArg(s Schema) bool {
	return s.Type == "integer" || s.Type == "string"
}

// zfsPropertyFields is the fixed property set of the recurring
// {value, rawvalue, parsed, source, source_info} shape TrueNAS uses for every
// ZFS dataset property returned by pool.dataset.get_instance.
var zfsPropertyFields = map[string]bool{
	"value":       true,
	"rawvalue":    true,
	"parsed":      true,
	"source":      true,
	"source_info": true,
}

// isZFSPropertyShape reports whether s is exactly the ZFS property object
// shape, regardless of field ordering.
func isZFSPropertyShape(s Schema) bool {
	if s.Type != "object" || len(s.Properties) != len(zfsPropertyFields) {
		return false
	}
	for name := range s.Properties {
		if !zfsPropertyFields[name] {
			return false
		}
	}
	return true
}

// structUsesZFSProperty reports whether any field of s is ZFS-property-shaped.
func structUsesZFSProperty(s Schema) bool {
	s = resolveAllOf(s)
	for _, field := range s.Properties {
		if isZFSPropertyShape(field) {
			return true
		}
	}
	return false
}

// discriminatorField returns the property name that carries a distinct const
// value on every branch of s.AnyOf, or "" if s is not a discriminated union
// of object branches.
func discriminatorField(s Schema) string {
	if len(s.AnyOf) < 2 {
		return ""
	}
	for _, branch := range s.AnyOf {
		if !isObjectLike(branch) {
			return ""
		}
	}
	counts := make(map[string]int)
	for _, branch := range s.AnyOf {
		for name, prop := range branch.Properties {
			if len(prop.Const) > 0 {
				counts[name]++
			}
		}
	}
	for name, n := range counts {
		if n == len(s.AnyOf) {
			return name
		}
	}
	return ""
}

// selectBranch returns the branch of the discriminated union s.AnyOf whose
// field property's const equals value.
func selectBranch(s Schema, field, value string) (Schema, error) {
	for _, branch := range s.AnyOf {
		prop, ok := branch.Properties[field]
		if !ok || len(prop.Const) == 0 {
			continue
		}
		var constVal string
		if err := json.Unmarshal(prop.Const, &constVal); err != nil {
			continue
		}
		if constVal == value {
			return branch, nil
		}
	}
	return Schema{}, fmt.Errorf("no anyOf branch has %s=%q", field, value)
}

// isRequiredArg reports whether an accepts-list entry is required
// (i.e. the caller must pass it, it is not an optional trailing argument).
func isRequiredArg(s Schema) bool {
	return s.RequiredProp != nil && *s.RequiredProp
}

// schemaToGoType maps a schema node to a Go type string suitable for struct fields.
func schemaToGoType(s Schema) string {
	if isNullable(s) {
		return "*" + schemaToGoType(*nonNullBranch(s))
	}
	if isPolymorphic(s) || len(s.OneOf) > 0 {
		return "json.RawMessage"
	}
	if len(s.AllOf) > 0 {
		// allOf handled as merged struct at the call site; use raw as fallback here.
		return "json.RawMessage"
	}
	switch s.Type {
	case "integer":
		return "int64"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "string":
		// TrueNAS serializes datetimes over the wire as a Mongo-style
		// {"$date": <epoch_ms>} object, not an ISO string, so format:date-time
		// fields decode into the DateTime type (see datetime.go), not string.
		if s.Format == "date-time" {
			return "DateTime"
		}
		return "string"
	case "array":
		if len(s.Items) > 0 {
			return "[]" + schemaToGoType(s.Items[0])
		}
		return "[]json.RawMessage"
	case "object":
		if isZFSPropertyShape(s) {
			return "ZFSProperty"
		}
		return "map[string]json.RawMessage"
	default:
		return "json.RawMessage"
	}
}

// resolveAllOf flattens allOf compositions into a single schema.
func resolveAllOf(s Schema) Schema {
	if len(s.AllOf) == 0 {
		return s
	}
	merged := Schema{
		Properties: make(map[string]Schema),
	}
	for _, sub := range s.AllOf {
		sub = resolveAllOf(sub)
		maps.Copy(merged.Properties, sub.Properties)
		merged.Required = append(merged.Required, sub.Required...)
		merged.AttrsOrder = append(merged.AttrsOrder, sub.AttrsOrder...)
	}
	maps.Copy(merged.Properties, s.Properties)
	merged.Required = append(merged.Required, s.Required...)
	if len(merged.AttrsOrder) == 0 {
		merged.AttrsOrder = s.AttrsOrder
	}
	return merged
}

// isFieldRequired returns true if fieldName is required on schema s, checking
// both the standard "required" list and the per-property "_required_" boolean.
func isFieldRequired(s Schema, fieldName string, fieldSchema Schema) bool {
	if slices.Contains(s.Required, fieldName) {
		return true
	}
	return fieldSchema.RequiredProp != nil && *fieldSchema.RequiredProp
}

func fieldOrder(s Schema) []string {
	if len(s.AttrsOrder) > 0 {
		seen := make(map[string]bool)
		var fields []string
		for _, f := range s.AttrsOrder {
			if _, ok := s.Properties[f]; ok {
				fields = append(fields, f)
				seen[f] = true
			}
		}
		for f := range s.Properties {
			if !seen[f] {
				fields = append(fields, f)
			}
		}
		return fields
	}
	var fields []string
	for f := range s.Properties {
		fields = append(fields, f)
	}
	sort.Strings(fields)
	return fields
}

// ---- method classification -------------------------------------------------

// standardCRUDVerbs are the verbs that share the namespace entity struct as their return type.
// Any other verb that returns an object gets its own "<FuncName>Result" struct.
var standardCRUDVerbs = map[string]bool{
	"create":       true,
	"update":       true,
	"get_instance": true,
	"query":        true,
}

type methodSig struct {
	method           string
	funcName         string
	hasIDArg         bool
	idType           string // Go type of the id positional arg; only meaningful when hasIDArg
	hasQueryFilters  bool
	isUpdateArgs     bool
	argsStructName   string
	argsSchema       *Schema
	returnStructName string
	returnSchema     *Schema
	returnType       string // Go type expression; "" means the function returns only error
}

func classifyMethod(method string, def MethodDef, prefix string, branchSelectors map[string]BranchSelector) (methodSig, error) {
	verb := methodVerb(method)
	funcName := prefix + snakeToPascal(verb)
	sig := methodSig{method: method, funcName: funcName}

	// Classify accepts.
	//
	// Pattern 1: no arguments.
	// Pattern 2: single required object  →  (ctx, c, args ArgsType)
	// Pattern 2b: single discriminated-union (anyOf of const-tagged objects)  →
	//             (ctx, c, args ArgsType), branch picked by configured BranchSelector
	//             (e.g. pool.dataset.create's FILESYSTEM/VOLUME split)
	// Pattern 3: id-like (integer or string) first, required object second  →
	//            (ctx, c, id IDType, args ArgsType) (the [id, patch] update pattern)
	// Pattern 4: id-like first, optional or absent second  →  (ctx, c, id IDType)
	//            (get_instance, delete — optional trailing options are dropped)
	// Pattern 5: query verb with optional array first arg  →  (ctx, c, filters ...QueryFilter)
	switch {
	case len(def.Accepts) == 0:
		// no args

	case verb == "query" && len(def.Accepts) >= 1 && def.Accepts[0].Type == "array":
		sig.hasQueryFilters = true

	case len(def.Accepts) == 1 && discriminatorField(def.Accepts[0]) != "":
		field := discriminatorField(def.Accepts[0])
		sel, ok := branchSelectors[method]
		if !ok {
			return sig, fmt.Errorf("%s: accepts a discriminated union on %q but no branch selector is configured", method, field)
		}
		branch, err := selectBranch(def.Accepts[0], sel.Field, sel.Value)
		if err != nil {
			return sig, fmt.Errorf("%s: %w", method, err)
		}
		sig.argsSchema = &branch
		sig.argsStructName = funcName + "Args"

	case len(def.Accepts) == 1 && isObjectLike(def.Accepts[0]):
		a := def.Accepts[0]
		sig.argsSchema = &a
		// Use funcName+"Args" (not the schema title) to avoid clashing with the
		// function name itself (e.g. title "user_create" → "UserCreate" == funcName).
		sig.argsStructName = funcName + "Args"

	case len(def.Accepts) >= 2 && isIDArg(def.Accepts[0]):
		sig.hasIDArg = true
		sig.idType = schemaToGoType(def.Accepts[0])
		second := def.Accepts[1]
		if isObjectLike(second) && isRequiredArg(second) {
			// Required object: the [id, patch] pattern.
			sig.argsSchema = &second
			sig.argsStructName = funcName + "Args"
			sig.isUpdateArgs = verb == "update"
		}
		// Optional second arg (query options etc.) is dropped from the signature.

	case len(def.Accepts) >= 1 && isIDArg(def.Accepts[0]):
		sig.hasIDArg = true
		sig.idType = schemaToGoType(def.Accepts[0])
	}

	// Classify returns.
	if len(def.Returns) == 0 {
		return sig, nil
	}
	ret := def.Returns[0]
	switch {
	case isObjectLike(ret):
		// Standard CRUD verbs share the namespace entity struct (e.g. Group, User).
		// Non-standard verbs get their own "<FuncName>Result" struct so they don't
		// clobber the entity struct built from get_instance.
		structName := prefix
		if !standardCRUDVerbs[verb] {
			structName = funcName + "Result"
		}
		sig.returnStructName = structName
		sig.returnSchema = &ret
		sig.returnType = "*" + structName

	case ret.Type == "array":
		if len(ret.Items) > 0 && isObjectLike(ret.Items[0]) {
			structName := prefix
			if !standardCRUDVerbs[verb] {
				structName = funcName + "Result"
			}
			sig.returnStructName = structName
			item := ret.Items[0]
			sig.returnSchema = &item
			sig.returnType = "[]*" + structName
		} else {
			sig.returnType = "json.RawMessage"
		}

	case isPolymorphic(ret) || len(ret.AnyOf) > 0 || len(ret.OneOf) > 0:
		sig.returnType = "json.RawMessage"

	case ret.Type == "integer":
		if verb == "delete" {
			sig.returnType = "" // discard: delete returns are never useful
		} else {
			sig.returnType = "int64"
		}

	case ret.Type == "string":
		sig.returnType = "string"

	case ret.Type == "boolean", ret.Type == "null", ret.Type == "":
		sig.returnType = "" // discard
	}

	return sig, nil
}

// ---- code generation -------------------------------------------------------

func generateFile(ns string, methods []string, reg Registry, branchSelectors map[string]BranchSelector) ([]byte, error) {
	prefix := namespaceToPrefix(ns)

	var sigs []methodSig
	for _, m := range methods {
		sig, err := classifyMethod(m, reg[m], prefix, branchSelectors)
		if err != nil {
			return nil, err
		}
		sigs = append(sigs, sig)
	}

	needsJSON := false
	needsZFSProperty := false
	for _, s := range sigs {
		if s.returnType != "" || s.argsStructName != "" {
			needsJSON = true
		}
		if s.argsSchema != nil && structNeedsJSON(*s.argsSchema) {
			needsJSON = true
		}
		if s.returnSchema != nil && structNeedsJSON(*s.returnSchema) {
			needsJSON = true
		}
		if s.returnSchema != nil && structUsesZFSProperty(*s.returnSchema) {
			needsZFSProperty = true
		}
		if s.argsSchema != nil && structUsesZFSProperty(*s.argsSchema) {
			needsZFSProperty = true
		}
	}
	if needsZFSProperty {
		needsJSON = true
	}

	var buf bytes.Buffer

	buf.WriteString("// Code generated by cmd/codegen. DO NOT EDIT.\n\n")
	buf.WriteString("package truenas\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n")
	if needsJSON {
		buf.WriteString("\t\"encoding/json\"\n")
	}
	buf.WriteString("\n\t\"github.com/gringolito/terraform-provider-truenas/internal/client\"\n")
	buf.WriteString(")\n\n")

	// Emit structs, deduped by name.
	emitted := make(map[string]bool)

	if needsZFSProperty {
		emitZFSPropertyStruct(&buf)
	}

	// Return structs first (shared across methods).
	for _, s := range sigs {
		if s.returnStructName != "" && !emitted[s.returnStructName] && s.returnSchema != nil {
			emitStruct(&buf, s.returnStructName, *s.returnSchema, false, false)
			emitted[s.returnStructName] = true
		}
	}
	// Args structs (isArgs=true so optional bools become *bool).
	for _, s := range sigs {
		if s.argsStructName != "" && !emitted[s.argsStructName] && s.argsSchema != nil {
			emitStruct(&buf, s.argsStructName, *s.argsSchema, s.isUpdateArgs, true)
			emitted[s.argsStructName] = true
		}
	}

	for _, s := range sigs {
		emitFunc(&buf, s)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("formatting source for %s: %w\n\nRaw output:\n%s", ns, err, buf.String())
	}
	return formatted, nil
}

func structNeedsJSON(s Schema) bool {
	s = resolveAllOf(s)
	for _, field := range s.Properties {
		if strings.Contains(schemaToGoType(field), "json.RawMessage") {
			return true
		}
	}
	return false
}

// emitStruct writes a Go struct definition. updateArgs=true activates
// update-arg-specific transformations (slices become *[]T). isArgs=true
// additionally makes optional bool fields *bool so callers can explicitly
// send false (not just omit the field).
func emitStruct(buf *bytes.Buffer, name string, s Schema, updateArgs, isArgs bool) {
	s = resolveAllOf(s)

	buf.WriteString("type " + name + " struct {\n")
	for _, fieldName := range fieldOrder(s) {
		fieldSchema, ok := s.Properties[fieldName]
		if !ok {
			continue
		}
		goName := snakeToPascal(fieldName)
		goType := schemaToGoType(fieldSchema)
		tag := fieldName
		required := isFieldRequired(s, fieldName, fieldSchema)

		switch {
		case isArgs && !required && goType == "bool":
			// *bool with omitempty: nil → field omitted; non-nil → value sent.
			// Applies to both create and update args so callers can send explicit false.
			goType = "*bool"
			tag += ",omitempty"
		case updateArgs && strings.HasPrefix(goType, "[]"):
			// Use *[]T with omitempty: nil → field omitted from JSON entirely;
			// &[]T{} → field sent as []; &[]T{v} → field sent as [v].
			// This allows callers to omit fields they do not manage (e.g. groups
			// on truenas_user, which is owned by truenas_user_group_membership).
			goType = "*" + goType
			tag += ",omitempty"
		case !required:
			tag += ",omitempty"
		}

		fmt.Fprintf(buf, "\t%s %s `json:\"%s\"`\n", goName, goType, tag)
	}
	buf.WriteString("}\n\n")
}

// emitZFSPropertyStruct writes the shared ZFSProperty struct definition once
// per file, used for every field matching the recurring ZFS property shape.
func emitZFSPropertyStruct(buf *bytes.Buffer) {
	buf.WriteString("// ZFSProperty is the recurring {value, rawvalue, parsed, source, source_info}\n")
	buf.WriteString("// shape TrueNAS uses for ZFS dataset properties.\n")
	buf.WriteString("type ZFSProperty struct {\n")
	buf.WriteString("\tValue      *string         `json:\"value,omitempty\"`\n")
	buf.WriteString("\tRawValue   *string         `json:\"rawvalue,omitempty\"`\n")
	buf.WriteString("\tParsed     json.RawMessage `json:\"parsed,omitempty\"`\n")
	buf.WriteString("\tSource     *string         `json:\"source,omitempty\"`\n")
	buf.WriteString("\tSourceInfo json.RawMessage `json:\"source_info,omitempty\"`\n")
	buf.WriteString("}\n\n")
}

// zeroValue returns the Go zero-value literal for the given type expression.
// Pointer, slice, and interface types use nil; value types use their typed zero.
func zeroValue(t string) string {
	switch t {
	case "int64":
		return "0"
	case "float64":
		return "0"
	case "bool":
		return "false"
	case "string":
		return `""`
	default:
		return "nil" // pointer, slice, json.RawMessage, etc.
	}
}

func emitFunc(buf *bytes.Buffer, s methodSig) {
	buf.WriteString("func " + s.funcName + "(ctx context.Context, c client.Caller")
	if s.hasIDArg {
		fmt.Fprintf(buf, ", id %s", s.idType)
	}
	if s.hasQueryFilters {
		buf.WriteString(", filters ...QueryFilter")
	}
	if s.argsStructName != "" {
		buf.WriteString(", args " + s.argsStructName)
	}

	if s.returnType != "" {
		buf.WriteString(") (" + s.returnType + ", error) {\n")
	} else {
		buf.WriteString(") error {\n")
	}

	params := buildParams(s)
	if s.returnType == "" {
		fmt.Fprintf(buf, "\t_, err := c.Call(ctx, %q, %s)\n", s.method, params)
		buf.WriteString("\treturn err\n}\n\n")
		return
	}

	zero := zeroValue(s.returnType)
	fmt.Fprintf(buf, "\traw, err := c.Call(ctx, %q, %s)\n", s.method, params)
	fmt.Fprintf(buf, "\tif err != nil {\n\t\treturn %s, err\n\t}\n", zero)

	switch {
	case strings.HasPrefix(s.returnType, "*"):
		structName := strings.TrimPrefix(s.returnType, "*")
		fmt.Fprintf(buf, "\tvar result %s\n", structName)
		fmt.Fprintf(buf, "\tif err := json.Unmarshal(raw, &result); err != nil {\n\t\treturn %s, err\n\t}\n", zero)
		buf.WriteString("\treturn &result, nil\n}\n\n")
	default:
		fmt.Fprintf(buf, "\tvar result %s\n", s.returnType)
		fmt.Fprintf(buf, "\tif err := json.Unmarshal(raw, &result); err != nil {\n\t\treturn %s, err\n\t}\n", zero)
		buf.WriteString("\treturn result, nil\n}\n\n")
	}
}

func buildParams(s methodSig) string {
	switch {
	case s.hasQueryFilters:
		return "[]any{filtersToRaw(filters)}"
	case s.hasIDArg && s.argsStructName != "":
		return "[]any{id, args}"
	case s.hasIDArg:
		return "[]any{id}"
	case s.argsStructName != "":
		return "[]any{args}"
	default:
		return "[]any{}"
	}
}
