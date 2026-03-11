package types

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/stdlib"
)

type Time struct {
	*proto
}

var TimeType = Time{newProto("Time", "Object", ClassRegistry)}

var TimeClass = NewClass("Time", "Object", TimeType, ClassRegistry)

func (t Time) Equals(t2 Type) bool { return t == t2 }
func (t Time) String() string      { return "TimeType" }
func (t Time) GoType() string      { return "time.Time" }
func (t Time) IsComposite() bool   { return false }

func (t Time) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.proto.MustResolve(m, false).ReturnType(t, b, args)
}

func (t Time) BlockArgTypes(m string, args []Type) []Type {
	return t.proto.MustResolve(m, false).blockArgs(t, args)
}

func (t Time) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.proto.MustResolve(m, false).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t Time) HasMethod(m string) bool {
	return t.proto.HasMethod(m, false)
}

func (t Time) Resolve(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
}

func (t Time) MustResolve(m string) MethodSpec {
	spec, ok := t.Resolve(m)
	if !ok {
		panic("Could not resolve method '" + m + "' on Time")
	}
	return spec
}

func (t Time) GetMethodSpec(m string) (MethodSpec, bool) {
	return t.Resolve(m)
}

func (t Time) Alias(existingMethod, newMethod string) {
	t.proto.MakeAlias(existingMethod, newMethod, false)
}

func (t Time) GenerateMethods(goType interface{}, exclude ...string) {
	t.proto.GenerateMethods(goType, exclude...)
}

func (t Time) Methods() map[string]MethodSpec {
	return t.proto.Methods()
}

func simpleTimeAccessor(goMethod string, returnType Type) MethodSpec {
	return MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return returnType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, goMethod),
			}
		},
	}
}

