package net_http

import (
	"go/ast"

	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/types"
)

var netHTTPImport = "github.com/redneckbeard/thanos/net_http"

func init() {
	// --- Convenience class methods ---

	// Net::HTTP.get(url) or Net::HTTP.get(host, path) -> string
	types.NetHTTPClass.Def("get", types.MethodSpec{
		ReturnType: func(r types.Type, b types.Type, args []types.Type) (types.Type, error) {
			return types.StringType, nil
		},
		TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
			if len(args) >= 2 {
				return types.Transform{
					Expr:    bst.Call("net_http", "GetHostPath", args[0].Expr, args[1].Expr),
					Imports: []string{netHTTPImport},
				}
			}
			return types.Transform{
				Expr:    bst.Call("net_http", "Get", args[0].Expr),
				Imports: []string{netHTTPImport},
			}
		},
	})

	// Net::HTTP.get_response(url) or Net::HTTP.get_response(host, path) -> Net::HTTPResponse
	types.NetHTTPClass.Def("get_response", types.MethodSpec{
		ReturnType: func(r types.Type, b types.Type, args []types.Type) (types.Type, error) {
			return responseType(), nil
		},
		TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
			if len(args) >= 2 {
				return types.Transform{
					Expr:    bst.Call("net_http", "GetResponseHostPath", args[0].Expr, args[1].Expr),
					Imports: []string{netHTTPImport},
				}
			}
			return types.Transform{
				Expr:    bst.Call("net_http", "GetResponse", args[0].Expr),
				Imports: []string{netHTTPImport},
			}
		},
	})

	// Net::HTTP.post(url, data) -> Net::HTTPResponse
	types.NetHTTPClass.Def("post", types.MethodSpec{
		ReturnType: func(r types.Type, b types.Type, args []types.Type) (types.Type, error) {
			return responseType(), nil
		},
		TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
			return types.Transform{
				Expr:    bst.Call("net_http", "Post", args[0].Expr, args[1].Expr),
				Imports: []string{netHTTPImport},
			}
		},
	})

	// Net::HTTP.post_form(url, params) -> Net::HTTPResponse
	types.NetHTTPClass.Def("post_form", types.MethodSpec{
		ReturnType: func(r types.Type, b types.Type, args []types.Type) (types.Type, error) {
			return responseType(), nil
		},
		TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
			return types.Transform{
				Expr:    bst.Call("net_http", "PostForm", args[0].Expr, bst.Call(args[1].Expr, "Data")),
				Imports: []string{netHTTPImport},
			}
		},
	})

	// --- start and new ---

	// Net::HTTP.start(host, port, use_ssl: bool) { |http| ... } -> nil (with block) or Client (without)
	types.NetHTTPClass.Def("start", types.MethodSpec{
		KwargsSpec: []types.KwargSpec{
			{Name: "use_ssl", Type: types.BoolType},
		},
		ReturnType: func(r types.Type, b types.Type, args []types.Type) (types.Type, error) {
			if b != nil {
				return types.NilType, nil
			}
			return clientType(), nil
		},
		TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
			newClientExpr := buildNewClientExpr(args)

			if blk != nil {
				types.StripBlockReturn(blk)
				types.BlankUnusedBlockArgs(blk)
				clientDecl := bst.Define(blk.Args[0], newClientExpr)
				stmts := []ast.Stmt{clientDecl}
				stmts = append(stmts, blk.Statements...)
				return types.Transform{
					Stmts:   stmts,
					Imports: []string{netHTTPImport},
				}
			}
			return types.Transform{
				Expr:    newClientExpr,
				Imports: []string{netHTTPImport},
			}
		},
	})
	types.NetHTTPClass.SetBlockArgs("start", func(r types.Type, args []types.Type) []types.Type {
		return []types.Type{clientType()}
	})

	// Net::HTTP.new(host, port, use_ssl: bool) -> Client
	types.NetHTTPClass.Def("new", types.MethodSpec{
		KwargsSpec: []types.KwargSpec{
			{Name: "use_ssl", Type: types.BoolType},
		},
		ReturnType: func(r types.Type, b types.Type, args []types.Type) (types.Type, error) {
			return clientType(), nil
		},
		TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
			return types.Transform{
				Expr:    buildNewClientExpr(args),
				Imports: []string{netHTTPImport},
			}
		},
	})

	// --- Request constructors ---
	// These are registered as class methods on sentinel classes (Get, Post, etc.)
	// that live inside the Net::HTTP namespace.
	for method, cls := range requestClasses() {
		registerRequestConstructor(cls, method)
	}
}

// buildNewClientExpr builds net_http.NewClient(host, port, useSSL) from args.
// args layout: [host, port (optional), use_ssl kwarg (optional)]
func buildNewClientExpr(args []types.TypeExpr) ast.Expr {
	host := args[0].Expr

	// Port: second positional arg, or 0 if not provided
	var port ast.Expr
	if len(args) >= 2 && args[1].Expr != nil {
		port = args[1].Expr
	} else {
		port = bst.Int(0)
	}

	// use_ssl: kwarg after positional args
	// With KwargsSpec [use_ssl], it's at index 2 (after host, port)
	// or index 1 (after host) if port was omitted... but KwargsSpec
	// always appends after positional args. With 1 positional, kwarg is at [1].
	// With 2 positional, kwarg is at [2].
	var useSSL ast.Expr
	// The kwarg is always the last arg in the spec
	kwargIdx := len(args) - 1
	if kwargIdx >= 0 && args[kwargIdx].Type == types.BoolType && args[kwargIdx].Expr != nil {
		useSSL = args[kwargIdx].Expr
	} else {
		useSSL = ast.NewIdent("false")
	}

	return bst.Call("net_http", "NewClient", host, port, useSSL)
}

func responseType() types.Type {
	t, _ := types.LookupNamedType("Net::HTTPResponse")
	return t
}

func clientType() types.Type {
	t, _ := types.LookupNamedType("Net::HTTPClient")
	return t
}

func requestType() types.Type {
	t, _ := types.LookupNamedType("Net::HTTPRequest")
	return t
}

// requestClasses returns the HTTP verb → class mapping for request constructors.
func requestClasses() map[string]*types.Class {
	verbs := map[string]string{
		"GET":    "Get",
		"POST":   "Post",
		"PUT":    "Put",
		"PATCH":  "Patch",
		"DELETE": "Delete",
		"HEAD":   "Head",
	}
	classes := map[string]*types.Class{}
	for method, name := range verbs {
		cls := types.NewClass(name, "Object", nil, types.ClassRegistry)
		classes[method] = cls
		types.RegisterNamedType("Net::HTTP::"+name, cls)
	}
	return classes
}

// registerRequestConstructor adds a `new` method to a request verb class
// that creates a net_http.NewRequest with the appropriate HTTP method.
func registerRequestConstructor(cls *types.Class, method string) {
	httpMethod := method // capture for closure
	cls.Def("new", types.MethodSpec{
		ReturnType: func(r types.Type, b types.Type, args []types.Type) (types.Type, error) {
			return requestType(), nil
		},
		TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
			return types.Transform{
				Expr:    bst.Call("net_http", "NewRequest", bst.String(httpMethod), args[0].Expr),
				Imports: []string{netHTTPImport},
			}
		},
	})
}
