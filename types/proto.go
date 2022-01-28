package types

import (
	"fmt"
	"go/ast"
	"reflect"
	"regexp"
	"strings"

	"github.com/redneckbeard/thanos/bst"
)

// Holds data for methods implemented "natively" in Thanos, initially targeting
// built-in methods on native Ruby datastructures and types. This of course
// means all the stuff that comes in Enumerable, which involves doing things
// with blocks that in theory _could_ be done with anonymous functions in Go
// but translates most efficiently and idiomatically to simple for loops.
type MethodSpec struct {
	// When inferring the return types for methods that take a block, we must
	// consider the return type of the block (since blocks cannot explicitly
	// return, this means resolving the type of the last expression in the block
	// -- no plans to support `break` or `next` yet as I'm honestly unsure if
	// I've ever seen them in the wild) and the receiver, since in a composite
	// type the inner type will determine or factor into the return type. Args
	// are often not given with Enumerable but when they are that can determine
	// the return type.
	ReturnType ReturnTypeFunc
	// For any methods where we're creating a MethodSpec, we don't have the
	// implementation to examine in Ruby and see what types of args get passed to
	// `block.call`. Therefore we must first provide a way to compute what the
	// types of the args will be so that we can use them to seed inference of the
	// return type of the block.
	blockArgs    func(Type, []Type) []Type
	TransformAST TransformFunc
}

type ReturnTypeFunc func(receiverType Type, blockReturnType Type, args []Type) (Type, error)
type TransformFunc func(TypeExpr, []TypeExpr, *Block, bst.IdentTracker) Transform

type Transform struct {
	Stmts   []ast.Stmt
	Expr    ast.Expr
	Imports []string
}

type proto struct {
	class, parent  string
	methods        map[string]MethodSpec
	bracketAliases map[Type]string
	registry       *classRegistry
	initialized    bool
	_type          Type
}

func newProto(class, parent string, registry *classRegistry) *proto {
	p := &proto{
		class:          class,
		parent:         parent,
		methods:        make(map[string]MethodSpec),
		bracketAliases: make(map[Type]string),
		registry:       registry,
	}
	return p
}

func (p *proto) ClassName() string { return p.class }
func (p *proto) UserDefined() bool { return false }

func (p *proto) Methods() map[string]MethodSpec { return p.methods }

func (p *proto) SelfDef(m string, spec MethodSpec) {
	go func() {
		for {
			class, err := p.registry.Get(p.class)
			if err == nil {
				class.Def(m, spec)
				break
			}
		}
	}()
}

func (p *proto) Resolve(m string, classMethod bool) (MethodSpec, bool) {
	method, has := p.methods[m]
	if !has {
		className := p.class
		if className == "" {
			className = "Object"
		}
		if p.registry == nil {
			panic("tried to Resolve method but registry not set")
		}
		class, err := p.registry.Get(className)
		if err != nil {
			panic(err)
		}
		for class.parent != nil {
			class = class.parent
			if classMethod {
				method, has = class.proto.methods[m]
			} else {
				method, has = class.Instance.Methods()[m]
			}
			if has {
				return method, has
			}
		}
	}
	return method, has
}

func (p *proto) MustResolve(m string, classMethod bool) MethodSpec {
	method, has := p.Resolve(m, classMethod)
	methodType := "instance"
	if classMethod {
		methodType = "class"
	}
	if !has {
		panic(fmt.Errorf("Could not resolve %s method '%s' on class '%s'", methodType, m, p.class))
	}
	return method
}

func (p *proto) HasMethod(m string, classMethod bool) bool {
	_, has := p.Resolve(m, classMethod)
	return has
}

func (p *proto) Def(m string, spec MethodSpec) {
	p.methods[m] = spec
}

func (p *proto) Alias(existingMethod, newMethod string) {
	panic("client types must call `MakeAlias`")
}

func (p *proto) MakeAlias(existingMethod, newMethod string, classMethod bool) {
	p.methods[newMethod] = p.methods[existingMethod]
}

func (p *proto) IsMultiple() bool { return false }

func (p *proto) GenerateMethods(iface interface{}, exclusions ...string) {
	t := reflect.TypeOf(iface)

	// generics aren't yet something that can be introspected with reflect, so we
	// make the probably bad assumption here that the type parameter supplied in
	// the interface{} value above is the only type parameter this type has,
	// allowing us to use it as a stand-in for a true type parameter provided by
	// the reflect package.
	name, typeParam := typeName(t)

	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		var excluded bool
		for _, exclusion := range exclusions {
			if m.Name == exclusion {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}
		mt := m.Type
		v := reflect.Indirect(reflect.ValueOf(iface)).Type()
		methodName := ToSnakeCase(m.Name)
		if strings.HasSuffix(methodName, "_q") {
			methodName = strings.TrimSuffix(methodName, "_q") + "?"
		}
		p.Def(methodName, MethodSpec{
			ReturnType: func(mt reflect.Type, receiverName, typeParam string) ReturnTypeFunc {
				return func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
					var retType Type
					if mt.NumOut() > 1 {
						multiple := Multiple{}
						for j := 0; j < mt.NumOut(); j++ {
							rt := mt.Out(j)
							if tt := getGenericType(rt, receiverType, typeParam); tt != nil {
								multiple = append(multiple, tt)
							} else {
								multiple = append(multiple, reflectTypeToThanosType(rt))
							}
						}
						retType = multiple
					} else {
						rt := mt.Out(0)
						if tt := getGenericType(rt, receiverType, typeParam); tt != nil {
							retType = tt
						} else {
							retType = reflectTypeToThanosType(rt)
						}
					}
					return retType, nil
				}
			}(mt, name, typeParam),
			TransformAST: func(name, path string) TransformFunc {
				return func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					argExprs := []ast.Expr{}
					for _, a := range args {
						argExprs = append(argExprs, a.Expr)
					}
					return Transform{
						Expr:    bst.Call(rcvr.Expr, name, argExprs...),
						Imports: []string{path},
					}
				}
			}(m.Name, v.PkgPath()),
		})
	}
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func reflectTypeToThanosType(t reflect.Type) Type {
	switch t.Kind() {
	case reflect.Array, reflect.Slice:
		return NewArray(reflectTypeToThanosType(t.Elem()))
	case reflect.Map:
		return NewHash(reflectTypeToThanosType(t.Key()), reflectTypeToThanosType(t.Elem()))
	default:
		if tt, exists := goTypeMap[t.Kind()]; exists {
			return tt
		} else {
			return nil
		}
	}
}
