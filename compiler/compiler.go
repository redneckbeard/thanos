package compiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"math"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/parser"
	"github.com/redneckbeard/thanos/types"
)

const (
	InFuncDeclaration   parser.State = "InFuncDeclaration"
	InMethodDeclaration parser.State = "InMethodDeclaration"
	InReturnStatement   parser.State = "InReturnStatement"
	InCondAssignment    parser.State = "InCondAssignment"
	InBlockBody         parser.State = "InBlockBody"
)

var globalIdents = make(bst.IdentTracker)

type GoProgram struct {
	*parser.FSM
	Imports      map[string]bool
	CurrentLhs   []parser.Node
	BlockStack   []*ast.BlockStmt
	GlobalVars   []*ast.ValueSpec
	Constants    []*ast.ValueSpec
	TrackerStack []bst.IdentTracker
	it           bst.IdentTracker
	currentRcvr  *ast.Ident
}

func Compile(p *parser.Program) (string, error) {
	g := &GoProgram{FSM: &parser.FSM{}, Imports: make(map[string]bool)}
	g.pushTracker()

	f := &ast.File{
		Name: ast.NewIdent("main"),
	}

	decls := []ast.Decl{}

	for _, o := range p.Objects {
		if m, ok := o.(*parser.Method); ok {
			decls = append(decls, g.CompileFunc(m, nil))
		}
	}

	for _, class := range p.Classes {
		decls = append(decls, g.CompileClass(class)...)
	}

	mainFunc := &ast.FuncDecl{
		Name: ast.NewIdent("main"),
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
		},
	}

	g.newBlockStmt()
	g.pushTracker()
	for _, stmt := range p.Statements {
		g.CompileStmt(stmt)
	}
	g.popTracker()
	mainFunc.Body = g.currentBlockStmt()
	g.popBlockStmt()

	decls = append(decls, mainFunc)

	importPaths := []string{}

	for imp, _ := range g.Imports {
		importPaths = append(importPaths, imp)
	}

	sort.Strings(importPaths)

	importSpecs := []ast.Spec{}

	for _, path := range importPaths {
		importSpecs = append(importSpecs, &ast.ImportSpec{
			Path: bst.String(path),
		})
	}

	topDecls := []ast.Decl{}

	if len(importSpecs) > 0 {
		topDecls = append(topDecls, &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: importSpecs,
		})
	}

	for _, spec := range g.Constants {
		topDecls = append(topDecls, &ast.GenDecl{
			Tok:   token.CONST,
			Specs: []ast.Spec{spec},
		})
	}

	for _, spec := range g.GlobalVars {
		topDecls = append(topDecls, &ast.GenDecl{
			Tok:   token.VAR,
			Specs: []ast.Spec{spec},
		})
	}

	f.Decls = append(topDecls, decls...)

	var in, out bytes.Buffer
	err := format.Node(&in, token.NewFileSet(), f)
	if err != nil {
		return "", fmt.Errorf("Error converting AST to []byte: %s", err.Error())
	}

	cmd := exec.Command("goimports")
	cmd.Stdin = &in
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error running gofmt: %s", err.Error())
	}

	return out.String(), nil
}

func (g *GoProgram) CompileFunc(m *parser.Method, c *parser.Class) *ast.FuncDecl {
	if c == nil {
		g.PushState(InFuncDeclaration)
	} else {
		g.currentRcvr = g.it.Get(strings.ToLower(c.Name()[:1]))
		g.PushState(InMethodDeclaration)
	}
	g.ScopeChain = m.Scope
	g.pushTracker()
	defer func() {
		g.PopState()
		g.PopScope()
		g.popTracker()
		g.currentRcvr = nil
	}()
	params := g.GetFuncParams(m)

	fields := []*ast.Field{}
	if m.ReturnType().IsMultiple() {
		multiple := m.ReturnType().(types.Multiple)
		for _, t := range multiple {
			fields = append(fields, g.retTypeField(t))
		}
	} else {
		fields = append(fields, g.retTypeField(m.ReturnType()))
	}

	signature := &ast.FuncType{
		Params: &ast.FieldList{
			List: params,
		},
		Results: &ast.FieldList{
			List: fields,
		},
	}

	decl := &ast.FuncDecl{
		Type: signature,
		Body: g.CompileBlockStmt(m.Body.Statements),
		Name: g.it.Get(m.GoName()),
	}

	if c != nil {
		className := g.it.Get(c.Name())
		decl.Recv = &ast.FieldList{
			List: []*ast.Field{
				&ast.Field{
					Names: []*ast.Ident{g.currentRcvr},
					Type: &ast.StarExpr{
						X: className,
					},
				},
			},
		}
	}

	return decl
}

