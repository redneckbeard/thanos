package parser

import "fmt"

// synthesizeNumberedParams scans a block body for numbered parameters (_1 through _9).
// If found (and no explicit params were declared), it synthesizes positional params
// up to the highest numbered param used.
func synthesizeNumberedParams(blk *Block) {
	if len(blk.Params) > 0 {
		return // explicit params declared, don't synthesize
	}
	highest := findHighestNumberedParam(blk.Body.Statements)
	if highest == 0 {
		return
	}
	for i := 1; i <= highest; i++ {
		blk.AddParam(&Param{
			Name: fmt.Sprintf("_%d", i),
			Kind: Positional,
		})
	}
}

// findHighestNumberedParam walks a slice of nodes looking for IdentNode
// references to _1 through _9, returning the highest number found (0 if none).
func findHighestNumberedParam(nodes []Node) int {
	highest := 0
	for _, n := range nodes {
		if h := scanNodeForNumberedParam(n); h > highest {
			highest = h
		}
	}
	return highest
}

func scanNodeForNumberedParam(n Node) int {
	if n == nil {
		return 0
	}
	switch v := n.(type) {
	case *IdentNode:
		return numberedParamIndex(v.Val)
	case *MethodCall:
		h := scanNodeForNumberedParam(v.Receiver)
		for _, a := range v.Args {
			if hh := scanNodeForNumberedParam(a); hh > h {
				h = hh
			}
		}
		// Don't descend into nested blocks — they have their own numbered param scope
		return h
	case *InfixExpressionNode:
		h := scanNodeForNumberedParam(v.Left)
		if hh := scanNodeForNumberedParam(v.Right); hh > h {
			h = hh
		}
		return h
	case *NotExpressionNode:
		return scanNodeForNumberedParam(v.Arg)
	case *AssignmentNode:
		h := findHighestNumberedParam(v.Left)
		if hh := findHighestNumberedParam(v.Right); hh > h {
			h = hh
		}
		return h
	case *ReturnNode:
		return findHighestNumberedParam([]Node(v.Val))
	case *ArrayNode:
		return findHighestNumberedParam([]Node(v.Args))
	case *HashNode:
		h := 0
		for _, p := range v.Pairs {
			if hh := scanNodeForNumberedParam(p); hh > h {
				h = hh
			}
		}
		return h
	case *KeyValuePair:
		h := scanNodeForNumberedParam(v.Key)
		if hh := scanNodeForNumberedParam(v.Value); hh > h {
			h = hh
		}
		return h
	case *BracketAccessNode:
		h := scanNodeForNumberedParam(v.Composite)
		for _, a := range v.Args {
			if hh := scanNodeForNumberedParam(a); hh > h {
				h = hh
			}
		}
		return h
	case *BracketAssignmentNode:
		h := scanNodeForNumberedParam(v.Composite)
		for _, a := range v.Args {
			if hh := scanNodeForNumberedParam(a); hh > h {
				h = hh
			}
		}
		return h
	case *StringNode:
		h := 0
		for _, nodes := range v.Interps {
			for _, node := range nodes {
				if hh := scanNodeForNumberedParam(node); hh > h {
					h = hh
				}
			}
		}
		return h
	case *Condition:
		h := scanNodeForNumberedParam(v.Condition)
		if hh := findHighestNumberedParam([]Node(v.True)); hh > h {
			h = hh
		}
		if v.False != nil {
			if hh := scanNodeForNumberedParam(v.False); hh > h {
				h = hh
			}
		}
		return h
	case *CaseNode:
		h := scanNodeForNumberedParam(v.Value)
		for _, w := range v.Whens {
			if hh := scanNodeForNumberedParam(w); hh > h {
				h = hh
			}
		}
		return h
	case *WhenNode:
		h := findHighestNumberedParam([]Node(v.Conditions))
		if hh := findHighestNumberedParam([]Node(v.Statements)); hh > h {
			h = hh
		}
		return h
	case *WhileNode:
		h := scanNodeForNumberedParam(v.Condition)
		if hh := findHighestNumberedParam([]Node(v.Body)); hh > h {
			h = hh
		}
		return h
	case *ForInNode:
		h := scanNodeForNumberedParam(v.In)
		if hh := findHighestNumberedParam([]Node(v.Body)); hh > h {
			h = hh
		}
		return h
	case *SplatNode:
		return scanNodeForNumberedParam(v.Arg)
	case *LambdaNode:
		// Don't descend into lambdas — separate scope
		return 0
	case *RangeNode:
		h := scanNodeForNumberedParam(v.Lower)
		if hh := scanNodeForNumberedParam(v.Upper); hh > h {
			h = hh
		}
		return h
	case ArgsNode:
		return findHighestNumberedParam([]Node(v))
	case Statements:
		return findHighestNumberedParam([]Node(v))
	case *BeginNode:
		h := findHighestNumberedParam([]Node(v.Body))
		for _, clause := range v.RescueClauses {
			if hh := findHighestNumberedParam([]Node(clause.Body)); hh > h {
				h = hh
			}
		}
		if hh := findHighestNumberedParam([]Node(v.EnsureBody)); hh > h {
			h = hh
		}
		return h
	case *NextNode:
		return scanNodeForNumberedParam(v.Val)
	case *BreakNode:
		return 0
	}
	// For all other node types (primitives, self, nil, etc.), no children to scan
	return 0
}

// CollectIdents returns a set of all identifier names used in a node tree.
func CollectIdents(nodes []Node) map[string]bool {
	result := make(map[string]bool)
	collectIdentsFromNodes(nodes, result)
	return result
}

func collectIdentsFromNodes(nodes []Node, result map[string]bool) {
	for _, n := range nodes {
		collectIdentsFromNode(n, result)
	}
}

func collectIdentsFromNode(n Node, result map[string]bool) {
	if n == nil {
		return
	}
	switch v := n.(type) {
	case *IdentNode:
		result[v.Val] = true
	case *MethodCall:
		collectIdentsFromNode(v.Receiver, result)
		collectIdentsFromNodes([]Node(v.Args), result)
		if v.Block != nil {
			collectIdentsFromNodes(v.Block.Body.Statements, result)
		}
	case *InfixExpressionNode:
		collectIdentsFromNode(v.Left, result)
		collectIdentsFromNode(v.Right, result)
	case *AssignmentNode:
		collectIdentsFromNodes(v.Left, result)
		collectIdentsFromNodes(v.Right, result)
	case *ReturnNode:
		collectIdentsFromNodes([]Node(v.Val), result)
	case *ArrayNode:
		collectIdentsFromNodes([]Node(v.Args), result)
	case *Condition:
		collectIdentsFromNode(v.Condition, result)
		collectIdentsFromNodes([]Node(v.True), result)
		if v.False != nil {
			collectIdentsFromNode(v.False, result)
		}
	case Statements:
		collectIdentsFromNodes([]Node(v), result)
	case ArgsNode:
		collectIdentsFromNodes([]Node(v), result)
	case *StringNode:
		for _, nodes := range v.Interps {
			collectIdentsFromNodes(nodes, result)
		}
	case *BracketAccessNode:
		collectIdentsFromNode(v.Composite, result)
		collectIdentsFromNodes([]Node(v.Args), result)
	case *NotExpressionNode:
		collectIdentsFromNode(v.Arg, result)
	}
}

func numberedParamIndex(name string) int {
	if len(name) == 2 && name[0] == '_' && name[1] >= '1' && name[1] <= '9' {
		return int(name[1] - '0')
	}
	return 0
}
