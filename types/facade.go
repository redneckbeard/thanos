package types

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"strings"

	"github.com/redneckbeard/thanos/bst"
)

// FacadeConfig is the top-level config file structure.
// Keys are Ruby require names (e.g., "base64").
type FacadeConfig map[string]*LibraryFacade

// LibraryFacade describes how a Ruby library maps to Go.
type LibraryFacade struct {
	GoImport  string                    `json:"go_import"`
	GoImports []string                  `json:"go_imports"`
	Modules   map[string]*ModuleFacade `json:"modules"`
	Types     map[string]*TypeFacade   `json:"types"`
	Coverage  string                    `json:"coverage,omitempty"` // e.g. "6/10 methods" — empty means complete
}

// TypeFacade describes a Ruby type declaratively (e.g., CSV::Row).
// The facade system synthesizes a Type implementation from this spec.
type TypeFacade struct {
	GoType  string                       `json:"go_type"`
	Methods map[string]*TypeMethodFacade `json:"methods"`
}

// TypeMethodFacade describes a single method on a facade-defined type.
type TypeMethodFacade struct {
	Call      string `json:"call"`       // Go method name (e.g., "Get")
	Returns   string `json:"returns"`    // return type name (e.g., "string", "CSV::Row")
	ReturnsGo string `json:"returns_go"` // actual Go return type when it differs from thanos type (e.g., "map[string]string")
	Iterate   string `json:"iterate"`    // Go method returning slice to range over
	Yields    string `json:"yields"`     // type yielded to block in iteration
}

// allImports returns the combined list of imports from both fields.
func (lb *LibraryFacade) allImports() []string {
	var imports []string
	if lb.GoImport != "" {
		imports = append(imports, lb.GoImport)
	}
	imports = append(imports, lb.GoImports...)
	return imports
}

// ModuleFacade describes a Ruby module's methods.
type ModuleFacade struct {
	Methods map[string]*MethodFacade `json:"methods"`
}

// MethodFacade describes how a single Ruby method maps to Go.
type MethodFacade struct {
	// Call is a pipeline of Go functions. The Ruby args are passed to the first,
	// and each subsequent function receives the result of the previous.
	// e.g. ["base64.StdEncoding.EncodeToString"] or
	//      ["base64.StdEncoding.DecodeString", "string"]
	Call []string `json:"call"`
	// Args describes argument transforms. Each entry can have a "cast" field.
	Args []ArgFacade `json:"args"`
	// Returns is the thanos type name of the return value ("string", "int", "bool", "float", "nil").
	Returns string `json:"returns"`
	// IgnoreError means the first call returns (T, error) and we use val, _ := ...
	IgnoreError bool `json:"ignore_error"`
}

// ArgFacade describes how to transform a single argument.
type ArgFacade struct {
	Cast string `json:"cast"` // e.g. "[]byte" to wrap arg in []byte(arg)
}

// LoadFacades reads a facades JSON file and returns the config.
func LoadFacades(path string) (FacadeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config FacadeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid facades file %s: %w", path, err)
	}
	return config, nil
}

// FacadeNamespace describes a scoped module name (e.g., "Digest" -> ["SHA256", "MD5"]).
type FacadeNamespace struct {
	Name    string   // outer module name (e.g., "Digest")
	Members []string // inner class names (e.g., ["SHA256", "MD5"])
}

// RegisterFacade registers all modules and methods from a library facade
// into the ClassRegistry so the type system can resolve them. It returns a list
// of namespaces for any scoped module names (e.g., "Digest::SHA256").
func RegisterFacade(requireName string, lib *LibraryFacade) []FacadeNamespace {
	namespaces := map[string]*FacadeNamespace{}
	for moduleName, mod := range lib.Modules {
		// Handle scoped names like "Digest::SHA256"
		className := moduleName
		if parts := strings.Split(moduleName, "::"); len(parts) > 1 {
			outerName := parts[0]
			innerName := parts[len(parts)-1]
			className = innerName
			if ns, ok := namespaces[outerName]; ok {
				ns.Members = append(ns.Members, innerName)
			} else {
				namespaces[outerName] = &FacadeNamespace{
					Name:    outerName,
					Members: []string{innerName},
				}
			}
		}
		cls := NewClass(className, "", nil, ClassRegistry)
		for methodName, mb := range mod.Methods {
			spec := buildFacadeMethodSpec(lib.allImports(), mb)
			cls.Def(methodName, spec)
		}
	}

	// Process declarative type definitions
	typeNamespaces := registerFacadeTypes(lib)
	for _, ns := range typeNamespaces {
		if existing, ok := namespaces[ns.Name]; ok {
			existing.Members = append(existing.Members, ns.Members...)
		} else {
			nsCopy := ns
			namespaces[ns.Name] = &nsCopy
		}
	}

	var result []FacadeNamespace
	for _, ns := range namespaces {
		result = append(result, *ns)
	}
	return result
}