func (g *GoProgram) GetFuncParams(m *parser.Method) []*ast.Field {
	params := []*ast.Field{}
	for _, p := range m.Params {
		var (
			lastParam    *ast.Field
			lastSeenType string
		)
		if len(params) > 0 {
			lastParam = params[len(params)-1]
			lastSeenType = lastParam.Type.(*ast.Ident).Name
		}
		if lastParam != nil && lastSeenType == p.Type().GoType() {
			lastParam.Names = append(lastParam.Names, g.it.Get(p.Name))
		} else {
			params = append(params, &ast.Field{
				Names: []*ast.Ident{g.it.Get(p.Name)},
				Type:  g.it.Get(p.Type().GoType()),
			})
		}
	}
	return params
}

func (g *GoProgram) CompileClass(c *parser.Class) []ast.Decl {
	className := globalIdents.Get(c.Name())
	decls := []ast.Decl{}

	structFields := []*ast.Field{}
	for _, t := range c.IVars(nil) {
		name := t.Name
		if t.Readable || t.Writeable {
			name = strings.Title(name)
		}
		structFields = append(structFields, &ast.Field{
			Names: []*ast.Ident{g.it.Get(name)},
			Type:  g.it.Get(t.Type().GoType()),
		})
	}
	for _, constant := range c.Constants {
		g.addConstant(g.it.Get(c.Name()+constant.Name), g.CompileExpr(constant.Val))
	}
	decls = append(decls, &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: className,
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: structFields,
					},
				},
			},
		},
	})
	// hand roll a constructor
	params := []*ast.Field{}
	results := []*ast.Field{
		&ast.Field{
			Type: &ast.StarExpr{
				X: className,
			},
		},
	}

	setStructFields := []ast.Expr{}

	g.newBlockStmt()

	var (
		initialize *parser.Method
		cls        = c
	)

	for cls != nil && initialize == nil {
		initialize = cls.MethodSet.Methods["initialize"]
		if initialize == nil {
			cls = cls.Parent()
		}
	}

	if initialize != nil {
		// Currently naive but feeling lazy. What we really want here is the last
		// non-operator assignment to every instance var to be converted to a
		// struct initialization k/v pair, every preceding assignment to them to be
		// locals, and every one after to mutate the field on the already-created
		// struct.
		params = g.GetFuncParams(initialize)
		for _, stmt := range initialize.Body.Statements {
			if assign, ok := stmt.(*parser.AssignmentNode); ok {
				if ivar, isIvar := assign.Left[0].(*parser.IVarNode); isIvar {
					name := ivar.NormalizedVal()
					if ivar.IVar().Readable || ivar.IVar().Writeable {
						name = strings.Title(name)
					}
					setStructFields = append(setStructFields, &ast.KeyValueExpr{
						Key:   g.it.Get(name),
						Value: g.CompileExpr(assign.Right),
					})
				}
			}
		}
	}

	signature := &ast.FuncType{
		Params: &ast.FieldList{
			List: params,
		},
		Results: &ast.FieldList{
			List: results,
		},
	}

	g.appendToCurrentBlock(&ast.ReturnStmt{
		Results: []ast.Expr{
			&ast.UnaryExpr{
				Op: token.AND,
				X: &ast.CompositeLit{
					Type: className,
					Elts: setStructFields,
				},
			},
		},
	})

	constructor := &ast.FuncDecl{
		Name: g.it.Get(fmt.Sprintf("New%s", c.Name())),
		Type: signature,
		Body: g.currentBlockStmt(),
	}

	decls = append(decls, constructor)

	g.popBlockStmt()

	for _, m := range c.Methods(nil) {
		if m.Name == "initialize" {
			continue
		}
		decls = append(decls, g.CompileFunc(m, c))
	}

	return decls
}

