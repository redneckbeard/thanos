//go:generate stringer -type Simple

// package types provides an interface and many implementations of that
// interface as an abstraction, however leaky, of the union of the Go type
// system and the Ruby Object system.
package types

import (
	"go/ast"
	"reflect"
	"regexp"
	"strings"

	"github.com/redneckbeard/thanos/bst"
)

type Type interface {
	BlockArgTypes(string, []Type) []Type
	ClassName() string
	Equals(Type) bool
	GoType() string
	HasMethod(string) bool
	IsComposite() bool
	IsMultiple() bool
	MethodReturnType(string, Type, []Type) (Type, error)
	GetMethodSpec(string) (MethodSpec, bool)
	String() string
	TransformAST(string, ast.Expr, []TypeExpr, *Block, bst.IdentTracker) Transform
}

type TypeExpr struct {
	Type Type
	Expr ast.Expr
}

type Simple int

func (t Simple) GoType() string      { return typeMap[t] }
func (t Simple) IsComposite() bool   { return false }
func (t Simple) IsMultiple() bool    { return false }
func (t Simple) ClassName() string   { return "" }
func (t Simple) Equals(t2 Type) bool { return t == t2 }

// lies but needed for now
func (t Simple) HasMethod(m string) bool                                      { return false }
func (t Simple) MethodReturnType(m string, b Type, args []Type) (Type, error) { return nil, nil }
func (t Simple) GetMethodSpec(m string) (MethodSpec, bool)                    { return MethodSpec{}, false }
func (t Simple) BlockArgTypes(m string, args []Type) []Type                   { return []Type{nil} }
func (t Simple) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return Transform{}
}

const (
	ConstType Simple = iota
	NilType
	FuncType
	AnyType
	ErrorType
)

var typeMap = map[Simple]string{
	ConstType: "const",
	NilType:   "nil",
	FuncType:  "func",
	AnyType:   "interface{}",
}

var goTypeMap = map[reflect.Kind]Type{
	reflect.Int:     IntType,
	reflect.Float64: FloatType,
	reflect.String:  StringType,
	reflect.Bool:    BoolType,
}

var goTypeMapByString = map[string]interface{}{
	"int":     IntType,
	"float64": FloatType,
	"string":  StringType,
	"bool":    BoolType,
}

var generic = regexp.MustCompile(`([A-Z]\w+)\[(\w+)\]`)

func typeName(t reflect.Type) (string, string) {
	submatches := generic.FindStringSubmatch(t.Name())
	if len(submatches) > 1 {
		return submatches[1], submatches[2]
	}
	return t.Name(), ""
}

func RegisterType(goValue interface{}, thanosTypeOrConstructor interface{}) {
	switch thanosTypeOrConstructor.(type) {
	case Type:
	case func(Type) Type:
	default:
		panic("Attempted to register a Go type with something other than a thanos type or type constructor")
	}
	container, _ := typeName(reflect.TypeOf(goValue))
	goTypeMapByString[container] = thanosTypeOrConstructor
}

func getGenericType(t reflect.Type, rcvr Type, typeParam string) Type {
	container, inner := typeName(t)
	if typeParam != "" && inner == typeParam {
		outer := goTypeMapByString[container].(func(Type) Type)
		return outer(rcvr.(CompositeType).Inner())
	}
	tt := goTypeMapByString[container]
	if tt != nil {
		return tt.(Type)
	}
	return nil
}

type Multiple []Type

func (t Multiple) GoType() string    { return "" }
func (t Multiple) IsComposite() bool { return false }
func (t Multiple) IsMultiple() bool  { return true }
func (t Multiple) ClassName() string { return "" }
func (t Multiple) String() string {
	segments := []string{}
	for _, s := range t {
		segments = append(segments, s.String())
	}
	return strings.Join(segments, ", ")
}
func (t Multiple) HasMethod(m string) bool                                      { return false }
func (t Multiple) MethodReturnType(m string, b Type, args []Type) (Type, error) { return nil, nil }
func (t Multiple) GetMethodSpec(m string) (MethodSpec, bool)                    { return MethodSpec{}, false }
func (t Multiple) BlockArgTypes(m string, args []Type) []Type                   { return []Type{nil} }
func (t Multiple) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return Transform{}
}
func (t Multiple) Imports(s string) []string { return []string{} }

