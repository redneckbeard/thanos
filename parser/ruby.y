%{
package parser

import "strings"

func setRoot(yylex yyLexer, nodes []Node) {
  p := yylex.(*Lexer).Root
  for _, n := range nodes {
    p.AddStatement(n)
  }
}

func root(yylex yyLexer) *Root {
  return yylex.(*Lexer).Root
}

// scopeAccessPrefix extracts the namespace prefix from a primary_value chain
// used in scoped constant assignment (e.g., Diff::LCS::Change = ...).
// Returns "::" separated names like "Diff::LCS".
func scopeAccessPrefix(node Node) string {
  switch n := node.(type) {
  case *ConstantNode:
    if n.Namespace != "" {
      return n.Namespace + "::" + n.Val
    }
    return n.Val
  case *ScopeAccessNode:
    prefix := scopeAccessPrefix(n.Receiver)
    if prefix != "" {
      return prefix + "::" + n.Constant
    }
    return n.Constant
  default:
    return ""
  }
}
%}

%nonassoc <str> LOWEST
%right <str> ASSIGN MODASSIGN MULASSIGN ADDASSIGN SUBASSIGN DIVASSIGN LSHIFTASSIGN RSHIFTASSIGN ORASSIGN
%right <str>  QMARK COLON
%nonassoc <str> DOT2 DOT3 
%left <str> LOGICALOR
%left <str> LOGICALAND
%nonassoc <str> SPACESHIP EQ NEQ MATCH NOTMATCH
%left <str> GT GTE LT LTE
%left <str> AND
%left <str> PIPE CARET
%left <str> LSHIFT RSHIFT
%left <str> PLUS MINUS
%left <str> ASTERISK SLASH MODULO
%right <str> UNARY_NUM
%right <str> POW
%right <str> BANG
%token <str> NIL SYMBOL STRING
%token <str> INT
%token <str> FLOAT
%token <str> RATIONAL IMAGINARY
%token <str> TRUE FALSE
%token <str> CLASS MODULE DEF END IF IF_MOD UNLESS UNLESS_MOD BEGIN RESCUE RESCUE_MOD THEN ELSE WHILE WHILE_MOD RETURN YIELD SELF CONSTANT 
%token <str> ENSURE ELSIF CASE WHEN UNTIL UNTIL_MOD FOR BREAK NEXT SUPER ALIAS DO DO_COND DO_BLOCK PRIVATE PROTECTED IN

%token <str> IVAR CVAR GVAR METHODIDENT IDENT COMMENT LABEL

%token <str> ANDDOT DOT LBRACE LBRACEBLOCK RBRACE NEWLINE COMMA DOUBLESPLAT
%token <str> STRINGBEG STRINGEND INTERPBEG INTERPEND STRINGBODY REGEXBEG REGEXEND REGEXPOPT RAWSTRINGBEG RAWSTRINGEND WORDSBEG RAWWORDSBEG XSTRINGBEG RAWXSTRINGBEG
%token <str> SEMICOLON LBRACKET LBRACKETSTART RBRACKET LPAREN LPARENSTART RPAREN HASHROCKET
%token <str> SCOPE LAMBDA LOOP


%type <str> fcall operation rparen op fname then term relop rbracket string_beg string_end string_contents string_interp regex_beg regex_end cpath singleton_cpath op_asgn superclass private do raw_string_beg class module comment call_op
%type <node> symbol numeric user_variable keyword_variable simple_numeric expr arg primary literal lhs var_ref var_lhs primary_value expr_value command_asgn command_rhs command command_call regexp expr_value_do block_command block_call 
%type <node> arg_rhs arg_value method_call stmt if_tail opt_else none rel_expr string raw_string mlhs_item mlhs_node 
%type <node_list> compstmt stmts root mlhs mlhs_basic mlhs_head mlhs_inner for_var
%type <args> args call_args opt_call_args paren_args opt_paren_args aref_args command_args mrhs mrhs_arg
%type <param> f_arg_item f_kw f_opt f_block_arg f_rest_arg f_kwrest
%type <params> f_arglist f_opt_paren_args f_args f_arg opt_block_param f_kwarg opt_args_tail args_tail f_optarg opt_f_block_arg f_marg_list
%type <body> bodystmt
%type <when> when
%type <whens> case_body cases
%type <blk> brace_body brace_block do_block
%type <meth> defn_head defs_head method_head
%type <kv> assoc
%type <kvs> assocs assoc_list
%type <rescue_clause> rescue_clause
%type <rescue_clauses> opt_rescue
%type <node_list> opt_ensure
%type <str_list> rescue_types
%type <in_clause> p_in_clause
%type <in_clauses> p_case_body p_cases
%type <node> p_pattern p_pattern_item
%type <node_list> p_pattern_list

%union{
 args ArgsNode
 blk *Block
 body *Body
 kv  *KeyValuePair
 kvs []*KeyValuePair
 meth *Method
 node Node
 node_list Statements
 param *Param
 params []*Param
 rescue_clause *RescueClause
 rescue_clauses []*RescueClause
 root *Root
 str_list []string
 regexp string
 when *WhenNode
 whens []*WhenNode
 in_clause *InClause
 in_clauses []*InClause
 str string
}


%%

main: 
  root
  {
    setRoot(yylex, $1)
  }
root: 
  stmts opt_terms
  {
    $$ = $1
  }