// A Ruby expression will often translate into multiple Go statements, and so
// we need a way to prepend statements prior to where an expression gets
// translated if required. To achieve this, we maintain a stack of
// *ast.BlockStmt that is pushed to and popped from as we work our way down the
// tree. The top of this stack is available for method translating other nodes
// to append to. Because they can append before they complete, they can get
// preceding variable declarations, loops, etc. in place before the expression
// or statement at hand is added.
func (g *GoProgram) newBlockStmt() *ast.BlockStmt {
	blockStmt := &ast.BlockStmt{}
	g.BlockStack = append(g.BlockStack, blockStmt)
	return blockStmt
}

func (g *GoProgram) popBlockStmt() {
	g.BlockStack = g.BlockStack[:len(g.BlockStack)-1]
}

func (g *GoProgram) currentBlockStmt() *ast.BlockStmt {
	return g.BlockStack[len(g.BlockStack)-1]
}

func (g *GoProgram) pushTracker() {
	g.it = make(bst.IdentTracker)
	g.TrackerStack = append(g.TrackerStack, g.it)
}

func (g *GoProgram) popTracker() {
	if len(g.TrackerStack) > 0 {
		g.TrackerStack = g.TrackerStack[:len(g.TrackerStack)-1]
		if len(g.TrackerStack) > 0 {
			g.it = g.TrackerStack[len(g.TrackerStack)-1]
		}
	}
}

func (g *GoProgram) appendToCurrentBlock(stmts ...ast.Stmt) {
	currentBlock := g.currentBlockStmt()
	currentBlock.List = append(currentBlock.List, stmts...)
}

func (g *GoProgram) CompileBlockStmt(node parser.Node) *ast.BlockStmt {
	blockStmt := g.newBlockStmt()
	defer g.popBlockStmt()
	switch n := node.(type) {
	case *parser.Condition:
		if n.False != nil {
			g.CompileStmt(n)
			return blockStmt
		} else {
			return g.CompileBlockStmt(n.True)
		}
	case parser.Statements:
		last := len(n) - 1
		for i, s := range n {
			if i == last {
				if g.CurrentState() == InReturnStatement && s.Type() != types.NilType {
					t := s.Type()
					s = &parser.ReturnNode{Val: parser.ArgsNode{s}}
					s.SetType(t)
				} else if g.CurrentState() == InCondAssignment {
					s = &parser.AssignmentNode{
						Left:         g.CurrentLhs,
						Right:        s,
						Reassignment: true,
					}
				}
			}
			g.CompileStmt(s)
		}
		return blockStmt
	default:
		return &ast.BlockStmt{}
	}
}

