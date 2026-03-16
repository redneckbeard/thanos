package parser

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/redneckbeard/thanos/types"
)

var (
	setterPatt     = regexp.MustCompile(`\w+=`)
	interrogPatt   = regexp.MustCompile(`\w+\?`)
	interrogPrefix = regexp.MustCompile(`^(Can|Is|Does|Has)`)
)

type MethodSet struct {
	Methods map[string]*Method
	Order   []string
	Calls   map[string][]*MethodCall
	Class   *Class
	Module  *Module
}

func (ms *MethodSet) AddMethod(m *Method) {
	ms.Methods[m.Name] = m
	ms.Order = append(ms.Order, m.Name)
}

func (ms *MethodSet) AddCall(c *MethodCall) {
	ms.Calls[c.MethodName] = append(ms.Calls[c.MethodName], c)
	cls := ms.Class
	// Only propagate to parent if this class doesn't define the method itself
	if cls != nil && cls.Parent() != nil {
		if _, defined := ms.Methods[c.MethodName]; !defined {
			cls.Parent().MethodSet.AddCall(c)
		}
	}
}

func NewMethodSet() *MethodSet {
	return &MethodSet{
		Methods: make(map[string]*Method),
		Calls:   make(map[string][]*MethodCall),
	}
}

var globalMethodSet *MethodSet

type ParamKind int

const (
	Positional ParamKind = iota
	Named
	Keyword
	ExplicitBlock
	Splat
	DoubleSplat
	Destructured
)

type Param struct {
	Position int
	Name     string
	Kind     ParamKind
	_type    types.Type
	Default  Node
	Required bool
	Nested   []*Param
}

func (p *Param) Type() types.Type {
	if p._type != nil {
		if p.Kind == Splat {
			return types.NewArray(p._type)
		}
		return p._type
	}
	if p.Default != nil {
		return p.Default.Type()
	}
	return nil
}

func (p *Param) String() string {
	switch p.Kind {
	case Positional:
		return p.Name
	case Named:
		return fmt.Sprintf("%s = %s", p.Name, p.Default)
	case Keyword:
		return fmt.Sprintf("%s: %s", p.Name, p.Default)
	case Splat:
		return "*" + p.Name
	}
	panic("kind not set!")
}

type ParamList struct {
	Params   []*Param
	ParamMap map[string]*Param
}

func NewParamList() *ParamList {
	return &ParamList{ParamMap: make(map[string]*Param)}
}

func (list *ParamList) AddParam(p *Param) error {
	if _, found := list.ParamMap[p.Name]; found {
		return fmt.Errorf("parameter '%s' declared twice", p.Name)
	}
	list.Params = append(list.Params, p)
	list.ParamMap[p.Name] = p
	p.Position = len(list.Params) - 1
	return nil
}

func (list *ParamList) GetParam(i int) (*Param, error) {
	if i < len(list.Params) {
		return list.Params[i], nil
	} else if len(list.Params) > 0 {
		last := list.Params[len(list.Params)-1]
		if last.Kind == Splat {
			return last, nil
		}
	}
	return nil, errors.New("out of bounds")
}

func (list *ParamList) PositionalParams() []*Param {
	params := []*Param{}
	for i := 0; ; i++ {
		p, err := list.GetParam(i)
		if err != nil || p.Kind != Positional {
			break
		}
		params = append(params, p)
	}
	return params
}

func (list *ParamList) GetParamByName(s string) *Param {
	return list.ParamMap[s]
}

func (list *ParamList) DoubleSplatParam() *Param {
	for _, p := range list.Params {
		if p.Kind == DoubleSplat {
			return p
		}
	}
	return nil
}

type BlockParam struct {
	Name       string
	ReturnType types.Type
	*ParamList
}

type Method struct {
	Receiver Node
	Name     string
	Body     *Body
	*ParamList
	Locals    *SimpleScope
	Scope     ScopeChain
	Root      *Root
	Block     *BlockParam
	Pos
	Private     bool
	ClassMethod bool
	FromGem     bool
	analyzed    bool
	analyzing   bool
	uncallable  bool
}

func NewMethod(name string, r *Root) *Method {
	locals := NewScope(name)
	r.currentMethod = &Method{
		Name:      name,
		ParamList: NewParamList(),
		Locals:    locals,
		Scope:     r.ScopeChain.Extend(locals),
		Root:      r,
		FromGem:   r.loadingGem,
	}
	return r.currentMethod
}

func (n *Method) paramString() string {
	strs := []string{}
	for _, p := range n.Params {
		strs = append(strs, p.String())
	}
	if n.Block != nil {
		strs = append(strs, "&"+n.Block.Name)
	}
	return strings.Join(strs, ", ")
}

func (n *Method) String() string {
	if n.Receiver != nil {
		return fmt.Sprintf("(def %s#%s(%s) %s)", n.Receiver, n.Name, n.paramString(), Indent(n.Body.String()))
	} else {
		return fmt.Sprintf("(def %s(%s) %s)", n.Name, n.paramString(), Indent(n.Body.String()))
	}
}

func (n *Method) IsUncallable() bool    { return n.uncallable }
func (n *Method) Type() types.Type     { return types.FuncType }
func (n *Method) SetType(t types.Type) {}
func (n *Method) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.FuncType, nil
}
func (n *Method) Copy() Node {
	//TODO how will we actually clone for subclassing and super?
	return n
}

func (m *Method) ReturnType() types.Type {
	return m.Body.ReturnType
}