bodystmt:
  compstmt opt_rescue
  {
    if len($2) > 0 {
      beginNode := &BeginNode{Body: $1, RescueClauses: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
      $$ = &Body{Statements: []Node{beginNode}}
    } else {
      $$ = &Body{Statements: $1}
    }
  }

compstmt: stmts opt_terms
            {
              $$ = $1
            }

stmts:
  {
    $$ = []Node{}
  }
| stmt
  {
    switch root(yylex).State.Peek() {
    case InClassBody:
      root(yylex).currentClass.AddStatement($1) 
      $$ = []Node{}
    case InModuleBody:
      mod := root(yylex).moduleStack.Peek()
      mod.Statements = append(mod.Statements, $1) 
      $$ = []Node{}
    default:
      $$ = []Node{$1}
    }
  }
| stmts terms stmt
  {
    switch root(yylex).State.Peek() {
    case InClassBody:
      root(yylex).currentClass.AddStatement($3) 
      $$ = []Node{}
    case InModuleBody:
      mod := root(yylex).moduleStack.Peek()
      mod.Statements = append(mod.Statements, $3) 
      $$ = []Node{}
    default:
      $$ = append($1, $3)
    }
  }

stmt:
  ALIAS fname fname
  {
    $$ = &AliasNode{NewName: $2, OldName: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| ALIAS SYMBOL SYMBOL
  {
    $$ = &AliasNode{NewName: $2[1:], OldName: $3[1:], Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| stmt IF_MOD expr_value
  {
    $$ = &Condition{Condition: $3, True: Statements{$1}, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| stmt UNLESS_MOD expr_value
  {
    $$ = &Condition{Condition: &NotExpressionNode{Arg: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}, True: Statements{$1}, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| stmt WHILE_MOD expr_value
  {
    $$ = &WhileNode{Condition: $3, Body: Statements{$1}, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| stmt UNTIL_MOD expr_value
  {
    $$ = &WhileNode{Condition: &NotExpressionNode{Arg: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}, Body: Statements{$1}, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
//| stmt RESCUE_MOD stmt
| command_asgn
| mlhs ASSIGN command_call
  {
    $$ = &AssignmentNode{Left: $1, Right: []Node{$3}, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| lhs ASSIGN mrhs
  {
    $$ = &AssignmentNode{Left: []Node{$1}, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| mlhs ASSIGN mrhs_arg
  {
    $$ = &AssignmentNode{Left: $1, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| private
  {
    root(yylex).inPrivateMethods = true
    $$ = &NoopNode{}
  }
| private SYMBOL
  {
    // private :method_name — make specific method private (ignored for now)
    $$ = &NoopNode{}
  }
| private fname
  {
    // private :method_name or private def ... — ignored for now
    $$ = &NoopNode{}
  }
| expr

command_asgn: 
  lhs ASSIGN command_rhs
  {
   
    $$ = &AssignmentNode{Left: []Node{$1}, Right: []Node{$3}, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| var_lhs op_asgn command_rhs
  {
    operation := &InfixExpressionNode{Left: $1, Operator: strings.Trim($2, "="), Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    $$ = &AssignmentNode{Left: []Node{$1}, Right: []Node{operation}, OpAssignment: true, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| primary_value LBRACKET opt_call_args rbracket op_asgn command_rhs
  {
    access := &BracketAccessNode{Composite: $1, Args: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    operation := &InfixExpressionNode{Left: access, Operator: strings.Trim($5, "="), Right: $6, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    assignment := &BracketAssignmentNode{
      Composite: $1,
      Args: $3,
      Pos: Pos{lineNo: currentLineNo, file: currentFile},
    }
    $$ = &AssignmentNode{Left: []Node{assignment}, Right: []Node{operation}, OpAssignment: true, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| primary_value call_op IDENT op_asgn command_rhs
  {
    call := &MethodCall{Receiver: $1, MethodName: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    operation := &InfixExpressionNode{Left: call, Operator: strings.Trim($4, "="), Right: $5, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    assignment := &MethodCall{Receiver: $1, MethodName: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    $$ = &AssignmentNode{Left: []Node{assignment}, Right: []Node{operation}, OpAssignment: true, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| primary_value call_op CONSTANT op_asgn command_rhs
  {
     noop := &NoopNode{Pos{currentLineNo, currentFile}}
     root(yylex).AddError(NewParseError(&NoopNode{Pos{currentLineNo, currentFile}}, "Tried to modify constant '%s'. In Ruby this only warns, but thanos forbids it.", $3).Terminal())
     $$ = noop
  }
| primary_value SCOPE CONSTANT op_asgn command_rhs
  {
     noop := &NoopNode{Pos{currentLineNo, currentFile}}
     root(yylex).AddError(NewParseError(&NoopNode{Pos{currentLineNo, currentFile}}, "Tried to modify constant '%s'. In Ruby this only warns, but thanos forbids it.", $3).Terminal())
     $$ = noop
  }

command_rhs: 
  command_call %prec ASSIGN
//| command_call kRESCUE_MOD stmt
| command_asgn

expr: 
  command_call
| BANG command_call
  {
    $$ = &NotExpressionNode{Arg: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg

expr_value: expr

expr_value_do:
  {
    yylex.(*Lexer).cond.Push(true)
  }
  expr_value do
  {
    yylex.(*Lexer).cond.Pop()
    $$ = $2
  }

command_call: 
  command
| block_command

block_command: 
  block_call
| block_call DOT operation command_args
  {
    call := &MethodCall{Receiver: $1, MethodName: $3, Args: $4, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    root(yylex).AddCall(call)
    $$ = call
  }
//cmd_brace_block: tLBRACE_ARG

fcall: operation

command: 
  fcall command_args %prec LOWEST
  {
    call := &MethodCall{MethodName: $1, Args: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    root(yylex).AddCall(call)
    $$ = call
  }
| fcall command_args brace_block
  {
    call := &MethodCall{MethodName: $1, Args: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    call.SetBlock($3)
    root(yylex).AddCall(call)
    $$ = call
  }
| primary_value call_op operation command_args %prec LOWEST
//| primary_value call_op operation2 command_args cmd_brace_block
| SUPER command_args
  {
    $$ = &SuperNode{Args: $2, Method: root(yylex).currentMethod, Class: root(yylex).currentClass, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
	}
| YIELD command_args
  {
    root(yylex).currentMethod.AddParam(&Param{Name: "blk", Kind: ExplicitBlock})
    $$ = &MethodCall{Receiver: &IdentNode{Val: "blk"}, MethodName: "call", Args: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| RETURN call_args
  {
    r := &ReturnNode{Val: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    root(yylex).AddReturn(r)
    $$ = r
  }
//| kBREAK call_args
//| kNEXT call_args

mlhs: 
  mlhs_basic
| LPARENSTART mlhs_inner rparen
  {
    $$ = $2
  }

mlhs_inner: 
  mlhs_basic
| LPARENSTART mlhs_inner rparen
  {
    $$ = $2
  }

mlhs_basic: 
  mlhs_head
| mlhs_head mlhs_item
  {
		$$ = append($1, $2)
  }
| mlhs_head ASTERISK mlhs_node
  {
		$$ = append($1, &SplatNode{Arg: $3})
  }
//| mlhs_head tSTAR mlhs_node tCOMMA mlhs_post
//| mlhs_head tSTAR
//| mlhs_head tSTAR tCOMMA mlhs_post
//| tSTAR mlhs_node
//| tSTAR mlhs_node tCOMMA mlhs_post
//| tSTAR
//| tSTAR tCOMMA mlhs_post

mlhs_item: 
  mlhs_node
//| tLPAREN mlhs_inner rparen

mlhs_head: 
  mlhs_item COMMA
  {
		$$ = []Node{$1}
  }
| mlhs_head mlhs_item COMMA
  {
		$$ = append($1, $2)
  }
//mlhs_post: mlhs_item
//| mlhs_post tCOMMA mlhs_item

mlhs_node: user_variable
| keyword_variable
| primary_value LBRACKET opt_call_args rbracket
  {
		$$ = &BracketAssignmentNode{
      Composite: $1,
      Args: $3,
      Pos: Pos{lineNo: currentLineNo, file: currentFile},
    }
  }
| primary_value call_op IDENT
  {
    call := &MethodCall{Receiver: $1, MethodName: $3, Op: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    $$ = call
  }

lhs: 
  user_variable
| keyword_variable
| primary_value LBRACKET opt_call_args rbracket
  {
    $$ = &BracketAssignmentNode{
      Composite: $1,
      Args: $3,
      Pos: Pos{lineNo: currentLineNo, file: currentFile},
    }
  }
| primary_value call_op IDENT
  {
    call := &MethodCall{Receiver: $1, MethodName: $3, Op: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    $$ = call
  }
| primary_value SCOPE CONSTANT
  {
    $$ = &ConstantNode{Val: $3, Namespace: scopeAccessPrefix($1), Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }

class:
  CLASS
  {
    root(yylex).nextConstantType = CLASS
    $$ = $1
  } 
module:
  MODULE
  {
    root(yylex).nextConstantType = MODULE
    $$ = $1
  } 
cpath: //SCOPE Constant
  CONSTANT
  {
    r := root(yylex)
    if r.nextConstantType == MODULE {
      r.PushModule($1, currentLineNo)
    } else {
      r.PushClass($1, currentLineNo)
    }
    r.cpathDepth = 0
    $$ = $1
  }
| cpath SCOPE CONSTANT
  {
    // For class/module A::B::C, each :: segment converts the previous head to
    // an intermediate module and pushes the new segment as the target.
    r := root(yylex)
    if r.nextConstantType == CLASS {
      // Previous segment was a class — convert to intermediate module.
      r.ConvertClassToModule()
    }
    r.cpathDepth++
    // Push the new segment as the actual target type
    if r.nextConstantType == CLASS {
      r.PushClass($3, currentLineNo)
    } else {
      r.PushModule($3, currentLineNo)
    }
    $$ = $1 + "::" + $3
  }

singleton_cpath:
  CONSTANT
  {
    root(yylex).PushSingletonTarget($1)
    $$ = $1
  }
| singleton_cpath SCOPE CONSTANT
  {
    root(yylex).PushSingletonTarget($3)
    $$ = $1 + "::" + $3
  }

fname: 
  operation
  {
    $$ = $1
  }
| op 
// | reswords

//fsym: fname
//| symbol
//fitem: fsym
//| dsym

op:   PIPE    | CARET  | AND  | SPACESHIP  | EQ
  |   MATCH   | NOTMATCH | GT      | GTE  | LT     | LTE
  |   NEQ     | LSHIFT  | RSHIFT   | PLUS | MINUS 
  |   ASTERISK    | SLASH | MODULO | POW  | BANG 

arg: 
  lhs ASSIGN arg_rhs
  {
    $$ = &AssignmentNode{Left: []Node{$1}, Right: []Node{$3}, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| var_lhs op_asgn arg_rhs
  {
    operation := &InfixExpressionNode{Left: $1, Operator: strings.Trim($2, "="), Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    $$ = &AssignmentNode{Left: []Node{$1}, Right: []Node{operation}, OpAssignment: true, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| primary_value LBRACKET opt_call_args rbracket op_asgn arg_rhs
  {
    access := &BracketAccessNode{Composite: $1, Args: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    operation := &InfixExpressionNode{Left: access, Operator: strings.Trim($5, "="), Right: $6, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    assignment := &BracketAssignmentNode{
      Composite: $1,
      Args: $3,
      Pos: Pos{lineNo: currentLineNo, file: currentFile},
    }
    $$ = &AssignmentNode{Left: []Node{assignment}, Right: []Node{operation}, OpAssignment: true, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| primary_value call_op IDENT op_asgn arg_rhs
  {
    call := &MethodCall{Receiver: $1, MethodName: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    operation := &InfixExpressionNode{Left: call, Operator: strings.Trim($4, "="), Right: $5, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    assignment := &MethodCall{Receiver: $1, MethodName: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    $$ = &AssignmentNode{Left: []Node{assignment}, Right: []Node{operation}, OpAssignment: true, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| primary_value call_op CONSTANT op_asgn arg_rhs
  {
     noop := &NoopNode{Pos{currentLineNo, currentFile}}
     root(yylex).AddError(NewParseError(&NoopNode{Pos{currentLineNo, currentFile}}, "Tried to modify constant '%s'. In Ruby this only warns, but thanos forbids it.", $3).Terminal())
     $$ = noop
  }
| primary_value SCOPE CONSTANT op_asgn arg_rhs
  {
     noop := &NoopNode{Pos{currentLineNo, currentFile}}
     root(yylex).AddError(NewParseError(&NoopNode{Pos{currentLineNo, currentFile}}, "Tried to modify constant '%s'. In Ruby this only warns, but thanos forbids it.", $3).Terminal())
     $$ = noop
  }
//| tCOLON3 tCONSTANT tOP_ASGN arg_rhs
| arg DOT2 arg
  {
    $$ = &RangeNode{Lower: $1, Upper: $3, Inclusive: true, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg DOT3 arg
  {
    $$ = &RangeNode{Lower: $1, Upper: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg DOT2
  {
    $$ = &RangeNode{Lower: $1, Inclusive: true, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg DOT3
  {
    $$ = &RangeNode{Lower: $1, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg PLUS arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg MINUS arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg ASTERISK arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg SLASH arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg MODULO arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg POW arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
//| tUNARY_NUM simple_numeric tPOW arg
//| tUPLUS arg
//| tUMINUS arg
| arg PIPE arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg CARET arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg AND arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg SPACESHIP arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| rel_expr %prec SPACESHIP
| arg EQ arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg NEQ arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg MATCH arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg NOTMATCH arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| BANG arg
  {
    $$ = &NotExpressionNode{Arg: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg LSHIFT arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg RSHIFT arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg LOGICALAND arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg LOGICALOR arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| arg QMARK arg opt_nl COLON arg
  {
    $$ = &Condition{
       Condition: $1,
       True: Statements{$3},
       False: &Condition{
         True: Statements{$6},
         elseBranch: true,
       },
       Pos: Pos{lineNo: currentLineNo, file: currentFile},
    }
  }
| defn_head f_opt_paren_args ASSIGN arg
  {
    for _, p := range $2 {
      if err := $1.AddParam(p); err != nil {
        root(yylex).AddError(err)
      }
    }
    $1.Body = &Body{Statements: []Node{$4}}
    root(yylex).AddMethod($1)
    $$ = $1
  }
| defs_head f_opt_paren_args ASSIGN arg
  {
    for _, p := range $2 {
      if err := $1.AddParam(p); err != nil {
        root(yylex).AddError(err)
      }
    }
    $1.Body = &Body{Statements: []Node{$4}}
    root(yylex).AddMethod($1)
    $$ = $1
  }
| primary

relop: GT | LT | GTE | LTE

rel_expr: 
  arg relop arg %prec GT
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| rel_expr relop arg %prec GT
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }

arg_value: arg

aref_args:  // none here in Ruby grammar
  { 
  $$ = nil 
  }
| args trailer
  { 
    $$ = $1 
  }
//| args tCOMMA assocs trailer
//| assocs trailer

arg_rhs: arg %prec ASSIGN
//| arg kRESCUE_MOD arg


paren_args: 
  LPAREN opt_call_args rparen
  {
    $$ = $2
  }

opt_paren_args: 
  {
    $$ = ArgsNode{}
  }
| paren_args

opt_call_args:
  {
    $$ = ArgsNode{}
  }
| call_args
| args COMMA
//| args tCOMMA assocs tCOMMA
//| assocs tCOMMA
| AND SYMBOL
  {
    $$ = ArgsNode{&SymbolToProcNode{MethodName: strings.TrimPrefix($2, ":"), Pos: Pos{lineNo: currentLineNo, file: currentFile}}}
  }
| args COMMA AND SYMBOL
  {
    $$ = append($1, &SymbolToProcNode{MethodName: strings.TrimPrefix($4, ":"), Pos: Pos{lineNo: currentLineNo, file: currentFile}})
  }
| AND IDENT
  {
    $$ = ArgsNode{&BlockPassNode{Name: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}}
  }
| args COMMA AND IDENT
  {
    $$ = append($1, &BlockPassNode{Name: $4, Pos: Pos{lineNo: currentLineNo, file: currentFile}})
  }


call_args:
//command |
  args //opt_block_arg
  {
    $$ = $1
  }
| assocs //opt_block_arg
  {
    for _, kv := range $1 {
      $$ = append($$, kv)
    }
  }
| args COMMA assocs //opt_block_arg
  {
    for _, kv := range $3 {
      $1 = append($1, kv)
    }
    $$ = $1
  }
//| block_arg

args: 
  arg_value
  {
    $$ = ArgsNode{$1}
  }
| args COMMA arg_value
  {
    $$ = append($1, $3)
  }
| ASTERISK arg_value
  {
    $$ = ArgsNode{&SplatNode{Arg: $2}}
  }
| args COMMA ASTERISK arg_value
  {
    $$ = append($1, &SplatNode{Arg: $4})
  }

mrhs_arg: 
  mrhs
| arg_value
  {
    $$ = []Node{$1}
  }


command_args:
  {
    if yyrcvr.Lookahead() == LBRACKETSTART || yyrcvr.Lookahead() == LPARENSTART {
      top := yylex.(*Lexer).cmdArg.Pop()
      yylex.(*Lexer).cmdArg.Push(true)
      yylex.(*Lexer).cmdArg.Push(top)
    } else {
      yylex.(*Lexer).cmdArg.Push(true)
    }
  }
  call_args //Ruby implementation includes some lookahead here
  {
    yylex.(*Lexer).cmdArg.Pop()
    yylex.(*Lexer).cmdArg.Pop()
    $$ = $2
  }
//block_arg: tAMPER arg_value
//opt_block_arg: tCOMMA block_arg
//| # nothing
mrhs: 
  args COMMA arg_value
  {
		$$ = append($1, $3)
  }
| args COMMA ASTERISK arg_value
  {
		$$ = append($1, &SplatNode{Arg: $4})
  }
| ASTERISK arg_value
  {
		$$ = ArgsNode{$2}
  }

primary: 
  literal
| string
| raw_string
| regexp 
  { 
    $$ = $1 
  }
// | qwords
// | symbols
// | qsymbols
| var_ref
 {
   $$ = $1
 }
// | tFID
| BEGIN compstmt opt_rescue opt_ensure END
  {
    $$ = &BeginNode{Body: $2, RescueClauses: $3, EnsureBody: $4, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| LPARENSTART stmt rparen
  {
    $$ = $2
  }
// | tLPAREN_ARG // includes some sort of lexer manipulation
// | tLPAREN compstmt tRPAREN
| primary_value SCOPE CONSTANT
  {
    $$ = &ScopeAccessNode{Receiver: $1, Constant: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
// | tCOLON3 tCONSTANT
| LBRACKETSTART aref_args rbracket
  {
    $$ = &ArrayNode{Args: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| LBRACE assoc_list RBRACE
  {
    $$ = &HashNode{Pairs: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| YIELD LPAREN call_args rparen
  {
    // this is naive, as in theory the source could have non-block locals called "blk".
    root(yylex).currentMethod.AddParam(&Param{Name: "blk", Kind: ExplicitBlock})
    $$ = &MethodCall{Receiver: &IdentNode{Val: "blk"}, MethodName: "call", Args: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| YIELD LPAREN rparen
  {
    root(yylex).currentMethod.AddParam(&Param{Name: "blk", Kind: ExplicitBlock})
    $$ = &MethodCall{Receiver: &IdentNode{Val: "blk"}, MethodName: "call", Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| YIELD
  {
    root(yylex).currentMethod.AddParam(&Param{Name: "blk", Kind: ExplicitBlock})
    $$ = &MethodCall{Receiver: &IdentNode{Val: "blk"}, MethodName: "call", Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| fcall brace_block
  {
  	call := &MethodCall{MethodName: $1, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    call.SetBlock($2)
    $$ = call
   }
| METHODIDENT
  {
    // Bare predicate/bang method call with no args: get?, empty?, save!
    call := &MethodCall{MethodName: $1, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    // block_given? implies an optional block param on the current method
    if $1 == "block_given?" && root(yylex).currentMethod != nil {
      root(yylex).currentMethod.AddParam(&Param{Name: "blk", Kind: ExplicitBlock})
    }
    if root(yylex).currentClass != nil {
      root(yylex).currentClass.MethodSet.AddCall(call)
    } else {
      root(yylex).AddCall(call)
    }
    $$ = call
  }
| method_call
| method_call brace_block
 {
   call := $1.(*MethodCall)
   call.SetBlock($2)
   if yylex.(*Lexer).gauntlet && call.MethodName == "gauntlet" {
     lines := strings.Split(yylex.(*Lexer).lastParsedToken.RawBlock, "\n")
     call.RawBlock = strings.Join(lines[1:len(lines)-1], "\n")
     yylex.(*Lexer).gauntlet = false
   }
   $$ = call
 }
| LAMBDA LPARENSTART f_args RPAREN LBRACEBLOCK compstmt RBRACE
  {
    blk := &Block{Body: &Body{Statements: $6}, ParamList: NewParamList()}
    for _, p := range $3 {
      blk.AddParam(p)
    }
    $$ = &LambdaNode{Block: blk, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| LAMBDA LBRACEBLOCK compstmt RBRACE
  {
    blk := &Block{Body: &Body{Statements: $3}, ParamList: NewParamList()}
    $$ = &LambdaNode{Block: blk, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| LAMBDA LPARENSTART f_args RPAREN DO compstmt END
  {
    blk := &Block{Body: &Body{Statements: $6}, ParamList: NewParamList()}
    for _, p := range $3 {
      blk.AddParam(p)
    }
    $$ = &LambdaNode{Block: blk, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| IF expr_value then compstmt if_tail END
  {
    $$ = &Condition{Condition: $2, True: $4, False: $5, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| UNLESS expr_value then compstmt opt_else END
  {
    $$ = &Condition{Condition: &NotExpressionNode{Arg: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}, True: $4, False: $5, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| WHILE expr_value_do compstmt END
  {
    $$ = &WhileNode{Condition: $2, Body: $3, Pos: Pos{lineNo: $2.LineNo(), file: currentFile}}
  }
| UNTIL expr_value_do compstmt END
  {
    $$ = &WhileNode{Condition: &NotExpressionNode{Arg: $2, Pos: Pos{lineNo: $2.LineNo(), file: currentFile}}, Body: $3, Pos: Pos{lineNo: $2.LineNo(), file: currentFile}}
  }
| LOOP DO compstmt END
  {
    $$ = &WhileNode{Condition: &BooleanNode{Val: "true", Pos: Pos{lineNo: currentLineNo, file: currentFile}}, Body: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| LOOP LBRACEBLOCK compstmt RBRACE
  {
    $$ = &WhileNode{Condition: &BooleanNode{Val: "true", Pos: Pos{lineNo: currentLineNo, file: currentFile}}, Body: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| CASE expr_value opt_terms case_body END
  {
    $$ = &CaseNode{Value: $2, Whens: $4, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| CASE opt_terms case_body END
  {
    $$ = &CaseNode{Whens: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| CASE expr_value opt_terms p_case_body END
  {
    pm := &PatternMatchNode{Value: $2, InClauses: $4, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    // Check if last clause is an else (no pattern)
    if last := $4[len($4)-1]; last.Pattern == nil {
      pm.ElseBody = last.Statements
      pm.InClauses = $4[:len($4)-1]
    }
    $$ = pm
  }
| FOR for_var IN expr_value_do compstmt END
  {
    $$ = &ForInNode{For: $2, In: $4, Body: $5, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| class cpath superclass bodystmt END
  {
    r := root(yylex)
    r.currentClass.Superclass = $3
    $$ = r.PopClass()
    // Pop intermediate modules from :: chains (e.g., Diff in class Diff::Change)
    for i := 0; i < r.cpathDepth; i++ {
      r.PopIntermediateModule()
    }
    r.cpathDepth = 0
  }
| CLASS LSHIFT SELF term
  {
    root(yylex).inSingletonClass = true
  }
  bodystmt END
  {
    root(yylex).inSingletonClass = false
    $$ = &NoopNode{}
  }
| CLASS LSHIFT singleton_cpath term
  {
    root(yylex).inSingletonClass = true
  }
  bodystmt END
  {
    r := root(yylex)
    r.inSingletonClass = false
    r.PopSingletonTarget()
    $$ = &NoopNode{}
  }
| module cpath bodystmt END
  {
    r := root(yylex)
    module := r.PopModule()
    if parent := r.moduleStack.Peek(); parent != nil {
      parent.Modules = append(parent.Modules, module)
    } else {
      r.TopLevelModules = append(r.TopLevelModules, module)
    }
    // Pop intermediate modules from :: chains (e.g., Diff and LCS in Diff::LCS::Internals)
    for i := 0; i < r.cpathDepth; i++ {
      r.PopIntermediateModule()
    }
    r.cpathDepth = 0
    $$ = module
  }
| method_head bodystmt END
  {
    $1.Body = $2
    root(yylex).AddMethod($1)
    root(yylex).State.Pop()
    $$ = $1
  }
//| k_def singleton dot_or_colon
| BREAK
  {
    $$ = &BreakNode{Pos: Pos{lineNo: currentLineNo, file: currentFile}}  
  }
| NEXT
  {
    $$ = &NextNode{Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| NEXT LPARENSTART args rparen
  {
    if len($3) == 1 {
      $$ = &NextNode{Val: $3[0], Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    } else {
      $$ = &NextNode{Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    }
  }
//| kREDO
//| kRETRY

primary_value: primary

// k_class: kCLASS
// k_module: kMODULE
// k_def: kDEF
// k_return: kRETURN

then: 
  term
| THEN
| term THEN
  {
    $$ = $2
  }

opt_rescue:
  {
    $$ = []*RescueClause{}
  }
| opt_rescue rescue_clause
  {
    $$ = append($1, $2)
  }

rescue_clause:
  RESCUE HASHROCKET IDENT then compstmt
  {
    $$ = &RescueClause{ExceptionVar: $3, Body: $5, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| RESCUE rescue_types HASHROCKET IDENT then compstmt
  {
    $$ = &RescueClause{ExceptionTypes: $2, ExceptionVar: $4, Body: $6, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| RESCUE rescue_types then compstmt
  {
    $$ = &RescueClause{ExceptionTypes: $2, Body: $4, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| RESCUE then compstmt
  {
    $$ = &RescueClause{Body: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }

rescue_types:
  CONSTANT
  {
    $$ = []string{$1}
  }
| rescue_types COMMA CONSTANT
  {
    $$ = append($1, $3)
  }

opt_ensure:
  {
    $$ = nil
  }
| ENSURE compstmt
  {
    $$ = $2
  }

do:
  term
| DO_COND

if_tail: 
  opt_else
| ELSIF expr then compstmt if_tail
  {
    $$ = &Condition{Condition: $2, True: $4, False: $5, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }

opt_else: 
  none
| ELSE compstmt
  {
    $$ = &Condition{True: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}, elseBranch: true}
  }
        
for_var: 
  lhs
  {
    $$ = []Node{$1}
  }
| mlhs
//f_marg: f_norm_arg
//| tLPAREN f_margs rparen
//f_marg_list: f_marg
//| f_marg_list tCOMMA f_marg
//f_margs: f_marg_list
//| f_marg_list tCOMMA tSTAR f_norm_arg
//| f_marg_list tCOMMA tSTAR f_norm_arg tCOMMA f_marg_list
//| f_marg_list tCOMMA tSTAR
//| f_marg_list tCOMMA tSTAR            tCOMMA f_marg_list
//|                    tSTAR f_norm_arg
//|                    tSTAR f_norm_arg tCOMMA f_marg_list
//|                    tSTAR
//|                    tSTAR tCOMMA f_marg_list
//block_args_tail: f_block_kwarg tCOMMA f_kwrest opt_f_block_arg
//| f_block_kwarg opt_f_block_arg
//| f_kwrest opt_f_block_arg
//| f_block_arg
//opt_block_args_tail:
//| # nothing
//block_param: f_arg tCOMMA f_block_optarg tCOMMA f_rest_arg              opt_block_args_tail
//| f_arg tCOMMA f_block_optarg tCOMMA f_rest_arg tCOMMA f_arg opt_block_args_tail
//| f_arg tCOMMA f_block_optarg                                opt_block_args_tail
//| f_arg tCOMMA f_block_optarg tCOMMA                   f_arg opt_block_args_tail
//| f_arg tCOMMA                       f_rest_arg              opt_block_args_tail
//| f_arg tCOMMA
//| f_arg tCOMMA                       f_rest_arg tCOMMA f_arg opt_block_args_tail
//| f_arg                                                      opt_block_args_tail
//| f_block_optarg tCOMMA              f_rest_arg              opt_block_args_tail
//| f_block_optarg tCOMMA              f_rest_arg tCOMMA f_arg opt_block_args_tail
//| f_block_optarg                                             opt_block_args_tail
//| f_block_optarg tCOMMA                                f_arg opt_block_args_tail
//|                                    f_rest_arg              opt_block_args_tail
//|                                    f_rest_arg tCOMMA f_arg opt_block_args_tail
//|                                                                block_args_tail

opt_block_param: 
  {
    $$ = []*Param{}
  }
| PIPE f_arg PIPE // block_param_def here in ruby
  {
    $$ = $2
  }

//block_param_def: tPIPE opt_bv_decl tPIPE
//| tOROP
//| tPIPE block_param opt_bv_decl tPIPE
//opt_bv_decl: opt_nl
//| opt_nl tSEMI bv_decls opt_nl
//bv_decls: bvar
//| bv_decls tCOMMA bvar
//bvar: tIDENTIFIER
//| f_bad_arg
//lambda:   {
//f_larglist: tLPAREN2 f_args opt_bv_decl tRPAREN
//| f_args
//lambda_body: tLAMBEG
//| kDO_LAMBDA
do_block: 
  DO_BLOCK brace_body END
  {
    $$ = $2
  }

block_call: 
  command do_block
  {
    call := $1.(*MethodCall)
    call.SetBlock($2)
    if yylex.(*Lexer).gauntlet && call.MethodName == "gauntlet" {
      lines := strings.Split(yylex.(*Lexer).lastParsedToken.RawBlock, "\n")
      call.RawBlock = strings.Join(lines[1:len(lines)-1], "\n")
      yylex.(*Lexer).gauntlet = false
    }
    $$ = call
  }
| block_call DOT operation opt_paren_args
  {
    call := &MethodCall{Receiver: $1, MethodName: $3, Args: $4, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    root(yylex).AddCall(call)
    $$ = call
  }
| block_call DOT operation opt_paren_args brace_block
  {
    call := &MethodCall{Receiver: $1, MethodName: $3, Args: $4, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    call.SetBlock($5)
    root(yylex).AddCall(call)
    $$ = call
  }
| block_call DOT operation command_args do_block
  {
    call := &MethodCall{Receiver: $1, MethodName: $3, Args: $4, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    call.SetBlock($5)
    root(yylex).AddCall(call)
    $$ = call
  }

method_call: 
  fcall paren_args
  {
    call := &MethodCall{MethodName: $1, Args: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    if root(yylex).currentClass != nil {
      root(yylex).currentClass.MethodSet.AddCall(call)
    } else {
      root(yylex).AddCall(call)
    }
    $$ = call
  }
| primary_value call_op fname opt_paren_args
  {
    call := &MethodCall{Receiver: $1, MethodName: $3, Args: $4, Op: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    root(yylex).AddCall(call)
    $$ = call
  }
| SUPER paren_args
  {
    $$ = &SuperNode{Args: $2, Method: root(yylex).currentMethod, Class: root(yylex).currentClass, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
	}
| SUPER
  {
    $$ = &SuperNode{Method: root(yylex).currentMethod, Class: root(yylex).currentClass, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
	}
| primary_value LBRACKET opt_call_args rbracket
  {
    $$ = &BracketAccessNode{Composite: $1, Args: $3, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }

brace_block: 
  LBRACEBLOCK brace_body RBRACE
  {
    $$ = $2
  }
  // these shouldn't be the same; beyond precedence difference there are different things allowed in the body, but for now those aren't supported anyway so...
| DO brace_body END // should be do_body
  { 
    $$ = $2
  }

brace_body: 
  opt_block_param compstmt
  {
    blk := &Block{Body: &Body{Statements: $2}, ParamList: NewParamList()}
    for _, p := range $1 {
      blk.AddParam(p)
    }
    synthesizeNumberedParams(blk)
    $$ = blk
  }

// do_body:   
case_body: 
  when cases
  {
    $$ = append([]*WhenNode{$1}, $2...)
  }

when: 
  WHEN args then compstmt
  {
    $$ = &WhenNode{Conditions: $2, Statements: $4, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }

cases: 
  none
  {
    $$ = []*WhenNode{}
  }
| ELSE compstmt
  {
    $$ = []*WhenNode{{Statements: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}}
  }
| case_body

p_case_body:
  p_in_clause p_cases
  {
    $$ = append([]*InClause{$1}, $2...)
  }

p_in_clause:
  IN p_pattern then compstmt
  {
    $$ = &InClause{Pattern: $2, Statements: $4, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }

p_cases:
  none
  {
    $$ = []*InClause{}
  }
| ELSE compstmt
  {
    $$ = []*InClause{{Statements: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}}
  }
| p_case_body

p_pattern:
  LBRACKET RBRACKET
  {
    $$ = &ArrayPatternNode{Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| LBRACKETSTART RBRACKET
  {
    $$ = &ArrayPatternNode{Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| LBRACKET p_pattern_list RBRACKET
  {
    $$ = &ArrayPatternNode{Elements: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| LBRACKETSTART p_pattern_list RBRACKET
  {
    $$ = &ArrayPatternNode{Elements: $2, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| p_pattern_item

p_pattern_item:
  IDENT
  {
    if $1 == "_" {
      $$ = &WildcardPatternNode{Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    } else {
      $$ = &IdentNode{Val: $1, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    }
  }
| literal
| NIL
  {
    $$ = &NilNode{Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| TRUE
  {
    $$ = &BooleanNode{Val: "true", Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| FALSE
  {
    $$ = &BooleanNode{Val: "false", Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }

p_pattern_list:
  p_pattern
  {
    $$ = Statements{$1}
  }
| p_pattern_list COMMA p_pattern
  {
    $$ = append($1, $3)
  }

// opt_rescue: kRESCUE exc_list exc_var then compstmt opt_rescue
// |
// exc_list: arg_value
// | mrhs
// | none
// exc_var: tASSOC lhs
// | none
// opt_ensure: kENSURE compstmt
// | none

literal: 
  numeric
| symbol
//| dsym

string: 
  string_beg string_contents string_end
  {
    str := root(yylex).StringStack.Pop()
    str.delim = $3
		$$ = str
  }

raw_string: 
  raw_string_beg STRINGBODY RAWSTRINGEND
  {
    $$ = &StringNode{BodySegments: []string{$2}, Kind: getStringKind($1), Pos: Pos{lineNo: currentLineNo, file: currentFile}, delim: $3} 
  }

raw_string_beg:
  RAWSTRINGBEG
| RAWWORDSBEG
| RAWXSTRINGBEG

string_beg: 
  STRINGBEG
  {
    root(yylex).State.Push(InString)
    root(yylex).StringStack.Push(&StringNode{Kind: getStringKind($1), Interps: make(map[int][]Node), Pos: Pos{lineNo: currentLineNo, file: currentFile}})
    $$ = ""
  }
| WORDSBEG
  {
    root(yylex).State.Push(InString)
    root(yylex).StringStack.Push(&StringNode{Kind: getStringKind($1), Interps: make(map[int][]Node), Pos: Pos{lineNo: currentLineNo, file: currentFile}})
    $$ = ""
  }
| XSTRINGBEG
  {
    root(yylex).State.Push(InString)
    root(yylex).StringStack.Push(&StringNode{Kind: getStringKind($1), Interps: make(map[int][]Node), Pos: Pos{lineNo: currentLineNo, file: currentFile}})
    $$ = ""
  }

string_end: 
  STRINGEND
  {
    root(yylex).State.Pop()
    $$ = $1
  }

string_contents: 
  string_contents STRINGBODY 
  {
    curr := root(yylex).StringStack.Peek()
    curr.BodySegments = append(curr.BodySegments, $2)
    $$ = ""
  }
| string_contents string_interp
  {
    $$ = ""
  }
|
  {
    $$ = ""
  }

string_interp: 
  INTERPBEG primary INTERPEND
  {
    curr := root(yylex).StringStack.Peek() 
    curr.Interps[len(curr.BodySegments)] = append(curr.Interps[len(curr.BodySegments)], $2)
    $$ = ""
  }

regexp:
  regex_beg string_contents regex_end
	{
		regexp := root(yylex).StringStack.Pop()
    $$ = regexp
	}
| regex_beg string_contents regex_end REGEXPOPT
	{
		regexp := root(yylex).StringStack.Pop()
		regexp.Flags = $4
    $$ = regexp
	}

regex_beg: 
  REGEXBEG
	{
		root(yylex).State.Push(InString)
		root(yylex).StringStack.Push(&StringNode{Kind: Regexp, Interps: make(map[int][]Node), Pos: Pos{lineNo: currentLineNo, file: currentFile}})
		$$ = ""
	}

regex_end: 
  REGEXEND
	{
		root(yylex).State.Pop()
		$$ = ""
	}
//symbols: tSYMBOLS_BEG symbol_list tSTRING_END
//symbol_list: # nothing
//| symbol_list word tSPACE
//qwords: tQWORDS_BEG qword_list tSTRING_END
//qsymbols: tQSYMBOLS_BEG qsym_list tSTRING_END
//qword_list: # nothing
//| qword_list tSTRING_CONTENT tSPACE
//qsym_list: # nothing
//| qsym_list tSTRING_CONTENT tSPACE

defn_head:
  DEF fname
  {
    method := NewMethod($2, root(yylex))
    method.Private = root(yylex).inPrivateMethods
    if root(yylex).inSingletonClass {
      method.ClassMethod = true
    }
    method.Pos = Pos{lineNo: currentLineNo, file: currentFile}
    $$ = method
  }

defs_head:
  DEF SELF DOT fname
  {
    method := NewMethod($4, root(yylex))
    method.ClassMethod = true
    method.Pos = Pos{lineNo: currentLineNo, file: currentFile}
    $$ = method
  }

method_head:
  defn_head f_arglist
  {
    for _, p := range $2 {
      if err := $1.AddParam(p); err != nil {
        root(yylex).AddError(err)
      }
    }
    root(yylex).State.Push(InMethodDefinition)
    $$ = $1
    yylex.(*Lexer).resetExpr = true
  }
| defs_head f_arglist
  {
    for _, p := range $2 {
      if err := $1.AddParam(p); err != nil {
        root(yylex).AddError(err)
      }
    }
    root(yylex).State.Push(InMethodDefinition)
    $$ = $1
    yylex.(*Lexer).resetExpr = true
  }

f_opt_paren_args:
  {
    $$ = nil
  }
| LPAREN f_args rparen
  {
    $$ = $2
  }

symbol:
  SYMBOL
  {
    $$ = &SymbolNode{Val: $1, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
//dsym: tSYMBEG string_contents tSTRING_END


numeric: 
  simple_numeric
| UNARY_NUM simple_numeric %prec LOWEST
  {
    var negative Node
    switch x := $2.(type) {
    case *IntNode:
      x.Val = $1 + x.Val
      negative = x
    case *Float64Node:
      x.Val = $1 + x.Val
      negative = x
    }
    $$ = negative
  }

simple_numeric: 
  INT
  {
    $$ = &IntNode{Val: $1, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| FLOAT
  {
    $$ = &Float64Node{Val: $1, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| RATIONAL
  {
    $$ = &RationalNode{Val: $1, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| IMAGINARY
  {
    $$ = &ImaginaryNode{Val: $1, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }

user_variable: 
  IDENT
  {
    $$ = &IdentNode{Val: $1, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| IVAR
  {
    ivar := &IVarNode{Val: $1, Class: root(yylex).currentClass, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
    $$ = ivar
    cls := root(yylex).currentClass
    if cls != nil {
      cls.AddIVar(ivar.NormalizedVal(), &IVar{Name: ivar.NormalizedVal()})
    }
  }
| GVAR
  {
    $$ = &GVarNode{Val: $1, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| CONSTANT
  {
    $$ = &ConstantNode{Val: $1, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| CVAR
  {
    $$ = &CVarNode{Val: $1, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }

keyword_variable: 
  NIL
  {
    $$ = &NilNode{Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| SELF
  {
    $$ = &SelfNode{Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| TRUE
  {
    $$ = &BooleanNode{Val: $1, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }
| FALSE
  {
    $$ = &BooleanNode{Val: $1, Pos: Pos{lineNo: currentLineNo, file: currentFile}}
  }

var_ref: user_variable | keyword_variable
var_lhs: user_variable | keyword_variable
superclass: 
  {
    $$ = ""
  }
// The correct rule here would have expr_value instead of CONSTANT. This
// restricts the grammar to what's currently actually supported by the
// compiler.
| LT CONSTANT term 
  {
    $$ = $2
  }

f_arglist: 
  LPAREN f_args rparen
  {
    $$ = $2
  }
| f_args term
  {
    $$ = $1
  }
args_tail: 
  f_kwarg
| f_kwarg COMMA f_kwrest opt_f_block_arg
  {
    $$ = append(append($1, $3), $4...)
  }
| f_kwarg opt_f_block_arg
  {
    $$ = append($1, $2...)
  }
| f_kwrest opt_f_block_arg
  {
    $$ = append([]*Param{$1}, $2...)
  }
| f_block_arg
  {
    $$ = []*Param{$1}
  }

opt_args_tail: 
  COMMA args_tail
  {
    $$ = $2
  }
| 
  {
    $$ = []*Param{}
  }

f_args: 
  f_arg COMMA f_optarg opt_args_tail
  {
    $$ = append(append($1, $3...), $4...)
  }
| f_arg opt_args_tail
  {
    $$ = append($1, $2...)
  }
| f_optarg opt_args_tail
  {
    $$ = append($1, $2...)
  }
| args_tail
  {
    $$ = $1
  }
| 
  {
    $$ = []*Param{}
  }
| f_arg COMMA f_optarg COMMA f_rest_arg opt_args_tail
  {
    $$ = append(append(append($1, $3...), $5), $6...)
  }
| f_arg COMMA f_rest_arg opt_args_tail
  {
    $$ = append(append($1, $3), $4...)
  }
| f_optarg COMMA f_rest_arg opt_args_tail
  {
    $$ = append(append($1, $3), $4...)
  }
| f_rest_arg opt_args_tail
  {
    $$ = append([]*Param{$1}, $2...)
  }


//| f_arg tCOMMA f_optarg tCOMMA f_rest_arg tCOMMA f_arg opt_args_tail
//| f_arg tCOMMA f_optarg tCOMMA                   f_arg opt_args_tail
//| f_arg tCOMMA                 f_rest_arg tCOMMA f_arg opt_args_tail
//|              f_optarg tCOMMA f_rest_arg tCOMMA f_arg opt_args_tail
//|              f_optarg tCOMMA                   f_arg opt_args_tail
//|                              f_rest_arg tCOMMA f_arg opt_args_tail

//f_bad_arg: tCONSTANT
//| tIVAR
//| tGVAR
//| tCVAR
//f_norm_arg: f_bad_arg
//| tIDENTIFIER
//f_arg_asgn: f_norm_arg

f_marg_list:
  IDENT
  {
    $$ = []*Param{{Name: $1, Kind: Positional}}
  }
| f_marg_list COMMA IDENT
  {
    $$ = append($1, &Param{Name: $3, Kind: Positional})
  }

f_arg_item:
  IDENT
  {
    $$ = &Param{Name: $1, Kind: Positional}
  } // f_arg_asgn
| LPARENSTART f_marg_list rparen
  {
    $$ = &Param{Kind: Destructured, Nested: $2}
  }

f_arg: 
  f_arg_item
  {
    $$ = []*Param{$1}
  }
| f_arg COMMA f_arg_item
  {
    $$ = append($1, $3)
  }

f_kw: 
  LABEL arg_value
  {
    $$ = &Param{Name: strings.Trim($1, ":"), Default: $2, Kind: Keyword}  
  }
| LABEL
  {
    $$ = &Param{Name: strings.Trim($1, ":"), Kind: Keyword}  
  }
//f_block_kw: f_label primary_value
//| f_label
//f_block_kwarg: f_block_kw
//| f_block_kwarg tCOMMA f_block_kw
f_kwarg: 
  f_kw
  {
    $$ = []*Param{$1}
  }
| f_kwarg COMMA f_kw
  {
    $$ = append($1, $3)
  }
//kwrest_mark: tPOW | tDSTAR
f_kwrest: 
  DOUBLESPLAT IDENT
  {
    $$ = &Param{Name: $2, Kind: DoubleSplat}  
  }
//| kwrest_mark
f_opt: 
  IDENT ASSIGN arg_value
  {
    $$ = &Param{Name: $1, Default: $3, Kind: Named}  
  }
//f_block_opt: f_arg_asgn tEQL primary_value
//f_block_optarg: f_block_opt
//| f_block_optarg tCOMMA f_block_opt
f_optarg: 
  f_opt
  {
    $$ = []*Param{$1}
  }
| f_optarg COMMA f_opt
  {
    $$ = append($1, $3)
  }
f_rest_arg: 
	ASTERISK IDENT
  {
    $$ = &Param{Name: $2, Kind: Splat}  
  }
//| restarg_mark
f_block_arg:
  AND IDENT
  {
    $$ = &Param{Name: $2, Kind: ExplicitBlock}  
  }
opt_f_block_arg: 
  COMMA f_block_arg
  {
    $$ = []*Param{$2}
  }
| 
  {
    $$ = []*Param{}
  }
//singleton: var_ref
//| tLPAREN2 expr rparen

assoc_list: 
  {
    $$ = []*KeyValuePair{}
  }
| assocs trailer

assocs: 
  assoc
  {
    $$ = []*KeyValuePair{$1}
  }
| assocs COMMA assoc
  {
    $$ = append($1, $3)
  }

assoc: 
  arg_value HASHROCKET arg_value
  {
    $$ = &KeyValuePair{Key: $1, Value: $3}
  }
| LABEL arg_value
  {
    $$ = &KeyValuePair{Label: strings.TrimRight($1, ":"), Value: $2}
  }
| LABEL
  {
    // Value-omission hash shorthand: {action:} means {action: action}
    name := strings.TrimRight($1, ":")
    $$ = &KeyValuePair{Label: name, Value: &IdentNode{Val: name, Pos: Pos{lineNo: currentLineNo, file: currentFile}}}
  }
| DOUBLESPLAT arg_value
  {
    $$ = &KeyValuePair{Value: $2, DoubleSplat: true}
  }

operation: IDENT | CONSTANT | METHODIDENT

call_op: DOT | ANDDOT

opt_terms:  | terms
opt_nl:  | NEWLINE

rparen: 
  opt_nl RPAREN
  {
    $$ = $2
  }

rbracket: 
  opt_nl RBRACKET
  {
    $$ = $2
  }

trailer: | NEWLINE | COMMA
         
term: 
  SEMICOLON
| NEWLINE
| comment

comment:
  COMMENT
  {
    root(yylex).AddComment(Comment{Text: strings.TrimSpace($1), LineNo: currentLineNo})
    $$ = $1
  }

terms: 
  term
| terms NEWLINE
| terms comment

none: { $$ = nil }

op_asgn: MODASSIGN | MULASSIGN | ADDASSIGN | SUBASSIGN | DIVASSIGN | LSHIFTASSIGN | RSHIFTASSIGN | ORASSIGN

private:
  PRIVATE
| PROTECTED