// Statement translation methods never return AST nodes. Instead, they always
// append to the current block statement.
func (g *GoProgram) CompileStmt(node parser.Node) {
	switch n := node.(type) {
	case *parser.AssignmentNode:
		if len(n.Left) == 1 {
			if constant, ok := n.Left[0].(*parser.ConstantNode); ok {
				g.addConstant(g.it.Get(constant.Namespace+constant.Val), g.CompileExpr(n.Right))
				return
			}
		}
		g.CompileAssignmentNode(n)
	case *parser.ReturnNode:
		if !n.Type().IsMultiple() {
			switch stmt := n.Val[0].(type) {
			case *parser.Condition, *parser.CaseNode:
				g.PushState(InReturnStatement)
				defer g.PopState()
				g.CompileStmt(stmt)
			case *parser.MethodCall:
				if stmt.RequiresTransform() {
					g.PushState(InReturnStatement)
					defer g.PopState()
					g.CompileStmt(stmt)
				}
			default:
				g.appendToCurrentBlock(&ast.ReturnStmt{
					Results: g.mapToExprs(n.Val),
				})
			}
		} else {
			g.appendToCurrentBlock(&ast.ReturnStmt{
				Results: g.mapToExprs(n.Val),
			})
		}
	case *parser.Condition:
		cond := g.CompileExpr(n.Condition)
		// Remove conditional entirely if boolean value of cond expr is known at compile time
		if justBool, ok := cond.(*ast.Ident); ok && (justBool.Name == "true" || justBool.Name == "false") {
			if justBool.Name == "true" {
				g.appendToCurrentBlock(g.CompileBlockStmt(n.True).List...)
			} else {
				g.appendToCurrentBlock(g.CompileBlockStmt(n.False).List...)
			}
		} else {
			stmt := &ast.IfStmt{
				Cond: cond,
				Body: g.CompileBlockStmt(n.True),
			}
			if n.False != nil {
				stmt.Else = g.CompileBlockStmt(n.False)
			}
			g.appendToCurrentBlock(stmt)
		}
	case *parser.CaseNode:
		stmt := &ast.SwitchStmt{}
		tag := g.CompileExpr(n.Value)
		if n.Value != nil && !n.RequiresExpansion {
			stmt.Tag = tag
		}
		g.newBlockStmt()
		for _, when := range n.Whens {
			list := []ast.Expr{}
			for _, arg := range when.Conditions {
				var expr ast.Expr
				switch a := arg.(type) {
				case *parser.RangeNode:
					upperTok := token.LSS
					if a.Inclusive {
						upperTok = token.LEQ
					}
					expr = bst.Binary(
						bst.Binary(tag, token.GEQ, g.CompileExpr(a.Lower)),
						token.LAND,
						bst.Binary(tag, upperTok, g.CompileExpr(a.Upper)),
					)
				default:
					expr = g.CompileExpr(arg)
					if n.RequiresExpansion && isSimple(expr) {
						expr = bst.Binary(tag, token.EQL, expr)
					}
				}
				list = append(list, expr)
			}
			if len(list) == 0 {
				list = nil
			}
			if len(list) > 1 && n.RequiresExpansion {
				conds := list[0]
				for _, cond := range list[1:] {
					conds = bst.Binary(conds, token.LOR, cond)
				}
				list = []ast.Expr{conds}
			}
			g.appendToCurrentBlock(&ast.CaseClause{
				List: list,
				Body: g.CompileBlockStmt(when.Statements).List,
			})
		}
		stmt.Body = g.currentBlockStmt()
		g.popBlockStmt()
		g.appendToCurrentBlock(stmt)
	case *parser.MethodCall:
		if n.RequiresTransform() {
			transform := g.TransformMethodCall(n)
			g.appendToCurrentBlock(transform.Stmts...)
			if g.CurrentState() == InReturnStatement && n.Type() != types.NilType {
				g.appendToCurrentBlock(&ast.ReturnStmt{
					Results: []ast.Expr{transform.Expr},
				})
				// A Transform may yield only statements, in which case Expr could be
				// nil.  It could also return an expression solely for the purposes of
				// chaining, which in this case we don't need the Expr because we already
				// are expecting a statement. Ignore both these cases.
			} else if _, ok := transform.Expr.(*ast.CallExpr); ok {
				stmt := &ast.ExprStmt{
					X: transform.Expr,
				}
				g.appendToCurrentBlock(stmt)
			}
		} else {
			g.appendToCurrentBlock(&ast.ExprStmt{
				X: g.CompileExpr(n),
			})
		}
	default:
		expr := g.CompileExpr(n)
		// A single ident being returned here means we've prepended statements and
		// a transform has supplied an ident for potential chained operations. If
		// we got here, we're not going to make further calls on this object, so
		// skip it.
		if _, ok := expr.(*ast.Ident); !ok {
			g.appendToCurrentBlock(&ast.ExprStmt{
				X: expr,
			})
		}
	}
}