var operatorGoNames = map[string]string{
	"<=>": "Spaceship",
	"<":   "Lt",
	">":   "Gt",
	"<=":  "Lte",
	">=":  "Gte",
	"==":  "Eq",
	"+":   "Plus",
	"-":   "Minus",
	"*":   "Times",
	"/":   "Div",
	"%":   "Mod",
	"<<":  "Lshift",
	">>":  "Rshift",
	"[]":  "Index",
	"[]=": "IndexSet",
}

func GoName(rubyName string) string {
	if goName, ok := operatorGoNames[rubyName]; ok {
		return goName
	}
	name := strings.TrimRight(rubyName, "!")
	name = strings.Title(name)
	if setterPatt.MatchString(name) {
		name = "Set" + strings.TrimRight(name, "=")
	}
	if interrogPatt.MatchString(name) {
		name = strings.TrimRight(name, "?")
		if !interrogPrefix.MatchString(name) {
			name = "Is" + name
		}
	}
	return name
}

func (m *Method) GoName() string {
	name := GoName(m.Name)
	if m.Private {
		name = strings.ToLower(name[:1]) + name[1:]
	}
	return name
}

func (m *Method) AddParam(p *Param) error {
	if p.Kind == ExplicitBlock {
		m.Block = &BlockParam{Name: p.Name, ParamList: NewParamList()}
		return nil
	}
	err := m.ParamList.AddParam(p)
	if err != nil {
		return NewParseError(m, err.Error()).Terminal()
	}
	m.Locals.Set(p.Name, &RubyLocal{})
	return nil
}

func (m *Method) Analyze(ms *MethodSet) error {
	if DebugLevel() >= 5 {
		fmt.Fprintf(os.Stderr, "DEBUG Method.Analyze: %s, calls=%d\n", m.Name, len(ms.Calls[m.Name]))
	}
	for _, c := range ms.Calls[m.Name] {
		if err := m.AnalyzeArguments(ms.Class, c, nil); err != nil {
			return err
		}
	}
	for _, param := range m.Params {
		if DebugLevel() >= 5 {
			fmt.Fprintf(os.Stderr, "DEBUG   param %s type=%v\n", param.Name, param.Type())
		}
		if param.Type() == nil {
			name := m.Name
			if ms.Class != nil {
				name = ms.Class.Name() + "#" + name
			}
			doubleSplat := m.DoubleSplatParam()
			if doubleSplat != nil && doubleSplat.Type() != nil && param.Kind == Keyword {
				m.Locals.Set(param.Name, &RubyLocal{_type: doubleSplat.Type().(types.Hash).Value})
			} else if len(ms.Calls[m.Name]) == 0 {
				if ms.Class != nil && ms.Class.DataDefine && m.Name == "initialize" {
					// Data.define class whose initialize is never called.
					// Use AnyType for all fields so the class can compile
					// with interface{} types. Fields are refined later if
					// concrete .new() calls are discovered.
					for _, p := range m.Params {
						if p.Type() == nil {
							p._type = types.AnyType
							m.Locals.Set(p.Name, &RubyLocal{_type: types.AnyType})
						}
					}
					for _, ivar := range ms.Class.IVars(nil) {
						if ivar.Type() == nil {
							ivar._type = types.AnyType
						}
					}
					// Don't mark as uncallable — let the body be analyzed.
					break
				}
				if m.FromGem {
					// Gem method is never called — can't infer param types.
					// Mark as uncallable and skip without erroring.
					m.uncallable = true
					return nil
				}
				return NewParseError(m, "unable to detect type signature of method '%s' because it is never called", name)
			} else {
				return NewParseError(m, "unable to detect type signature of method '%s' because it is never called", name)
			}
		} else {
			m.Locals.Set(param.Name, &RubyLocal{_type: param.Type()})
		}
	}
	if err := m.analyzeMethodBody(ms.Class, nil, nil); err != nil {
		return err
	}
	for _, c := range ms.Calls[m.Name] {
		c.Method = m
		if c.Type() == nil {
			c.SetType(m.ReturnType())
		}
		// Type caller's block params from method's yield arg types
		if c.Block != nil && m.Block != nil && len(m.Block.Params) > 0 {
			for i, p := range c.Block.Params {
				if i < len(m.Block.Params) && m.Block.Params[i].Type() != nil {
					p._type = m.Block.Params[i].Type()
				}
			}
		}
	}
	m.analyzed = true
	return nil
}