func init() {
	// Class method: Time.now → time.Now()
	TimeClass.Def("now", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return TimeType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("time", "Now"),
				Imports: []string{"time"},
			}
		},
	})

	// Time.new with arguments → time.Date(year, time.Month(month), day, hour, min, sec, 0, time.Local)
	TimeType.Def("initialize", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return TimeType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if len(args) == 0 {
				return Transform{
					Expr:    bst.Call("time", "Now"),
					Imports: []string{"time"},
				}
			}
			// Build time.Date call with provided args, defaulting missing ones
			dateArgs := make([]ast.Expr, 8)
			defaults := []ast.Expr{
				bst.Int(0),                                   // year (will be replaced)
				bst.Call("time", "Month", bst.Int(1)),        // month
				bst.Int(1),                                   // day
				bst.Int(0),                                   // hour
				bst.Int(0),                                   // min
				bst.Int(0),                                   // sec
				bst.Int(0),                                   // nsec
				bst.Dot(ast.NewIdent("time"), "Local"),       // loc
			}
			copy(dateArgs, defaults)
			for i, arg := range args {
				if i == 1 {
					// month needs time.Month() wrapper
					dateArgs[i] = bst.Call("time", "Month", arg.Expr)
				} else if i < 7 {
					dateArgs[i] = arg.Expr
				}
			}
			return Transform{
				Expr:    bst.Call("time", "Date", dateArgs...),
				Imports: []string{"time"},
			}
		},
	})

	// Accessors
	TimeType.Def("year", simpleTimeAccessor("Year", IntType))
	TimeType.Def("day", simpleTimeAccessor("Day", IntType))
	TimeType.Def("hour", simpleTimeAccessor("Hour", IntType))
	TimeType.Def("min", simpleTimeAccessor("Minute", IntType))
	TimeType.Def("sec", simpleTimeAccessor("Second", IntType))
	TimeType.Def("yday", simpleTimeAccessor("YearDay", IntType))

	// month → int(t.Month())
	TimeType.Def("month", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: &ast.CallExpr{
					Fun:  ast.NewIdent("int"),
					Args: []ast.Expr{bst.Call(rcvr.Expr, "Month")},
				},
			}
		},
	})

	// wday → int(t.Weekday())
	TimeType.Def("wday", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: &ast.CallExpr{
					Fun:  ast.NewIdent("int"),
					Args: []ast.Expr{bst.Call(rcvr.Expr, "Weekday")},
				},
			}
		},
	})

	// strftime → t.Format("go layout")
	// Format string is translated at compile time when it's a literal (no runtime dependency).
	// For dynamic format strings, falls back to stdlib.RubyStrftime at runtime.
	TimeType.Def("strftime", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if lit, ok := args[0].Expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				// Compile-time evaluation: translate Ruby format to Go layout directly
				rubyFmt := strings.Trim(lit.Value, "\"")
				goLayout := stdlib.RubyStrftime(rubyFmt)
				return Transform{
					Expr: bst.Call(rcvr.Expr, "Format", bst.String(goLayout)),
				}
			}
			// Dynamic format string: needs runtime translation
			return Transform{
				Expr:    bst.Call(rcvr.Expr, "Format", bst.Call("stdlib", "RubyStrftime", args[0].Expr)),
				Imports: []string{"time"},
			}
		},
	})

	// to_i → t.Unix()
	TimeType.Def("to_i", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: &ast.CallExpr{
					Fun:  ast.NewIdent("int"),
					Args: []ast.Expr{bst.Call(rcvr.Expr, "Unix")},
				},
			}
		},
	})

	// to_f → float64(t.UnixNano()) / 1e9
	TimeType.Def("to_f", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return FloatType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(
					&ast.CallExpr{
						Fun:  ast.NewIdent("float64"),
						Args: []ast.Expr{bst.Call(rcvr.Expr, "UnixNano")},
					},
					token.QUO,
					&ast.BasicLit{Kind: token.FLOAT, Value: "1e9"},
				),
			}
		},
	})

	// to_s → t.String()
	TimeType.Def("to_s", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "String"),
			}
		},
	})

	// utc → t.UTC()
	TimeType.Def("utc", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return TimeType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "UTC"),
			}
		},
	})

	// utc? → t.Location() == time.UTC
	TimeType.Def("utc?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(
					bst.Call(rcvr.Expr, "Location"),
					token.EQL,
					bst.Dot(ast.NewIdent("time"), "UTC"),
				),
				Imports: []string{"time"},
			}
		},
	})

	// Comparison operators
	TimeType.Def("<", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: bst.Call(rcvr.Expr, "Before", args[0].Expr)}
		},
	})
	TimeType.Def(">", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: bst.Call(rcvr.Expr, "After", args[0].Expr)}
		},
	})
	TimeType.Def("==", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: bst.Call(rcvr.Expr, "Equal", args[0].Expr)}
		},
	})

	// + seconds → t.Add(time.Duration(n) * time.Second)
	TimeType.Def("+", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return TimeType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			dur := bst.Binary(
				&ast.CallExpr{
					Fun:  bst.Dot(ast.NewIdent("time"), "Duration"),
					Args: []ast.Expr{args[0].Expr},
				},
				token.MUL,
				bst.Dot(ast.NewIdent("time"), "Second"),
			)
			return Transform{
				Expr:    bst.Call(rcvr.Expr, "Add", dur),
				Imports: []string{"time"},
			}
		},
	})

	// - time → t.Sub(t2).Seconds() (returns float)
	// - seconds → t.Add(-duration)
	TimeType.Def("-", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			if args[0].Equals(TimeType) {
				return FloatType, nil
			}
			return TimeType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if args[0].Type.Equals(TimeType) {
				return Transform{
					Expr: bst.Call(bst.Call(rcvr.Expr, "Sub", args[0].Expr), "Seconds"),
				}
			}
			dur := bst.Binary(
				&ast.CallExpr{
					Fun:  bst.Dot(ast.NewIdent("time"), "Duration"),
					Args: []ast.Expr{&ast.UnaryExpr{Op: token.SUB, X: args[0].Expr}},
				},
				token.MUL,
				bst.Dot(ast.NewIdent("time"), "Second"),
			)
			return Transform{
				Expr:    bst.Call(rcvr.Expr, "Add", dur),
				Imports: []string{"time"},
			}
		},
	})
}