func (g *GoProgram) CompileAssignmentNode(node *parser.AssignmentNode) {
	if cond, ok := node.Right.(*parser.Condition); ok {

		// Here we handle the impedance mismatch between Ruby conditional
		// expressions and Go conditional statments.

		// We declare the variable outside the if statement, getting its type from
		// the local scope
		specs := []ast.Spec{}

		for _, left := range node.Left {
			identNode, isIdent := left.(*parser.IdentNode)
			if isIdent {
				localName := identNode.Val
				t := g.ScopeChain.ResolveVar(localName)
				if t.Type() == nil {
					panic(fmt.Sprintf("Attempted to resolve '%s' in '%s' but got no type on %#v", localName, node, t))
				}
				name := g.it.New(localName)
				specs = append(specs, &ast.ValueSpec{
					Names: []*ast.Ident{name},
					Type:  g.it.Get(t.Type().GoType()),
				})
			} else {
				panic("attempted to assign to LHS type other than ident")
			}

		}
		decl := &ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok:   token.VAR,
				Specs: specs,
			},
		}
		g.appendToCurrentBlock(decl)
		// Now before translating the branches of the conditional, we transition
		// the compiler's state.  This way subsequent steps will know to use an
		// assignment rather than definition operator.
		g.PushState(InCondAssignment)
		g.CurrentLhs = node.Left
		defer func() {
			g.PopState()
			g.CurrentLhs = nil
		}()
		g.CompileStmt(cond)
		return
	}
	var assignFunc bst.AssignFunc
	if node.OpAssignment {
		infix := node.Right.(*parser.InfixExpressionNode)
		assignFunc = bst.OpAssign(infix.Operator)
		node = node.Clone()
		node.Right = infix.Right
	} else if node.Reassignment {
		assignFunc = bst.Assign
	} else {
		assignFunc = bst.Define
	}

	// rhs must go first here for reason of generation of local variable names
	// in transforms
	rhs := []ast.Expr{g.CompileExpr(node.Right)}
	lhs := g.mapToExprs(node.Left)
	tautological := true
	if len(lhs) != len(rhs) {
		tautological = false
	}
	for i, left := range lhs {
		if rh, ok := rhs[i].(*ast.Ident); ok {
			if lh, ok := left.(*ast.Ident); ok {
				if rh.Name != lh.Name {
					tautological = false
					break
				}
			} else {
				tautological = false
				break
			}
		} else {
			tautological = false
			break
		}
	}
	if !tautological {
		g.appendToCurrentBlock(assignFunc(lhs, rhs))
	}
}