// resetBlockCallTypes clears cached types on blk.call() nodes,
// block_given? conditionals, and any MethodCall that takes a blk.call()
// result as an argument (e.g., result << yield(item)). This allows the
// method body to be re-analyzed with a known block return type.
func (m *Method) resetBlockCallTypes() {
	if m.Block == nil {
		return
	}
	blockName := m.Block.Name
	var walk func(n Node) bool // returns true if n is/contains a block call
	walk = func(n Node) bool {
		if n == nil {
			return false
		}
		switch v := n.(type) {
		case Statements:
			for _, s := range v {
				walk(s)
			}
		case *Condition:
			if v.isBlockGivenGuard() {
				v._type = nil
			}
			walk(v.True)
			if v.False != nil {
				walk(v.False)
			}
		case *AssignmentNode:
			for _, r := range v.Right {
				walk(r)
			}
		case *MethodCall:
			isBlockCall := false
			if v.MethodName == "call" {
				if ident, ok := v.Receiver.(*IdentNode); ok && ident.Val == blockName {
					v._type = nil
					isBlockCall = true
				}
			}
			// Check if any arg is/contains a block call
			for _, arg := range v.Args {
				if walk(arg) {
					// This call uses a block call result; clear its type too
					v._type = nil
					isBlockCall = true
				}
			}
			// Walk into block bodies (e.g., items.each { ... yield ... })
			if v.Block != nil && v.Block.Body != nil {
				if walk(v.Block.Body.Statements) {
					// The block body contains a block call — clear the
					// enclosing each/map call type and block body return type
					v._type = nil
					v.Block.Body.ReturnType = nil
				}
			}
			return isBlockCall
		case *WhileNode:
			walk(v.Body)
		case *ForInNode:
			walk(v.Body)
		case *InfixExpressionNode:
			leftHasBlock := walk(v.Left)
			rightHasBlock := walk(v.Right)
			if leftHasBlock || rightHasBlock {
				v._type = nil
				return true
			}
		case *ReturnNode:
			if v.Val != nil {
				for _, val := range v.Val {
					if walk(val) {
						return true
					}
				}
			}
		case *ArrayNode:
			for _, arg := range v.Args {
				walk(arg)
			}
		}
		return false
	}
	walk(m.Body.Statements)
	// Reset locals whose types derived from block calls (e.g., arrays
	// refined by << with block call results).
	m.Locals.Each(func(name string, local Local) {
		if rl, ok := local.(*RubyLocal); ok {
			if arr, isArr := rl._type.(types.Array); isArr && arr.Element == types.NilType {
					rl._type = types.NewArray(types.AnyType)
				rl.MarkAsRefinable()
			}
		}
	})
}

// analyzeMethodBody is the shared analysis logic for both instance methods (via
// Method.Analyze) and class methods (via MethodSpec closures). It sets param
// types, registers block locals, infers the body return type, and extracts
// yield arg types.
//
// When blockReturnType is non-nil and no callBlock is provided, a synthetic
// Block is created so that blk.call() dispatch in AnalyzeArguments correctly
// propagates the block return type (instead of returning NilType).
func (m *Method) analyzeMethodBody(class *Class, args []types.Type, blockReturnType types.Type) error {
	for i, param := range m.Params {
		if i < len(args) {
			param._type = args[i]
			m.Locals.Set(param.Name, &RubyLocal{_type: args[i]})
		}
	}
	if m.Block != nil {
		if blockReturnType != nil {
			// Clear cached types on blk.call() nodes from any prior analysis
			// so GetType will re-dispatch through the synthetic Block.
			m.resetBlockCallTypes()
			m.Body.ReturnType = nil
			// Create a synthetic Block so blk.call() takes the *Block path
			// in AnalyzeArguments, which sets Block.ReturnType properly
			synBlock := &Block{
				Body:      &Body{ReturnType: blockReturnType},
				Scope:     m.Scope.Extend(NewScope("block")),
				Method:    m,
				ParamList: m.Block.ParamList,
			}
			m.Locals.Set(m.Block.Name, synBlock)
		} else {
			m.Locals.Set(m.Block.Name, &RubyLocal{_type: types.NewProc()})
		}
	}
	if err := m.Body.InferReturnType(m.Scope, class); err != nil {
		return err
	}
	if m.Block != nil && len(m.Block.Params) == 0 {
		m.extractYieldArgTypes()
	}
	return nil
}

// extractYieldArgTypes walks the method body to find blk.call() patterns
// and populates the BlockParam with the yield argument types.
func (m *Method) extractYieldArgTypes() {
	var walk func(nodes Statements)
	walk = func(nodes Statements) {
		for _, node := range nodes {
			if DebugLevel() >= 5 {
				fmt.Fprintf(os.Stderr, "DEBUG extractYield walk: %T %s\n", node, node)
				if mc, ok2 := node.(*MethodCall); ok2 && mc.Receiver != nil {
					fmt.Fprintf(os.Stderr, "DEBUG extractYield   receiver: %T %s type=%v\n", mc.Receiver, mc.Receiver, mc.Receiver.Type())
				}
			}
			if mc, ok := node.(*MethodCall); ok {
				if mc.MethodName == "call" {
					if ident, ok := mc.Receiver.(*IdentNode); ok && ident.Val == m.Block.Name {
						if DebugLevel() >= 5 {
							fmt.Fprintf(os.Stderr, "DEBUG extractYield: found yield, args=%d\n", len(mc.Args))
							for i, arg := range mc.Args {
								fmt.Fprintf(os.Stderr, "DEBUG extractYield:   arg[%d] type=%v\n", i, arg.Type())
							}
						}
						for i, arg := range mc.Args {
							if arg.Type() != nil {
								name := fmt.Sprintf("arg%d", i)
								if DebugLevel() >= 5 {
									fmt.Fprintf(os.Stderr, "DEBUG extractYield: adding param %s type=%v to BlockParam (len before=%d)\n", name, arg.Type(), len(m.Block.Params))
								}
								m.Block.AddParam(&Param{Name: name, _type: arg.Type()})
								if DebugLevel() >= 5 {
									fmt.Fprintf(os.Stderr, "DEBUG extractYield: BlockParam now has %d params\n", len(m.Block.Params))
								}
							}
						}
						return
					}
				}
				// Check inside method call args (e.g., result << yield(item))
				for _, arg := range mc.Args {
					walk(Statements{arg})
				}
				// Check inside blocks of method calls
				if mc.Block != nil {
					walk(mc.Block.Body.Statements)
				}
			} else if ret, ok := node.(*ReturnNode); ok {
				walk(Statements(ret.Val))
			} else if cond, ok := node.(*Condition); ok {
				walk(cond.True)
				if cond.False != nil {
					if elseIf, ok := cond.False.(*Condition); ok {
						walk(Statements{elseIf})
					}
				}
			} else if caseNode, ok := node.(*CaseNode); ok {
				for _, w := range caseNode.Whens {
					walk(w.Statements)
				}
			} else if whileNode, ok := node.(*WhileNode); ok {
				walk(whileNode.Body)
			} else if forNode, ok := node.(*ForInNode); ok {
				walk(forNode.Body)
			} else if assign, ok := node.(*AssignmentNode); ok {
				for _, r := range assign.Right {
					walk(Statements{r})
				}
			} else if beginNode, ok := node.(*BeginNode); ok {
				walk(beginNode.Body)
			} else if infix, ok := node.(*InfixExpressionNode); ok {
				walk(Statements{infix.Left})
				walk(Statements{infix.Right})
			}
		}
	}
	walk(m.Body.Statements)
}