// resolveTypeName parses a type name string into a Type.
// Supports primitives ("string", "int"), arrays ("[]string"),
// hashes ("{string: string}"), and registered named types ("CSV::Row").
func resolveTypeName(name string) Type {
	switch name {
	case "string":
		return StringType
	case "int":
		return IntType
	case "float":
		return FloatType
	case "bool":
		return BoolType
	case "nil":
		return NilType
	case "":
		return AnyType
	}
	if strings.HasPrefix(name, "[]") {
		if inner := resolveTypeName(name[2:]); inner != nil {
			return NewArray(inner)
		}
	}
	if strings.HasPrefix(name, "{") && strings.HasSuffix(name, "}") {
		inner := name[1 : len(name)-1]
		if parts := strings.SplitN(inner, ": ", 2); len(parts) == 2 {
			k := resolveTypeName(strings.TrimSpace(parts[0]))
			v := resolveTypeName(strings.TrimSpace(parts[1]))
			if k != nil && v != nil {
				return NewHash(k, v)
			}
		}
	}
	if t, ok := LookupNamedType(name); ok {
		return t
	}
	return AnyType
}

// facadeReturnType resolves a type name string for facade method specs.
func facadeReturnType(name string) Type {
	return resolveTypeName(name)
}

// buildFacadeMethodSpec creates a MethodSpec from a MethodFacade config.
func buildFacadeMethodSpec(goImports []string, mb *MethodFacade) MethodSpec {
	retType := facadeReturnType(mb.Returns)
	pipeline := mb.Call
	ignoreError := mb.IgnoreError
	argFacades := mb.Args

	return MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return retType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			imports := append([]string{}, goImports...)

			if len(pipeline) == 0 {
				return Transform{Imports: imports}
			}

			// Transform arguments for the first call
			goArgs := make([]ast.Expr, len(args))
			for i, arg := range args {
				goArgs[i] = arg.Expr
				if i < len(argFacades) && argFacades[i].Cast != "" {
					goArgs[i] = &ast.CallExpr{
						Fun:  buildCastExpr(argFacades[i].Cast),
						Args: []ast.Expr{arg.Expr},
					}
				}
			}

			// Build first call
			firstCall := &ast.CallExpr{
				Fun:  buildCallExpr(pipeline[0], it),
				Args: goArgs,
			}

			var stmts []ast.Stmt
			var currentExpr ast.Expr = firstCall

			// If the first call returns an error, capture with val, _ :=
			if ignoreError {
				result := it.New("val")
				stmts = append(stmts, &ast.AssignStmt{
					Lhs: []ast.Expr{result, ast.NewIdent("_")},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{firstCall},
				})
				currentExpr = result
			}

			// Chain remaining pipeline steps: each wraps the previous result
			for _, step := range pipeline[1:] {
				currentExpr = &ast.CallExpr{
					Fun:  buildCallExpr(step, it),
					Args: []ast.Expr{currentExpr},
				}
			}

			return Transform{
				Stmts:   stmts,
				Expr:    currentExpr,
				Imports: imports,
			}
		},
	}
}

// buildCallExpr parses a dotted Go expression like "base64.StdEncoding.EncodeToString"
// into an ast.Expr chain of SelectorExprs.
func buildCallExpr(dotted string, it bst.IdentTracker) ast.Expr {
	parts := strings.Split(dotted, ".")
	if len(parts) == 1 {
		return it.Get(parts[0])
	}
	var expr ast.Expr = ast.NewIdent(parts[0])
	for _, part := range parts[1:] {
		expr = &ast.SelectorExpr{
			X:   expr,
			Sel: ast.NewIdent(part),
		}
	}
	return expr
}

// buildCastExpr creates an ast.Expr for a type cast like "[]byte".
func buildCastExpr(cast string) ast.Expr {
	if strings.HasPrefix(cast, "[]") {
		elemType := cast[2:]
		return &ast.ArrayType{
			Elt: ast.NewIdent(elemType),
		}
	}
	return ast.NewIdent(cast)
}

// --- facadeType: synthesized Type from declarative JSON spec ---

// facadeType implements the Type interface from a TypeFacade JSON spec.
type facadeType struct {
	name    string // qualified name, e.g. "CSV::Row"
	goType  string // Go type, e.g. "*csv.Row"
	methods map[string]*TypeMethodFacade
	imports []string
}

func (t *facadeType) Equals(t2 Type) bool    { return t == t2 }
func (t *facadeType) String() string         { return t.name }
func (t *facadeType) GoType() string         { return t.goType }
func (t *facadeType) IsComposite() bool      { return false }
func (t *facadeType) IsMultiple() bool       { return false }
func (t *facadeType) ClassName() string      { return t.name }