// Expression translation methods _do_ return AST Nodes because of the
// specificity of where they have to be inserted. Any additional statements can
// be prepended before returning.
func (g *GoProgram) CompileExpr(node parser.Node) ast.Expr {
	switch n := node.(type) {
	case *parser.InfixExpressionNode:
		if types.Operators[n.Operator].Spec.TransformAST != nil || n.HasMethod() {
			return g.TransformInfixExpressionNode(n)
		} else {
			return g.CompileInfixExpressionNode(n)
		}
	case *parser.MethodCall:
		if n.RequiresTransform() {
			transform := g.TransformMethodCall(n)
			g.appendToCurrentBlock(transform.Stmts...)
			return transform.Expr
		} else if n.Getter {
			return bst.Dot(g.CompileExpr(n.Receiver), strings.Title(n.MethodName))
		}
		args := []ast.Expr{}
		if n.Method == nil {
			panic("Method not set on MethodCall " + n.String())
		}
		for i := 0; i < len(n.Method.Params); i++ {
			p, _ := n.Method.GetParam(i)
			switch p.Kind {
			case parser.Positional:
				args = append(args, g.CompileExpr(n.Args[i]))
			case parser.Named:
				if i >= len(n.Args) {
					args = append(args, g.CompileExpr(p.Default))
				} else if _, ok := n.Args[i].(*parser.KeyValuePair); ok {
					args = append(args, g.CompileExpr(p.Default))
				} else {
					args = append(args, g.CompileExpr(n.Args[i]))
				}
			case parser.Keyword:
				if arg, err := n.Args.FindByName(p.Name); err != nil {
					args = append(args, g.CompileExpr(p.Default))
				} else {
					args = append(args, g.CompileExpr(arg.(*parser.KeyValuePair).Value))
				}
			}
		}
		//TODO take into account private/protected
		return bst.Call(nil, strings.Title(n.MethodName), args...)
	case *parser.IdentNode:
		if n.MethodCall != nil {
			return g.CompileExpr(n.MethodCall)
		}
		return g.it.Get(n.Val)
	case *parser.IVarNode:
		ivar := n.NormalizedVal()
		if n.IVar().Readable {
			ivar = strings.Title(ivar)
		}
		return &ast.SelectorExpr{
			X:   g.currentRcvr,
			Sel: g.it.Get(ivar),
		}
	case *parser.BooleanNode:
		return g.it.Get(n.Val)
	case *parser.IntNode:
		return bst.Int(n.Val)
	case *parser.Float64Node:
		return &ast.BasicLit{
			Kind:  token.FLOAT,
			Value: n.Val,
		}
	case *parser.SymbolNode:
		return bst.String(n.Val[1:])
	case *parser.StringNode:
		return g.CompileStringNode(n)

	case *parser.ArrayNode:
		elements := []ast.Expr{}
		for _, arg := range n.Args {
			elements = append(elements, g.CompileExpr(arg))
		}
		return &ast.CompositeLit{
			Type: &ast.ArrayType{
				Elt: g.it.Get(n.Type().(types.Array).Element.GoType()),
			},
			Elts: elements,
		}
	case *parser.HashNode:
		hashType := n.Type().(types.Hash)
		elements := []ast.Expr{}
		for _, pair := range n.Pairs {
			var key ast.Expr
			if pair.Label != "" {
				key = bst.String(pair.Label)
			} else {
				key = g.CompileExpr(pair.Key)
			}
			elements = append(elements, &ast.KeyValueExpr{
				Key:   key,
				Value: g.CompileExpr(pair.Value),
			})
		}
		return &ast.CompositeLit{
			Type: &ast.MapType{
				Key:   g.it.Get(hashType.Key.GoType()),
				Value: g.it.Get(hashType.Value.GoType()),
			},
			Elts: elements,
		}
	case *parser.BracketAccessNode:
		rcvr := g.CompileExpr(n.Composite)
		if method := n.Composite.Type().SupportsBrackets(n.Args[0].Type()); method != "" {
			transform := g.getTransform(rcvr, n.Composite.Type(), method, n.Args, nil)
			g.appendToCurrentBlock(transform.Stmts...)
			return transform.Expr
		}
		if r, ok := n.Args[0].(*parser.RangeNode); ok {
			return g.CompileRangeIndexNode(rcvr, r)
		} else {
			return &ast.IndexExpr{
				X:     g.CompileExpr(n.Composite),
				Index: g.CompileExpr(n.Args[0]),
			}
		}
	case *parser.BracketAssignmentNode:
		return &ast.IndexExpr{
			X:     g.CompileExpr(n.Composite),
			Index: g.CompileExpr(n.Args[0]),
		}
	case *parser.SelfNode:
		return g.currentRcvr
	case *parser.ConstantNode:
		return g.it.Get(n.Namespace + n.Val)
	case *parser.ScopeAccessNode:
		return g.it.Get(n.ReceiverName() + n.Constant)
	default:
		return &ast.BadExpr{}
	}
}

func (g *GoProgram) CompileInfixExpressionNode(node *parser.InfixExpressionNode) ast.Expr {
	return &ast.BinaryExpr{
		X:  g.CompileExpr(node.Left),
		Op: types.Operators[node.Operator].GoToken,
		Y:  g.CompileExpr(node.Right),
	}
}