func (method *Method) AnalyzeArguments(class *Class, c *MethodCall, scope ScopeChain) error {
	for _, p := range method.Params {
		if p.Default != nil {
			// Use the method's scope for default param resolution so that
			// scoped constants like Diff::LCS::BalancedCallbacks are visible.
			defaultScope := method.Scope
			if defaultScope == nil {
				defaultScope = ScopeChain{class}
			}
			t, err := GetType(p.Default, defaultScope, class)
			if err != nil {
				return err
			}
			//TODO this is happening in at least three places
			local := &RubyLocal{_type: t}
			// When the default is nil, the actual type will come from a
			// call-site argument or a body reassignment (e.g. `x ||= val`).
			// Use AnyType so the variable can be refined later.
			if _, isNil := p.Default.(*NilNode); isNil && t == types.NilType {
				local._type = types.AnyType
				local.MarkAsRefinable()
			}
			method.Locals.Set(p.Name, local)
		}
	}

	if c == nil {
		return nil
	}

	c.Method = method

	if len(method.PositionalParams()) > len(c.PositionalArgs()) {
		return NewParseError(c, "method '%s' called with %d positional arguments but %d expected", method.Name, len(c.PositionalArgs()), len(method.PositionalParams())).Terminal()
	}

	if c.AllKwargsForDoubleSplat() {
		param := c.Method.DoubleSplatParam()
		if param.Type() == nil {
			t, _ := GetType(c.Args[0], scope, class)
			param._type = types.NewHash(types.SymbolType, t)
			method.Scope.Set(param.Name, &RubyLocal{_type: param.Type()})
		}
		return nil
	}

	var (
		splatSeen  bool
		splatIndex int
	)

	for i, arg := range c.Args {
		var param *Param
		if kv, ok := arg.(*KeyValuePair); ok {
			param = method.GetParamByName(kv.Label)
			if param == nil {
				param = method.DoubleSplatParam()
				if param == nil {
					return NewParseError(c, "method '%s' called with keyword argument '%s' but '%s' has no such parameter", method.Name, kv.Label, method.Name).Terminal()
				}
			}
		} else {
			var err error
			if splatSeen {
				param, err = method.GetParam(splatIndex)
				c.splatLength++
			} else {
				param, err = method.GetParam(i)
			}
			if err != nil {
				if DebugLevel() >= 1 {
					fmt.Fprintf(os.Stderr, "DEBUG arg overflow: method=%s params=%d args=%d i=%d file=%s classMethod=%v\n", method.Name, len(method.Params), len(c.Args), i, c.file, method.ClassMethod)
				}
				return NewParseError(c, "method '%s' called with %d arguments but %d expected", method.Name, i+1, i).Terminal()
			}
			if param.Kind == Splat && !splatSeen {
				splatSeen, splatIndex = true, i
				c.splatStart = i
				c.splatLength = 1
			}
		}
		if scope == nil {
			scope = method.Scope
		}
		if param.Type() == nil {
			// unset, so set it
			if t, err := GetType(arg, scope, class); err == nil {
				if _, ok := t.(types.Hash); !ok && param.Kind == DoubleSplat {
					t = types.NewHash(types.SymbolType, t)
				}
				param._type = t
				method.Scope.Set(param.Name, &RubyLocal{_type: param.Type()})
			}
		} else {
			t, err := GetType(arg, scope, class)
			if err == nil && t != param.Type() && param.Type() == types.NilType {
				// Default was nil — adopt the actual call-site type
				param._type = t
				method.Scope.Set(param.Name, &RubyLocal{_type: t})
			} else if err == nil && t != param.Type() {
				if param.Kind == Splat {
					if splat, ok := arg.(*SplatNode); ok {
						t = splat.Type().(types.Array).Inner()
					}
					if t != param.Type().(types.Array).Inner() {
						return NewParseError(c, "method '%s' called with %s and %s for splat parameter '%s' but heterogenous splat arguments are not yet supported", method.Name, t, param.Type().(types.Array).Inner(), param.Name).Terminal()
					}
				} else if kv, ok := arg.(*KeyValuePair); ok && kv.DoubleSplat {
					if t.(types.Hash).Value != param.Type() {
						return NewParseError(c, "method '%s' called with double splat argument for parameter '%s' but hash value %s does not match", method.Name, param.Name, param.Type()).Terminal()
					}
				} else if kv, ok := arg.(*KeyValuePair); ok && param.Kind == DoubleSplat && !kv.DoubleSplat {
					// the keyword argument 'foo' is valid when a double splat parameter is also named 'foo', so we have to carve out an exception for that here
					return nil
				} else {
					// Try duck-type interface inference before erroring
					if iface := tryBuildDuckInterface(method, param, param.Type(), t); iface != nil {
						param._type = iface
						method.Scope.Set(param.Name, &RubyLocal{_type: iface})
						// Register the interface globally for the compiler
						found := false
						for _, existing := range DuckInterfaces {
							if existing.Name == iface.Name {
								found = true
								break
							}
						}
						if !found {
							DuckInterfaces = append(DuckInterfaces, iface)
						}
					} else {
						return NewParseError(c, "method '%s' called with %s for parameter '%s' but '%s' was previously seen as %s", method.Name, t, param.Name, param.Name, param.Type()).Terminal()
					}
				}
			}
		}
	}
	return nil
}

