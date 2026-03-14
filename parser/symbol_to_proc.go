package parser

import "github.com/redneckbeard/thanos/types"

// SymbolToProcNode represents &:method_name in method call arguments.
// It gets converted to a Block during MethodCall.Walk().
type SymbolToProcNode struct {
	MethodName string
	Pos
	_type      types.Type
}

func (n *SymbolToProcNode) String() string       { return "&:" + n.MethodName }
func (n *SymbolToProcNode) Type() types.Type     { return n._type }
func (n *SymbolToProcNode) SetType(t types.Type) { n._type = t }
func (n *SymbolToProcNode) Copy() Node           { return n }

func (n *SymbolToProcNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	// Type doesn't matter — this node gets extracted before type inference
	return types.SymbolType, nil
}

// BlockPassNode represents &variable in method call arguments.
// It passes an existing block/proc variable to the called method.
type BlockPassNode struct {
	Name  string
	Pos
	_type types.Type
}

func (n *BlockPassNode) String() string       { return "&" + n.Name }
func (n *BlockPassNode) Type() types.Type     { return n._type }
func (n *BlockPassNode) SetType(t types.Type) { n._type = t }
func (n *BlockPassNode) Copy() Node           { return n }

func (n *BlockPassNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.NilType, nil
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
	paramIdent := &IdentNode{Val: paramName, Pos: Pos{lineNo: stp.lineNo}}
	methodCall := &MethodCall{
		Receiver:   paramIdent,
		MethodName: stp.MethodName,
		Args:       ArgsNode{},
		Pos: Pos{lineNo: stp.lineNo},
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

// extractBlockPass checks if the last arg is a BlockPassNode and strips it
// from the args. Block forwarding is handled at compile time — the block
// parameter variable is passed directly as an argument to the called function.
func (c *MethodCall) extractBlockPass() {
	if len(c.Args) == 0 {
		return
	}
	last := c.Args[len(c.Args)-1]
	bp, ok := last.(*BlockPassNode)
	if !ok {
		return
	}
	// Remove from args and record as a block pass
	c.Args = c.Args[:len(c.Args)-1]
	c.BlockPass = bp
}