func (g *GoProgram) CompileRangeIndexNode(rcvr ast.Expr, r *parser.RangeNode) ast.Expr {
	bounds := map[int]ast.Expr{}

	for i, bound := range []parser.Node{r.Lower, r.Upper} {
		if bound != nil {
			switch b := bound.(type) {
			case *parser.IntNode:
				// if it's a literal, we can just set up the slice
				x, _ := strconv.Atoi(b.Val)
				if x < 0 {
					boundExpr := &ast.BinaryExpr{
						X:  bst.Call(nil, "len", rcvr),
						Op: token.SUB,
					}
					if r.Inclusive && i == 1 {
						x += 1
					}
					boundExpr.Y = bst.Int(int(math.Abs(float64(x))))
					bounds[i] = boundExpr
				} else {
					if r.Inclusive && i == 1 {
						b.Val = strconv.Itoa(x + 1)
					}
					bounds[i] = g.CompileExpr(b)
				}
			case *parser.IdentNode:
				/*
					This case is much worse than a literal. What we need to build is
					something like this:

					   var lower, upper int
					   if foo < 0 {
					     lower = len(x) + foo
					   } else {
					     lower = foo
					   }

					We could avoid doing this for cases when a variable for the slice
					value is defined and initialized with a literal inside the current
					block, but that would make this code even more complicated.
				*/
				var local *ast.Ident
				if i == 0 {
					local = g.it.New("lower")
				} else {
					local = g.it.New("upper")
				}
				g.appendToCurrentBlock(&ast.DeclStmt{
					Decl: &ast.GenDecl{
						Tok: token.VAR,
						Specs: []ast.Spec{&ast.ValueSpec{
							Names: []*ast.Ident{local},
							Type:  g.it.Get("int"),
						}},
					},
				})
				var rhs ast.Expr
				if r.Inclusive && i == 1 {
					rhs = bst.Binary(g.CompileExpr(b), token.ADD, bst.Int(1))
				} else {
					rhs = g.CompileExpr(b)
				}
				cond := &ast.IfStmt{
					Cond: &ast.BinaryExpr{
						X:  g.CompileExpr(b),
						Y:  bst.Int(0),
						Op: token.LSS,
					},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							bst.Assign(local, &ast.BinaryExpr{
								X:  bst.Call(nil, "len", rcvr),
								Op: token.ADD,
								Y:  rhs,
							}),
						},
					},
					Else: bst.Assign(local, rhs),
				}
				g.appendToCurrentBlock(cond)
				bounds[i] = local
			}
		}
	}

	sliceExpr := &ast.SliceExpr{X: rcvr}
	for k, v := range bounds {
		if k == 0 {
			sliceExpr.Low = v
		} else {
			sliceExpr.High = v
		}
	}

	return sliceExpr
}

func (g *GoProgram) TransformInfixExpressionNode(node *parser.InfixExpressionNode) ast.Expr {
	if node.HasMethod() {
		transform := g.getTransform(g.CompileExpr(node.Left), node.Left.Type(), node.Operator, parser.ArgsNode{node.Right}, nil)
		g.appendToCurrentBlock(transform.Stmts...)
		return transform.Expr
	}
	op := types.Operators[node.Operator]
	transform := op.Spec.TransformAST(
		types.TypeExpr{node.Left.Type(), g.CompileExpr(node.Left)},
		types.TypeExpr{node.Right.Type(), g.CompileExpr(node.Right)},
		op.GoToken,
	)
	return transform.Expr
}