type Block struct {
	Body       *Body
	Scope      ScopeChain
	Method     *Method
	SymbolProc string // Non-empty for &:method_name blocks
	*ParamList
}

func (b *Block) String() string {
	strs := []string{}
	for _, p := range b.Params {
		strs = append(strs, p.Name)
	}

	return fmt.Sprintf("(|%s| %s)", strings.Join(strs, ", "), b.Body)
}

func (b *Block) Type() types.Type {
	return types.NewProc()
}

func (b *Block) Copy() *Block {
	//TODO almost certainly wrong
	return b
}

type MethodCall struct {
	Receiver                Node
	Method                  *Method
	MethodName              string
	Args                    ArgsNode
	Block                   *Block
	BlockPass               *BlockPassNode
	RawBlock                string
	Getter, Setter          bool
	Op                      string
	splatStart, splatLength int
	_type                   types.Type
	Pos
}

func (n *MethodCall) String() string {
	var s string
	args := []string{}
	if len(n.Args) > 0 {
		args = append(args, n.Args.String())
	}
	if n.Block != nil {
		args = append(args, "block = "+n.Block.String())
	}
	s = fmt.Sprintf("%s(%s)", n.MethodName, strings.Join(args, ", "))

	if n.Receiver != nil {
		s = n.Receiver.String() + "." + s
	}
	return fmt.Sprintf("(%s)", s)
}

func (n *MethodCall) Type() types.Type     { return n._type }
func (n *MethodCall) SetType(t types.Type) { n._type = t }

