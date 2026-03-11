package parser

import "github.com/redneckbeard/thanos/types"

// SymbolToProcNode represents &:method_name in method call arguments.
// It gets converted to a Block during MethodCall.Walk().
type SymbolToProcNode struct {
	MethodName string
	lineNo     int
	_type      types.Type
}

func (n *SymbolToProcNode) String() string       { return "&:" + n.MethodName }
func (n *SymbolToProcNode) Type() types.Type     { return n._type }
func (n *SymbolToProcNode) SetType(t types.Type) { n._type = t }
func (n *SymbolToProcNode) LineNo() int          { return n.lineNo }
func (n *SymbolToProcNode) Copy() Node           { return n }

func (n *SymbolToProcNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	// Type doesn't matter — this node gets extracted before type inference
	return types.SymbolType, nil
}

// extractSymbolToProc checks if the last arg is a SymbolToProcNode and
// converts it to a synthetic block: &:method_name → { |x| x.method_name }
func (c *MethodCall) extractSymbolToProc() {
	if len(c.Args) == 0 || c.Block != nil {
		return
	}
	last := c.Args[len(c.Args)-1]
	stp, ok := last.(*SymbolToProcNode)
	if !ok {
		return
	}
	// Remove from args
	c.Args = c.Args[:len(c.Args)-1]

	// Create synthetic block: { |x| x.method_name }
	paramName := "_elem"
	param := &Param{Name: paramName, Kind: Positional}
	paramIdent := &IdentNode{Val: paramName, lineNo: stp.lineNo}
	methodCall := &MethodCall{
		Receiver:   paramIdent,
		MethodName: stp.MethodName,
		Args:       ArgsNode{},
		lineNo:     stp.lineNo,
	}

	blk := &Block{
		Body: &Body{
			Statements: []Node{methodCall},
		},
		ParamList: NewParamList(),
	}
	blk.AddParam(param)
	c.SetBlock(blk)
}