func (t *facadeType) HasMethod(m string) bool {
	_, ok := t.methods[m]
	return ok
}

func (t *facadeType) BlockArgTypes(m string, args []Type) []Type {
	if mb, ok := t.methods[m]; ok && mb.Yields != "" {
		return []Type{resolveTypeName(mb.Yields)}
	}
	return nil
}

func (t *facadeType) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	if mb, ok := t.methods[m]; ok {
		if mb.Iterate != "" {
			return NilType, nil
		}
		return resolveTypeName(mb.Returns), nil
	}
	return NilType, nil
}

func (t *facadeType) GetMethodSpec(m string) (MethodSpec, bool) {
	mb, ok := t.methods[m]
	if !ok {
		return MethodSpec{}, false
	}

	if mb.Iterate != "" {
		return t.buildIterateSpec(mb), true
	}
	return t.buildCallSpec(mb), true
}

func (t *facadeType) buildCallSpec(mb *TypeMethodFacade) MethodSpec {
	retType := resolveTypeName(mb.Returns)
	goMethod := mb.Call
	imports := t.imports
	returnsGo := mb.ReturnsGo
	needsBridge := returnsGo != "" && returnsGo != mb.Returns

	return MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return retType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			argExprs := make([]ast.Expr, len(args))
			for i, a := range args {
				argExprs[i] = a.Expr
			}
			var callExpr ast.Expr = bst.Call(rcvr.Expr, goMethod, argExprs...)
			resultImports := append([]string{}, imports...)

			if needsBridge {
				if bridgeExpr, bridgeImports := buildTypeBridge(callExpr, returnsGo, retType); bridgeExpr != nil {
					callExpr = bridgeExpr
					resultImports = append(resultImports, bridgeImports...)
				}
			}

			return Transform{
				Expr:    callExpr,
				Imports: resultImports,
			}
		},
	}
}

func (t *facadeType) buildIterateSpec(mb *TypeMethodFacade) MethodSpec {
	yieldsType := resolveTypeName(mb.Yields)
	goMethod := mb.Iterate
	imports := t.imports

	return MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{yieldsType}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NilType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			stripBlockReturn(blk)
			blankUnusedBlockArgs(blk)
			return Transform{
				Stmts: []ast.Stmt{
					&ast.RangeStmt{
						Key:   ast.NewIdent("_"),
						Value: blk.Args[0],
						Tok:   token.DEFINE,
						X:     bst.Call(rcvr.Expr, goMethod),
						Body:  &ast.BlockStmt{List: blk.Statements},
					},
				},
				Imports: imports,
			}
		},
	}
}

func (t *facadeType) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	if spec, ok := t.GetMethodSpec(m); ok {
		return spec.TransformAST(TypeExpr{t, rcvr}, args, blk, it)
	}
	return Transform{}
}

// registerFacadeTypes processes the "types" section of a LibraryFacade,
// creates facadeType instances, registers them in the named type registry,
// and returns namespace info for scope injection.
func registerFacadeTypes(lib *LibraryFacade) []FacadeNamespace {
	if len(lib.Types) == 0 {
		return nil
	}

	namespaces := map[string]*FacadeNamespace{}
	imports := lib.allImports()

	for qualifiedName, tb := range lib.Types {
		bt := &facadeType{
			name:    qualifiedName,
			goType:  tb.GoType,
			methods: tb.Methods,
			imports: imports,
		}
		RegisterNamedType(qualifiedName, bt)

		// Track namespaces for :: resolution
		if parts := strings.Split(qualifiedName, "::"); len(parts) > 1 {
			outerName := parts[0]
			innerName := parts[len(parts)-1]
			// Also register under short name for ClassRegistry lookups
			RegisterNamedType(innerName, bt)
			if ns, ok := namespaces[outerName]; ok {
				ns.Members = append(ns.Members, qualifiedName)
			} else {
				namespaces[outerName] = &FacadeNamespace{
					Name:    outerName,
					Members: []string{qualifiedName},
				}
			}
		}
	}

	var result []FacadeNamespace
	for _, ns := range namespaces {
		result = append(result, *ns)
	}
	return result
}

// buildTypeBridge generates wrapping code when a Go return type differs from
// the thanos type. For example, Go returns map[K]V but thanos expects
// *stdlib.OrderedMap[K,V] — emit stdlib.NewOrderedMapFromGoMap(expr).
// Returns nil if no bridge is needed or the pattern isn't recognized.
func buildTypeBridge(expr ast.Expr, goRetType string, thanosType Type) (ast.Expr, []string) {
	if _, isHash := thanosType.(Hash); isHash && strings.HasPrefix(goRetType, "map[") {
		return bst.Call("stdlib", "NewOrderedMapFromGoMap", expr),
			[]string{"github.com/redneckbeard/thanos/stdlib"}
	}
	return nil, nil
}