func (c *MethodCall) TargetType(scope ScopeChain, class *Class) (types.Type, error) {
	// defined?(expr) compiles to true at compile time — if the symbol exists,
	// the compiler already resolved it; if not, it would have errored earlier.
	if c.MethodName == "defined?" {
		return types.BoolType, nil
	}
	// Extract &:symbol from args and convert to a synthetic block
	c.extractSymbolToProc()
	// Extract &variable block pass from args
	c.extractBlockPass()
	receiverType := c.ReceiverType(scope, class)
	switch t := receiverType.(type) {
	case *types.Class:
		if c.MethodName == "new" && t.UserDefined {
			receiverType := t.Instance.(types.Type)
			initializeCall := &MethodCall{
				MethodName: "initialize",
				Args:       c.Args,
				Block:      c.Block,
				_type:      receiverType,
				Pos: Pos{lineNo: c.lineNo},
			}
			classMethodSets[receiverType].AddCall(initializeCall)
		}
	case types.Instance:
		// We'll only have a methodset for a user-defined class instance type
		if ms, ok := classMethodSets[t]; ok {
			ms.AddCall(c)
		}
	case *types.Proc:
		if c.MethodName == "call" {
			localName := c.Receiver.(*IdentNode).Val
			if local := scope.ResolveVar(localName); local != BadLocal {
				// yield-desugared block params are RubyLocals with Proc type
				if _, ok := local.(*RubyLocal); ok {
					// Type the args but return NilType — yield return type
					// is determined by the caller's block
					for _, arg := range c.Args {
						if _, err := GetType(arg, scope, class); err != nil {
							return nil, err
						}
					}
					return types.NilType, nil
				}
				blk := local.(*Block)
				for i, arg := range c.Args {
					if t, err := GetType(arg, scope, class); err != nil {
						return nil, err
					} else {
						p, err := blk.GetParam(i)
						if err != nil {
							return nil, NewParseError(c, err.Error())
						}
						p._type = t
						method := blk.Method
						method.Block.AddParam(p)
						blk.Scope.Set(p.Name, &RubyLocal{_type: t})
					}
				}
				// Synthetic blocks (from analyzeMethodBody) have ReturnType
				// pre-set with no Statements — skip InferReturnType.
				if blk.Body.ReturnType == nil {
					err := blk.Body.InferReturnType(blk.Scope, nil)
					if err != nil {
						return nil, err
					}
				}
				blk.Method.Block.ReturnType = blk.Body.ReturnType
				return blk.Body.ReturnType, nil
			}
		}
	}
	// Safe navigation operator: unwrap Optional for method resolution
	safeNav := false
	if c.Op == "&." {
		if opt, ok := receiverType.(types.Optional); ok {
			receiverType = opt.Element
			safeNav = true
		}
	}
	if c.Receiver != nil {
		if receiverType == nil {
			return nil, fmt.Errorf("Method '%s' called on '%s' but type of '%s' is not inferred", c.MethodName, c.Receiver, c.Receiver)
		}
		if !receiverType.HasMethod(c.MethodName) {
			if ms, ok := classMethodSets[receiverType]; ok && ms.Class != nil {
				if DebugLevel() >= 5 {
					fmt.Fprintf(os.Stderr, "DEBUG ivar lookup for %s on %s, ivars=%d\n", c.MethodName, receiverType, len(ms.Class.IVars(nil)))
					for _, iv := range ms.Class.IVars(nil) {
						fmt.Fprintf(os.Stderr, "  DEBUG ivar: name=%s readable=%v\n", iv.Name, iv.Readable)
					}
				}
				for _, ivar := range ms.Class.IVars(nil) {
					if c.MethodName == ivar.Name && ivar.Readable && len(c.Args) == 0 {
						c.Getter = true
						return ivar.Type(), nil
					} else if c.MethodName == ivar.Name+"=" && ivar.Writeable {
						c.Setter = true
						return ivar.Type(), nil
					}
				}
			} else if DebugLevel() >= 5 {
				fmt.Fprintf(os.Stderr, "DEBUG classMethodSets lookup FAILED for %s (type %T)\n", receiverType, receiverType)
			}
			return nil, NewParseError(c, "No known method '%s' on %s", c.MethodName, receiverType)
		}
	}
	// Handle class-level directives before arg type resolution, since their
	// args (symbols, constants) may not be resolvable as types.
	if c.Receiver == nil && class != nil {
		switch c.MethodName {
		case "attr_reader":
			class.AddIVars(c.Args, true, false)
			delete(class.MethodSet.Calls, c.MethodName)
			return nil, nil
		case "attr_writer":
			class.AddIVars(c.Args, false, true)
			delete(class.MethodSet.Calls, c.MethodName)
			return nil, nil
		case "attr_accessor":
			class.AddIVars(c.Args, true, true)
			delete(class.MethodSet.Calls, c.MethodName)
			return nil, nil
		case "alias_method":
			if len(c.Args) == 2 {
				newName := strings.TrimLeft(c.Args[0].(*SymbolNode).Val, ":")
				oldName := strings.TrimLeft(c.Args[1].(*SymbolNode).Val, ":")
				class.Aliases = append(class.Aliases, Alias{NewName: newName, OldName: oldName})
			}
			delete(class.MethodSet.Calls, c.MethodName)
			return nil, nil
		case "include":
			// Already handled during parsing in AddCall
			delete(class.MethodSet.Calls, c.MethodName)
			return nil, nil
		case "private_constant", "public_constant", "freeze":
			// Class-level directives with no compilation effect
			delete(class.MethodSet.Calls, c.MethodName)
			return nil, nil
		}
	}

	argTypes := []types.Type{}
	// When the MethodSpec has KwargsSpec, reorder arg types to match:
	// [positional..., kwarg1, kwarg2, ...] in KwargsSpec declaration order.
	var kwargsSpec []types.KwargSpec
	if receiverType != nil {
		if spec, hasSpec := receiverType.GetMethodSpec(c.MethodName); hasSpec {
			kwargsSpec = spec.KwargsSpec
		}
	}
	if len(kwargsSpec) > 0 {
		kwargTypes := map[string]types.Type{}
		for _, a := range c.Args {
			if kv, ok := a.(*KeyValuePair); ok {
				if t, err := GetType(kv.Value, scope, class); err != nil {
					return nil, err
				} else {
					kwargTypes[kv.Label] = t
				}
			} else {
				if t, err := GetType(a, scope, class); err != nil {
					return nil, err
				} else {
					argTypes = append(argTypes, t)
				}
			}
		}
		for _, ks := range kwargsSpec {
			if t, ok := kwargTypes[ks.Name]; ok {
				argTypes = append(argTypes, t)
			} else {
				argTypes = append(argTypes, nil)
			}
		}
	} else {
		for _, a := range c.Args {
			if t, err := GetType(a, scope, class); err != nil {
				return nil, err
			} else {
				argTypes = append(argTypes, t)
			}
		}
	}

	var method *Method

	if ms, ok := classMethodSets[receiverType]; ok {
		if m, userDefined := ms.Methods[c.MethodName]; userDefined {
			method = m
			class = ms.Class
		}
	} else if c.MethodName == "new" {
		if ms, ok := classMethodSets[receiverType.(*types.Class).Instance.(types.Type)]; ok {
			if m, userDefined := ms.Methods["initialize"]; userDefined {
				method = m
				class = ms.Class
			}
		}
	} else if c.Receiver == nil {
		switch c.MethodName {
		default:
			// Inside a class context, check class methods and instance methods
			// before falling back to the global method set. This handles cases
			// like `def self.patch!` calling peer class method `patch(...)`.
			if class != nil {
				for _, m := range class.ClassMethods {
					if m.Name == c.MethodName {
						method = m
						break
					}
				}
				if method == nil {
					method = class.MethodSet.Methods[c.MethodName]
				}
			}
			// Inside a module class method, check sibling class methods on
			// the enclosing module (e.g., position_hash called from self.lcs
			// in module Internals).
			if method == nil && class == nil {
				for i := len(scope) - 1; i >= 0; i-- {
					if mod, ok := scope[i].(*Module); ok {
						for _, m := range mod.ClassMethods {
							if m.Name == c.MethodName {
								method = m
								break
							}
						}
						if method != nil {
							break
						}
					}
				}
			}
			if method == nil {
				method = globalMethodSet.Methods[c.MethodName]
			}
			if method == nil {
				return nil, NewParseError(c, "Tried calling method '%s' inside but no such method exists", c.MethodName)
			}
		}
	}

	var blockRetType types.Type
	if method != nil {
		//TODO should be consolidated with AnalyzeArguments/AnalyzeMethodSet
		c.Method = method
		method.AnalyzeArguments(class, c, scope)
		// Guard against re-entrant analysis (e.g., list.each calls user's each
		// which calls @items.each — avoid infinite recursion).
		if method.analyzing {
			if method.Name == "initialize" {
				if cls, ok := receiverType.(*types.Class); ok {
					return cls.Instance.(types.Type), nil
				}
				return receiverType, nil
			}
			if rt := method.ReturnType(); rt != nil {
				return rt, nil
			}
			return types.NilType, nil
		}
		// Type block params from the method's BlockParam (yield arg types).
		// This must run AFTER the unanalyzed guard above, since
		// extractYieldArgTypes populates method.Block.Params during
		// AnalyzeMethodSet, which runs between the first and second passes.
		if c.Block != nil {
			c.Block.Scope = scope.Extend(NewScope("block"))
			c.Block.Method = method
			if method.Block != nil {
				method.Locals.Set(method.Block.Name, c.Block)
				for i, p := range c.Block.Params {
					if i < len(method.Block.Params) && method.Block.Params[i].Type() != nil {
						p._type = method.Block.Params[i].Type()
						c.Block.Scope.Set(p.Name, &RubyLocal{_type: p._type})
					}
				}
				if c.Block.Body != nil {
					c.Block.Body.InferReturnType(c.Block.Scope, nil)
				}
			}
		}
		// Ensure block local is registered before body analysis
		// (may not be set if method was uncallable during AnalyzeMethodSet)
		if method.Block != nil {
			if _, ok := method.Locals.Get(method.Block.Name); !ok {
				method.Locals.Set(method.Block.Name, &RubyLocal{_type: types.NewProc()})
			}
		}
		method.analyzing = true
		if err := method.Body.InferReturnType(method.Scope, class); err != nil {
			method.analyzing = false
			return nil, err
		} else {
			method.analyzing = false
			if method.Name == "initialize" {
				if cls, ok := receiverType.(*types.Class); ok {
					return cls.Instance.(types.Type), nil
				}
				// receiverType is already an Instance (e.g., from super call)
				return receiverType, nil
			}
			return method.ReturnType(), nil
		}
	} else if c.Receiver == nil {
		return nil, NewParseError(c, "Attempted to call undefined method '%s'", c.MethodName)
	} else {
		// For mixin-provided methods on user-defined types, defer analysis
		// until the class's methods have been analyzed (so extractYieldArgTypes
		// can populate block param types for getElementType resolution).
		if ms, ok := classMethodSets[receiverType]; ok && ms.Class != nil {
			allAnalyzed := true
			for _, m := range ms.Methods {
				if !m.analyzed {
					allAnalyzed = false
					break
				}
			}
			if !allAnalyzed {
				return nil, nil
			}
		}
		// This is all a special case for thanos-defined methods
		if c.Block != nil {
			blockScope := NewScope("block")
			blockArgTypes := receiverType.BlockArgTypes(c.MethodName, argTypes)
			for i, p := range c.Block.Params {
				if i >= len(blockArgTypes) {
					break
				}
				if p.Kind == Destructured {
					// Unpack composite type into nested params
					compositeType := blockArgTypes[i]
					if h, ok := compositeType.(types.Hash); ok {
						if len(p.Nested) >= 1 {
							p.Nested[0]._type = h.Key
							blockScope.Set(p.Nested[0].Name, &RubyLocal{_type: h.Key})
						}
						if len(p.Nested) >= 2 {
							p.Nested[1]._type = h.Value
							blockScope.Set(p.Nested[1].Name, &RubyLocal{_type: h.Value})
						}
					}
				} else {
					p._type = blockArgTypes[i]
					local := &RubyLocal{_type: blockArgTypes[i]}
					if arr, ok := blockArgTypes[i].(types.Array); ok && arr.Element == types.AnyType {
						local.MarkAsRefinable()
					}
					if hash, ok := blockArgTypes[i].(types.Hash); ok {
						// Mark hashes with AnyType key/value as refinable so bracket
						// access and element mutation (<<, push) can refine them.
						needsRefine := hash.Key == types.AnyType || hash.Value == types.AnyType
						if !needsRefine {
							if arr, ok := hash.Value.(types.Array); ok && arr.Element == types.AnyType {
								needsRefine = true
							}
						}
						if needsRefine {
							local.MarkAsRefinable()
						}
					}
					blockScope.Set(p.Name, local)
				}
			}
			err := c.Block.Body.InferReturnType(scope.Extend(blockScope), nil)
			if err != nil {
				return nil, err
			}
			blockRetType = c.Block.Body.ReturnType
			// Propagate refined block param types back to argTypes and receiverType.
			// When a block param was refinable (e.g. empty array or hash with
			// AnyType) and got refined during block body inference, update the
			// corresponding method arg type so MethodReturnType sees the refined
			// type. For methods like tap where the block param IS the receiver,
			// also update receiverType so the return type reflects the refinement.
			for i, p := range c.Block.Params {
				if p.Kind == Destructured {
					continue
				}
				local, found := blockScope.Get(p.Name)
				if !found {
					continue
				}
				refined := local.Type()
				origType := blockArgTypes[i]
				if refined == nil || refined == origType {
					continue
				}
				// If this block param's original type matched the receiver,
				// update receiverType so MethodReturnType sees the refinement.
				// This handles methods like tap where the block receives the receiver.
				if origType == receiverType {
					receiverType = refined
				}
				// Find the argType that matches the original unrefined type
				for j, at := range argTypes {
					if at == origType {
						argTypes[j] = refined
						c.Args[j].SetType(refined)
						break
					}
				}
			}
		}
	}

	if t, err := receiverType.MethodReturnType(c.MethodName, blockRetType, argTypes); err != nil {
		return nil, NewParseError(c, err.Error())
	} else {
		// Fan out duck-interface calls to concrete types so their methods get analyzed
		if di, ok := receiverType.(*types.DuckInterface); ok {
			for _, ct := range di.ConcreteTypes {
				if ms, msOk := classMethodSets[ct]; msOk {
					syntheticCall := &MethodCall{
						Receiver:   &SelfNode{_type: ct, Pos: Pos{lineNo: c.lineNo}},
						MethodName: c.MethodName,
						Args:       c.Args,
						Pos: Pos{lineNo: c.lineNo},
					}
					ms.AddCall(syntheticCall)
				}
			}
		}
		// Check if this method can refine variable types (e.g., << on empty arrays)
		if c.Receiver != nil {
			if ident, ok := c.Receiver.(*IdentNode); ok {
				// Try to get the method spec and call RefineVariable if it exists
				if spec, hasSpec := receiverType.GetMethodSpec(c.MethodName); hasSpec && spec.RefineVariable != nil {
					spec.RefineVariable(ident.Val, t, scope)
				}
			}
			// Refine hash value type when a refining method (push, <<) is called
			// on a hash-accessed element, e.g. h["key"].push("val") refines
			// Hash{K, Array{AnyType}} → Hash{K, Array{String}}
			if ba, ok := c.Receiver.(*BracketAccessNode); ok {
				if spec, hasSpec := receiverType.GetMethodSpec(c.MethodName); hasSpec && spec.RefineVariable != nil {
					if ident, ok := ba.Composite.(*IdentNode); ok {
						if h, isHash := ba.Composite.Type().(types.Hash); isHash {
							if t != receiverType {
								refined := types.NewHash(h.Key, t)
								if h.HasDefault {
									refined = types.NewDefaultHash(h.Key, t)
								}
								scope.RefineVariableType(ident.Val, refined)
							}
						}
					}
				}
			}
		}
		if safeNav {
			return types.NewOptional(t), nil
		}
		return t, nil
	}
}