func (mt Multiple) Equals(t Type) bool {
	mt2, ok := t.(Multiple)
	if !ok {
		return false
	}
	if len(mt) != len(mt2) {
		return false
	}
	for i, t := range mt {
		if t != mt2[i] {
			return false
		}
	}
	return true
}

type CompositeType interface {
	Type
	Outer() Type
	Inner() Type
}

// Named type registry — allows external packages to register types by name
// (e.g., "CSV::Row") so they can be looked up during facade processing and
// scope injection without creating import cycles.
var namedTypeRegistry = map[string]Type{}

// RegisterNamedType registers a Type under a qualified name (e.g., "CSV::Row").
func RegisterNamedType(name string, t Type) {
	namedTypeRegistry[name] = t
}

// LookupNamedType returns the Type registered under the given name.
func LookupNamedType(name string) (Type, bool) {
	t, ok := namedTypeRegistry[name]
	return t, ok
}

type Block struct {
	Args       []ast.Expr
	ArgTypes   []Type
	ReturnType Type
	Statements []ast.Stmt
}

// FuncLit builds an *ast.FuncLit from the block's compiled data.
func (b *Block) FuncLit(it bst.IdentTracker) *ast.FuncLit {
	params := []*ast.Field{}
	for i, arg := range b.Args {
		ident := arg.(*ast.Ident)
		var typeName string
		if i < len(b.ArgTypes) && b.ArgTypes[i] != nil {
			typeName = b.ArgTypes[i].GoType()
		}
		params = append(params, &ast.Field{
			Names: []*ast.Ident{ident},
			Type:  it.Get(typeName),
		})
	}
	var results []*ast.Field
	if b.ReturnType != nil && b.ReturnType != NilType {
		results = []*ast.Field{{Type: it.Get(b.ReturnType.GoType())}}
	}
	return &ast.FuncLit{
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: params},
			Results: &ast.FieldList{List: results},
		},
		Body: &ast.BlockStmt{List: b.Statements},
	}
}

// stripBlockReturn handles the final statement of a block body used in
// iterators like `each` where the block's return value is discarded. If the
// last statement is a ReturnStmt wrapping a bare ident (from a transform that
// prepended statements), the ReturnStmt is removed. Otherwise it's converted
// to an ExprStmt so side effects are preserved.
// BlankUnusedBlockArgs blanks unused block params to "_".
func BlankUnusedBlockArgs(blk *Block) { blankUnusedBlockArgs(blk) }

func blankUnusedBlockArgs(blk *Block) {
	for i, arg := range blk.Args {
		ident, ok := arg.(*ast.Ident)
		if !ok {
			continue
		}
		used := false
		ast.Inspect(&ast.BlockStmt{List: blk.Statements}, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && id.Name == ident.Name {
				used = true
				return false
			}
			return true
		})
		if !used {
			blk.Args[i] = ast.NewIdent("_")
		}
	}
}

// StripBlockReturn removes trailing return statements from block bodies.
func StripBlockReturn(blk *Block) { stripBlockReturn(blk) }

func stripBlockReturn(blk *Block) {
	if len(blk.Statements) == 0 {
		return
	}
	last := len(blk.Statements) - 1
	if ret, ok := blk.Statements[last].(*ast.ReturnStmt); ok {
		switch ret.Results[0].(type) {
		case *ast.Ident, *ast.IndexExpr:
			// Bare idents and map index expressions have no side effects — remove them
			blk.Statements = blk.Statements[:last]
		default:
			blk.Statements[last] = &ast.ExprStmt{X: ret.Results[0]}
		}
	}
}

// rewriteReturnsToAppend walks statements recursively, rewriting any
// ReturnStmt into `accum = append(accum, val)`. When a ReturnStmt is
// followed by a BranchStmt{CONTINUE} (from `next <value>`), the continue
// is kept so the loop iteration ends after the append.
func rewriteReturnsToAppend(stmts []ast.Stmt, accum *ast.Ident) []ast.Stmt {
	for i, s := range stmts {
		switch stmt := s.(type) {
		case *ast.ReturnStmt:
			stmts[i] = bst.Assign(accum, bst.Call(nil, "append", accum, stmt.Results[0]))
		case *ast.IfStmt:
			stmt.Body.List = rewriteReturnsToAppend(stmt.Body.List, accum)
			if stmt.Else != nil {
				if elseBlock, ok := stmt.Else.(*ast.BlockStmt); ok {
					elseBlock.List = rewriteReturnsToAppend(elseBlock.List, accum)
				}
			}
		}
	}
	return stmts
}