func (g *GoProgram) CompileStringNode(node *parser.StringNode) ast.Expr {
	// We don't want to use bst.String here, because node.GoString() will already
	//correctly surround the string
	str := &ast.BasicLit{
		Kind:  token.STRING,
		Value: node.GoString(),
	}
	if len(node.Interps) == 0 && node.Kind == parser.DoubleQuote {
		return str
	}

	args := []ast.Expr{str}
	for _, a := range node.OrderedInterps() {
		args = append(args, g.CompileExpr(a))
	}

	g.AddImports("fmt")

	formatted := bst.Call("fmt", "Sprintf", args...)
	switch node.Kind {
	case parser.Regexp:
		g.AddImports("regexp")
		var patt *ast.Ident
		if len(node.Interps) == 0 {
			// Ideally, people aren't regenerating regexes based on user input, so we can compile them at init time
			patt = globalIdents.New("patt")
			g.addGlobalVar(patt, nil, bst.Call("regexp", "MustCompile", str))
		} else {
			// ...but if not, just do it inline and swallow the error for now
			patt = g.it.New("patt")
			g.appendToCurrentBlock(bst.Define(
				[]ast.Expr{patt, g.it.Get("_")},
				bst.Call("regexp", "Compile", formatted),
			))
		}
		return patt
	default:
		return formatted
	}
}

func (g *GoProgram) TransformMethodCall(c *parser.MethodCall) types.Transform {
	var blk *types.Block
	if c.Block != nil {
		blk = g.BuildBlock(c.Block)
	}
	return g.getTransform(g.CompileExpr(c.Receiver), c.Receiver.Type(), c.MethodName, c.Args, blk)
}

func (g *GoProgram) getTransform(rcvr ast.Expr, rcvrType types.Type, methodName string, args parser.ArgsNode, blk *types.Block) types.Transform {
	argExprs := []types.TypeExpr{}
	for _, a := range args {
		argExprs = append(argExprs, types.TypeExpr{Expr: g.CompileExpr(a), Type: a.Type()})
	}
	transform := rcvrType.TransformAST(
		methodName,
		rcvr,
		argExprs,
		blk,
		g.it,
	)
	g.AddImports(transform.Imports...)
	return transform
}

func (g *GoProgram) BuildBlock(blk *parser.Block) *types.Block {
	g.pushTracker()
	args := []ast.Expr{}
	for _, p := range blk.Params {
		args = append(args, g.it.Get(p.Name))
	}
	g.newBlockStmt()
	g.PushState(InBlockBody)
	defer func() {
		g.popBlockStmt()
		g.PopState()
	}()
	for _, s := range blk.Body.Statements {
		g.CompileStmt(s)
	}
	g.popTracker()
	return &types.Block{
		ReturnType: blk.Body.ReturnType,
		Args:       args,
		Statements: g.currentBlockStmt().List,
	}
}

func (g *GoProgram) AddImports(packages ...string) {
	for _, pkg := range packages {
		if _, present := g.Imports[pkg]; !present {
			g.Imports[pkg] = true
		}
	}
}

func (g *GoProgram) mapToExprs(nodes []parser.Node) []ast.Expr {
	exprs := []ast.Expr{}
	for _, n := range nodes {
		exprs = append(exprs, g.CompileExpr(n))
	}
	return exprs
}

func (g *GoProgram) retTypeField(t types.Type) *ast.Field {
	var retType ast.Expr
	switch r := t.(type) {
	case types.Array:
		retType = &ast.ArrayType{
			Elt: g.it.Get(r.Element.GoType()),
		}
	default:
		if r == types.NilType {
			retType = g.it.Get("")
		} else {
			retType = g.it.Get(r.GoType())
		}
	}
	return &ast.Field{Type: retType}
}

func (g *GoProgram) addGlobalVar(name *ast.Ident, typeExpr ast.Expr, val ast.Expr) {
	g.GlobalVars = append(g.GlobalVars, &ast.ValueSpec{
		Names:  []*ast.Ident{name},
		Type:   typeExpr,
		Values: []ast.Expr{val},
	})
}

func (g *GoProgram) addConstant(name *ast.Ident, val ast.Expr) {
	g.Constants = append(g.Constants, &ast.ValueSpec{
		Names:  []*ast.Ident{name},
		Values: []ast.Expr{val},
	})
}

func isSimple(i interface{}) bool {
	switch i.(type) {
	case *ast.BasicLit:
		return true
	case *ast.Ident:
		return true
	default:
		return false
	}
}