func (n *MethodCall) Copy() Node {
	return &MethodCall{
		n.Receiver.Copy(),
		n.Method.Copy().(*Method),
		n.MethodName,
		n.Args.Copy().(ArgsNode),
		n.Block.Copy(),
		n.BlockPass,
		"",
		n.Getter,
		n.Setter,
		n.Op,
		n.splatStart,
		n.splatLength,
		n._type,
		n.Pos,
	}
}

func (n *MethodCall) RequiresTransform() bool {
	if n.Receiver == nil {
		return false // for now, will have some built-in top level funcs
	}
	if n.Receiver.Type() == nil {
		return false
	}

	return n.Receiver.Type().HasMethod(n.MethodName)
}

func (c *MethodCall) ReceiverType(scope ScopeChain, class *Class) types.Type {
	if c.Receiver != nil {
		if c.Receiver.Type() != nil {
			return c.Receiver.Type()
		}
		receiverType, err := GetType(c.Receiver, scope, class)
		if err == nil {
			return receiverType
		}
	}
	if types.KernelType.HasMethod(c.MethodName) {
		c.Receiver = &KernelNode{}
		return types.KernelType
	}
	return nil
}

func (c *MethodCall) PositionalArgs() ArgsNode {
	positional := ArgsNode{}
	for _, a := range c.Args {
		if _, ok := a.(*KeyValuePair); !ok {
			positional = append(positional, a)
		}
	}
	return positional
}

func (c *MethodCall) HasSplat() bool {
	for _, arg := range c.Args {
		if _, ok := arg.(*SplatNode); ok {
			return true
		}
	}
	return false
}

func (c *MethodCall) HasDoubleSplat() bool {
	for _, arg := range c.Args {
		if kv, ok := arg.(*KeyValuePair); ok && kv.DoubleSplat {
			return true
		}
	}
	return false
}

func (c *MethodCall) SplatArgs() []Node {
	var args []Node
	if c.splatLength > 0 {
		return c.Args[c.splatStart:(c.splatStart + c.splatLength)]
	}
	return args
}

func (c *MethodCall) KeywordArgsForDoubleSplatParam() *HashNode {
	pairs := []*KeyValuePair{}

	for _, a := range c.Args {
		if kv, ok := a.(*KeyValuePair); ok && !kv.DoubleSplat {
			if param := c.Method.GetParamByName(kv.Label); param == nil || param.Kind == DoubleSplat {
				pairs = append(pairs, kv)
			}
		}
	}
	return &HashNode{Pairs: pairs, Pos: Pos{lineNo: c.lineNo}, _type: c.Method.DoubleSplatParam().Type()}
}

func (c *MethodCall) ExtractDoubleSplatArg() Node {
	for _, a := range c.Args {
		if kv, ok := a.(*KeyValuePair); ok && kv.DoubleSplat {
			return kv.Value
		}
	}
	return nil
}

func (c *MethodCall) AllKwargsForDoubleSplat() bool {
	if c.Method.DoubleSplatParam() == nil || len(c.Args) == 0 {
		return false
	}
	for _, a := range c.Args {
		if kv, ok := a.(*KeyValuePair); ok && !kv.DoubleSplat {
			if c.Method.GetParamByName(kv.Label) != nil {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

func (c *MethodCall) SetBlock(blk *Block) {
	c.Block = blk
	if c.Method != nil {
		for _, p := range blk.Params {
			c.Method.Block.AddParam(p)
		}
	}
}
