%{
package parser

import "strings"

func setRoot(yylex yyLexer, nodes []Node) {
  p := yylex.(*Lexer).Program
  for _, n := range nodes {
    p.AddStatement(n)
  }
}

func root(yylex yyLexer) *Program {
  return yylex.(*Lexer).Program
}
%}

%nonassoc <str> LOWEST
%right <str> ASSIGN MODASSIGN MULASSIGN ADDASSIGN SUBASSIGN DIVASSIGN
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
%token <str> TRUE FALSE
%token <str> CLASS MODULE DEF END IF UNLESS BEGIN RESCUE THEN ELSE WHILE RETURN YIELD SELF CONSTANT 
%token <str> ENSURE ELSIF CASE WHEN UNTIL FOR BREAK NEXT SUPER ALIAS DO PRIVATE PROTECTED

%token <str> IVAR CVAR GVAR METHODIDENT IDENT COMMENT LABEL

%token <str> DOT LBRACE RBRACE NEWLINE COMMA
%token <str> STRINGBEG STRINGEND INTERPBEG INTERPEND STRINGBODY REGEXBEG REGEXEND REGEXPOPT RAWSTRINGBEG RAWSTRINGEND
%token <str> SEMICOLON LBRACKET LBRACKETSTART RBRACKET LPAREN LPARENSTART RPAREN HASHROCKET
%token <str> SCOPE


%type <str> fcall operation rparen op fname then term relop rbracket string_beg string_end string_contents string_interp regex_beg regex_end cpath op_asgn superclass private
%type <node> symbol numeric user_variable keyword_variable simple_numeric expr arg primary literal lhs var_ref var_lhs primary_value expr_value command_asgn command_rhs command command_call regexp 
%type <node> arg_rhs arg_value method_call stmt if_tail opt_else none rel_expr string raw_string mlhs_item mlhs_node
%type <node_list> compstmt stmts program mlhs mlhs_basic mlhs_head 
%type <args> args call_args opt_call_args paren_args opt_paren_args aref_args command_args
%type <param> f_arg_item f_kw f_opt
%type <params> f_arglist f_args f_arg opt_block_param f_kwarg opt_args_tail args_tail f_optarg
%type <body> bodystmt
%type <when> when
%type <whens> case_body cases
%type <blk> brace_body brace_block
%type <meth> method_signature
%type <kv> assoc
%type <kvs> assocs assoc_list

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
 program *Program
 regexp string
 when *WhenNode
 whens []*WhenNode
 str string
}


%%

main: 
  program
  {
    setRoot(yylex, $1)
  }
program: 
  stmts opt_terms
  {
    $$ = $1
  }

bodystmt: 
  compstmt // opt_rescue opt_else opt_ensure
  {
    $$ = &Body{Statements: $1}
  }

compstmt: stmts opt_terms
            {
              $$ = $1
            }

stmts:
//| stmt_or_begin
//| stmts terms stmt_or_begin
//| error stmt
  {
    $$ = []Node{}
  }
| stmt
  {
    if root(yylex).CurrentState() == InClassBody {
      root(yylex).currentClass.AddStatement($1) 
      $$ = []Node{}
    }
    $$ = []Node{$1}
  }
| stmts terms stmt
  {
    if root(yylex).CurrentState() == InClassBody {
      root(yylex).currentClass.AddStatement($3) 
      $$ = []Node{}
    } else {
      $$ = append($1, $3)
    }
  }

stmt: 
//  kALIAS fitem
//| kALIAS tGVAR tGVAR
//| kALIAS tGVAR tBACK_REF
//| kALIAS tGVAR tNTH_REF
//| stmt kIF_MOD expr_value
//| stmt kUNLESS_MOD expr_value
//| stmt kWHILE_MOD expr_value
//| stmt kUNTIL_MOD expr_value
//| stmt kRESCUE_MOD stmt
  command_asgn
| mlhs ASSIGN command_call
  {
    $$ = &AssignmentNode{Left: $1, Right: $3, lineNo: currentLineNo}
  }
//| lhs tEQL mrhs
//| mlhs tEQL mrhs_arg
| private
  {
    root(yylex).inPrivateMethods = true
    $$ = &NoopNode{}
  }
| expr 

command_asgn: 
  lhs ASSIGN command_rhs
  {
   
    $$ = &AssignmentNode{Left: []Node{$1}, Right: $3, lineNo: currentLineNo}
  }
| var_lhs op_asgn command_rhs
  {
    operation := &InfixExpressionNode{Left: $1, Operator: strings.Trim($2, "="), Right: $3, lineNo: currentLineNo}
    $$ = &AssignmentNode{Left: []Node{$1}, Right: operation, OpAssignment: true, lineNo: currentLineNo}
  }
//| primary_value tLBRACK2 opt_call_args rbracket tOP_ASGN command_rhs
//| primary_value call_op tIDENTIFIER tOP_ASGN command_rhs
//| primary_value call_op tCONSTANT tOP_ASGN command_rhs
//| primary_value tCOLON2 tCONSTANT tOP_ASGN command_rhs
//| primary_value tCOLON2 tIDENTIFIER tOP_ASGN command_rhs
//| backref tOP_ASGN command_rhs

command_rhs: 
  command_call %prec ASSIGN
//| command_call kRESCUE_MOD stmt
| command_asgn

expr: 
  command_call
  //| tBANG command_call
| arg

expr_value: expr

//expr_value_do:

command_call: 
  command
//| block_command
//block_command: block_call
//| block_call dot_or_colon operation2 command_args
//cmd_brace_block: tLBRACE_ARG

fcall: operation

command: 
  fcall command_args %prec LOWEST
  {
    call := &MethodCall{MethodName: $1, Args: $2, lineNo: currentLineNo}
    root(yylex).AddCall(call)
    $$ = call
  }
//| fcall command_args cmd_brace_block
//| primary_value call_op operation2 command_args =LOWEST
//| primary_value call_op operation2 command_args cmd_brace_block
//| primary_value tCOLON2 operation2 command_args =LOWEST
//| primary_value tCOLON2 operation2 command_args cmd_brace_block
//| kSUPER command_args
//| kYIELD command_args
//| k_return call_args
| RETURN call_args
  {
    r := &ReturnNode{Val: $2, lineNo: currentLineNo}
    root(yylex).AddReturn(r)
    $$ = r
  }
//| kBREAK call_args
//| kNEXT call_args

mlhs: mlhs_basic
//| tLPAREN mlhs_inner rparen
//mlhs_inner: mlhs_basic
//| tLPAREN mlhs_inner rparen

mlhs_basic: 
  mlhs_head
| mlhs_head mlhs_item
  {
		$$ = append($1, $2)
  }
//| mlhs_head tSTAR mlhs_node
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
      lineNo: currentLineNo,
    }
  }
//| primary_value call_op tIDENTIFIER
//| primary_value tCOLON2 tIDENTIFIER
//| primary_value call_op tCONSTANT
//| primary_value tCOLON2 tCONSTANT
//| tCOLON3 tCONSTANT
//| backref

lhs: 
  user_variable
| keyword_variable
| primary_value LBRACKET opt_call_args rbracket
  {
    $$ = &BracketAssignmentNode{
      Composite: $1,
      Args: $3,
      lineNo: currentLineNo,
    }
  }
// | primary_value call_op tIDENTIFIER
// | primary_value tCOLON2 tIDENTIFIER
// | primary_value call_op tCONSTANT
// | primary_value tCOLON2 tCONSTANT
// | tCOLON3 tCONSTANT
// | backref

// cname: tIDENTIFIER
// | tCONSTANT
cpath: //SCOPE Constant
  CONSTANT
  {
    root(yylex).PushClass($1, currentLineNo)
    $$ = $1
  }
//| primary_value SCOPE cname

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

// reswords: k__LINE__ | k__FILE__ | k__ENCODING__ 
// | kALIAS    | kAND      | kBEGIN        | kBREAK  | kCASE
// | kCLASS    | kDEF      | kDEFINED      | kDO     | kELSE
// | kELSIF    | kEND      | kENSURE       | kFALSE  | kFOR
// | kIN       | kMODULE   | kNEXT         | kNIL    | kNOT
// | kOR       | kREDO     | kRESCUE       | kRETRY  | kRETURN
// | kSELF     | kSUPER    | kTHEN         | kTRUE   
// | kWHEN     | kYIELD    | kIF           | kUNLESS | kWHILE
// | kUNTIL

arg: 
  lhs ASSIGN arg_rhs
  {
    $$ = &AssignmentNode{Left: []Node{$1}, Right: $3, lineNo: currentLineNo}
  }
| var_lhs op_asgn arg_rhs
  {
    operation := &InfixExpressionNode{Left: $1, Operator: strings.Trim($2, "="), Right: $3, lineNo: currentLineNo}
    $$ = &AssignmentNode{Left: []Node{$1}, Right: operation, OpAssignment: true, lineNo: currentLineNo}
  }
//| primary_value call_op tIDENTIFIER tOP_ASGN arg_rhs
//| primary_value call_op tCONSTANT tOP_ASGN arg_rhs
//| primary_value tCOLON2 tIDENTIFIER tOP_ASGN arg_rhs
//| primary_value tCOLON2 tCONSTANT tOP_ASGN arg_rhs
//| tCOLON3 tCONSTANT tOP_ASGN arg_rhs
//| backref tOP_ASGN arg_rhs
| arg DOT2 arg
  {
    $$ = &RangeNode{Lower: $1, Upper: $3, Inclusive: true, lineNo: currentLineNo}
  }
| arg DOT3 arg
  {
    $$ = &RangeNode{Lower: $1, Upper: $3, lineNo: currentLineNo}
  }
| arg DOT2
  {
    $$ = &RangeNode{Lower: $1, Inclusive: true, lineNo: currentLineNo}
  }
| arg DOT3
  {
    $$ = &RangeNode{Lower: $1, lineNo: currentLineNo}
  }
| arg PLUS arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
| arg MINUS arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
| arg ASTERISK arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
| arg SLASH arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
| arg MODULO arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
| arg POW arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
//| tUNARY_NUM simple_numeric tPOW arg
//| tUPLUS arg
//| tUMINUS arg
| arg PIPE arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
| arg CARET arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
| arg AND arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
| arg SPACESHIP arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
| rel_expr %prec SPACESHIP
| arg EQ arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
| arg NEQ arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
| arg MATCH arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
| arg NOTMATCH arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
| BANG arg
  {
    $$ = &NotExpressionNode{Arg: $2, lineNo: currentLineNo}
  }
| arg LSHIFT arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
| arg RSHIFT arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
| arg LOGICALAND arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
| arg LOGICALOR arg
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
| arg QMARK arg opt_nl COLON arg
  {
    $$ = &Condition{
       Condition: $1,
       True: Statements{$3},
       False: &Condition{
         True: Statements{$6},
       },
       lineNo: currentLineNo,
    }
  }
| primary

relop: GT | LT | GTE | LTE

rel_expr: 
  arg relop arg %prec GT
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
  }
| rel_expr relop arg %prec GT
  {
    $$ = &InfixExpressionNode{Left: $1, Operator: $2, Right: $3, lineNo: currentLineNo}
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
  LPAREN opt_call_args RPAREN
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
//| tSTAR arg_value
//| args tCOMMA tSTAR arg_value

command_args: call_args //Ruby implementation includes some lookahead here
//block_arg: tAMPER arg_value
//opt_block_arg: tCOMMA block_arg
//| # nothing
//mrhs_arg: mrhs
//| arg_value
//mrhs: args tCOMMA arg_value
//| args tCOMMA tSTAR arg_value
//| tSTAR arg_value

primary: 
  literal
| string
| raw_string
// | xstring
| regexp 
  { 
    $$ = $1 
  }
// | words
// | qwords
// | symbols
// | qsymbols
| var_ref
 {
   $$ = $1
 }
// backref
// | tFID
// | kBEGIN
| LPARENSTART stmt rparen
  {
    $$ = $2
  }
// | tLPAREN_ARG // includes some sort of lexer manipulation
// | tLPAREN compstmt tRPAREN
| primary_value SCOPE CONSTANT
  {
    $$ = &ScopeAccessNode{Receiver: $1, Constant: $3, lineNo: currentLineNo}
  }
// | tCOLON3 tCONSTANT
| LBRACKETSTART aref_args RBRACKET
  {
    $$ = &ArrayNode{Args: $2, lineNo: currentLineNo}
  }
| LBRACE assoc_list RBRACE
  {
    $$ = &HashNode{Pairs: $2, lineNo: currentLineNo}
  }
//| kYIELD tLPAREN2 call_args rparen
//| kYIELD tLPAREN2 rparen
//| kYIELD
//| fcall brace_block
| method_call
| method_call brace_block
 {
   call := $1.(*MethodCall)
   call.Block = $2
   if yylex.(*Lexer).blockDepth == 0 {
     call.RawBlock = yylex.(*Lexer).lastParsedToken.RawBlock
   }
   $$ = call
 }
//| tLAMBDA
| IF expr_value then compstmt if_tail END
  {
    $$ = &Condition{Condition: $2, True: $4, False: $5, lineNo: currentLineNo}
  }
| UNLESS expr_value then compstmt opt_else END
  {
    $$ = &Condition{Condition: &NotExpressionNode{Arg: $2, lineNo: currentLineNo}, True: $4, False: $5, lineNo: currentLineNo}
  }
//| kWHILE expr_value_do compstmt kEND
//| kUNTIL expr_value_do compstmt kEND
| CASE expr_value opt_terms case_body END
  {
    $$ = &CaseNode{Value: $2, Whens: $4, lineNo: currentLineNo}
  }
| CASE opt_terms case_body END
  {
    $$ = &CaseNode{Whens: $3, lineNo: currentLineNo}
  }
//| kFOR for_var kIN expr_value_do compstmt kEND
| CLASS cpath superclass bodystmt END
  {
    root(yylex).currentClass.Superclass = $3
    $$ = root(yylex).PopClass()
  }
//| k_class tLSHFT expr term
//| k_module cpath
| method_signature bodystmt END
  {
    $1.Body = $2
    root(yylex).AddMethod($1)
    root(yylex).PopState()
    root(yylex).PopScope()
    $$ = $1
  }
//| k_def singleton dot_or_colon
//| kBREAK
//| kNEXT
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

//do: term
//| kDO_COND

if_tail: 
  opt_else
| ELSIF expr then compstmt if_tail
  {
    $$ = &Condition{Condition: $2, True: $4, False: $5, lineNo: currentLineNo}
  }

opt_else: 
  none
| ELSE compstmt
  {
    $$ = &Condition{True: $2, lineNo: currentLineNo}
  }
        
//for_var: lhs
//| mlhs
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
//do_block: kDO_BLOCK
//block_call: command do_block
//| block_call dot_or_colon operation2 opt_paren_args
//| block_call dot_or_colon operation2 opt_paren_args brace_block
//| block_call dot_or_colon operation2 command_args do_block

method_call: 
  fcall paren_args
  {
    call := &MethodCall{MethodName: $1, Args: $2, lineNo: currentLineNo}
    if root(yylex).currentClass != nil {
      root(yylex).currentClass.MethodSet.AddCall(call)
    } else {
      root(yylex).AddCall(call)
    }
    $$ = call
  }
| primary_value DOT fname opt_paren_args
  {
    call := &MethodCall{Receiver: $1, MethodName: $3, Args: $4, lineNo: currentLineNo}
    root(yylex).AddCall(call)
    $$ = call
  }
//| primary_value tCOLON2 operation2 paren_args
//| primary_value tCOLON2 operation3
//| primary_value call_op paren_args
//| primary_value tCOLON2 paren_args
//| kSUPER paren_args
//| kSUPER
| primary_value LBRACKET opt_call_args rbracket
  {
    $$ = &BracketAccessNode{Composite: $1, Args: $3, lineNo: currentLineNo}
  }
//| primary_value tLBRACK2 opt_call_args rbracket

brace_block: 
  LBRACE brace_body RBRACE
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
    $$ = &Block{Params: $1, Body: &Body{Statements: $2}}
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
    $$ = &WhenNode{Conditions: $2, Statements: $4, lineNo: currentLineNo}
  }

cases: 
  none
  {
    $$ = []*WhenNode{}
  }
| ELSE compstmt
  {
    $$ = []*WhenNode{&WhenNode{Statements: $2, lineNo: currentLineNo}}
  }
| case_body
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
    $$ = root(yylex).PopString()
  }

raw_string: 
  RAWSTRINGBEG STRINGBODY RAWSTRINGEND
  {
    $$ = &StringNode{BodySegments: []string{$2}, Kind: SingleQuote, lineNo: currentLineNo} 
  }

string_beg: 
  STRINGBEG
  {
    root(yylex).PushState(InString)
    root(yylex).PushString()
    $$ = ""
  }

string_end: 
  STRINGEND
  {
    root(yylex).PopState()
    $$ = ""
  }

string_contents: 
  string_contents STRINGBODY 
  {
    curr := root(yylex).CurrentString()
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
    curr := root(yylex).CurrentString() 
    curr.Interps[len(curr.BodySegments)] = append(curr.Interps[len(curr.BodySegments)], $2)
    $$ = ""
  }
//xstring: tXSTRING_BEG xstring_contents tSTRING_END
regexp: 
  regex_beg string_contents regex_end //REGEXPOPT
	{
		regexp := root(yylex).PopString()
    regexp.Kind = Regexp
    $$ = regexp
	}

regex_beg: 
  REGEXBEG
	{
		root(yylex).PushState(InString)
		root(yylex).PushString()
		$$ = ""
	}

regex_end: 
  REGEXEND
	{
		root(yylex).PopState()
		$$ = ""
	}
//words: tWORDS_BEG word_list tSTRING_END
//word_list: # nothing
//| word_list word tSPACE
//word: string_content
//| word string_content
//symbols: tSYMBOLS_BEG symbol_list tSTRING_END
//symbol_list: # nothing
//| symbol_list word tSPACE
//qwords: tQWORDS_BEG qword_list tSTRING_END
//qsymbols: tQSYMBOLS_BEG qsym_list tSTRING_END
//qword_list: # nothing
//| qword_list tSTRING_CONTENT tSPACE
//qsym_list: # nothing
//| qsym_list tSTRING_CONTENT tSPACE
//xstring_contents: # nothing
//| xstring_contents string_content

method_signature: 
  DEF fname f_arglist
  {
    method := NewMethod($2, root(yylex))
    method.Private = root(yylex).inPrivateMethods
    method.lineNo = currentLineNo

    for _, p := range $3 {
      method.AddParam(p)
    }

    root(yylex).PushState(InMethodDefinition)
    root(yylex).PushScope(method.Locals)
    $$ = method
  }

symbol: 
  SYMBOL
  {
    $$ = &SymbolNode{Val: $1, lineNo: currentLineNo}
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
    $$ = &IntNode{Val: $1, lineNo: currentLineNo}
  }
| FLOAT
  {
    $$ = &Float64Node{Val: $1, lineNo: currentLineNo}
  }
//| tRATIONAL
//| tIMAGINARY

user_variable: 
  IDENT
  {
    $$ = &IdentNode{Val: $1, lineNo: currentLineNo}
  }
| IVAR
  {
    ivar := &IVarNode{Val: $1, Class: root(yylex).currentClass, lineNo: currentLineNo}
    $$ = ivar
    cls := root(yylex).currentClass
    if cls != nil {
      cls.ivars[ivar.NormalizedVal()] = &IVar{Name: ivar.NormalizedVal()}
      cls.ivarOrder = append(cls.ivarOrder, ivar.NormalizedVal())
    }
  }
| GVAR
  {
    $$ = &GVarNode{Val: $1, lineNo: currentLineNo}
  }
| CONSTANT
  {
    $$ = &ConstantNode{Val: $1, lineNo: currentLineNo}
  }
| CVAR
  {
    $$ = &CVarNode{Val: $1, lineNo: currentLineNo}
  }

keyword_variable: 
  NIL
  {
    $$ = &NilNode{lineNo: currentLineNo}
  }
| SELF
  {
    $$ = &SelfNode{lineNo: currentLineNo}
  }
| TRUE
  {
    $$ = &BooleanNode{Val: $1, lineNo: currentLineNo}
  }
| FALSE
  {
    $$ = &BooleanNode{Val: $1, lineNo: currentLineNo}
  }

var_ref: user_variable | keyword_variable
var_lhs: user_variable | keyword_variable
//backref: tNTH_REF
//| tBACK_REF
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
//f_kwarg tCOMMA f_kwrest opt_f_block_arg
//| f_kwarg opt_f_block_arg
//| f_kwrest opt_f_block_arg
//| f_block_arg
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

//f_args: f_arg tCOMMA f_optarg tCOMMA f_rest_arg              opt_args_tail
//| f_arg tCOMMA f_optarg tCOMMA f_rest_arg tCOMMA f_arg opt_args_tail
//| f_arg tCOMMA f_optarg tCOMMA                   f_arg opt_args_tail
//| f_arg tCOMMA                 f_rest_arg              opt_args_tail
//| f_arg tCOMMA                 f_rest_arg tCOMMA f_arg opt_args_tail
//|              f_optarg tCOMMA f_rest_arg              opt_args_tail
//|              f_optarg tCOMMA f_rest_arg tCOMMA f_arg opt_args_tail
//|              f_optarg tCOMMA                   f_arg opt_args_tail
//|                              f_rest_arg              opt_args_tail
//|                              f_rest_arg tCOMMA f_arg opt_args_tail

//f_bad_arg: tCONSTANT
//| tIVAR
//| tGVAR
//| tCVAR
//f_norm_arg: f_bad_arg
//| tIDENTIFIER
//f_arg_asgn: f_norm_arg

f_arg_item: 
  IDENT 
  { 
    $$ = &Param{Name: $1, Kind: Positional} 
  } // f_arg_asgn
//| tLPAREN f_margs rparen

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
//f_kwrest: kwrest_mark tIDENTIFIER
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
//restarg_mark: tSTAR2 | tSTAR
//f_rest_arg: restarg_mark tIDENTIFIER
//| restarg_mark
//blkarg_mark: tAMPER2 | tAMPER
//f_block_arg: blkarg_mark tIDENTIFIER
//opt_f_block_arg: tCOMMA f_block_arg
//|
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
//| tSTRING_BEG string_contents tLABEL_END arg_value
//| tDSTAR arg_value

operation: IDENT | CONSTANT | METHODIDENT

//operation2: tIDENTIFIER | tCONSTANT | tFID | op
//operation3: tIDENTIFIER | tFID | op
//dot_or_colon: call_op | tCOLON2
//call_op: tDOT
//| tANDDOT

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
| COMMENT NEWLINE
{
  root(yylex).AddComment(Comment{Text: $1, LineNo: currentLineNo})
  $$ = $2
}
| NEWLINE COMMENT NEWLINE
{
  root(yylex).AddComment(Comment{Text: $2, LineNo: currentLineNo})
  $$ = $3
}

terms: 
  term
| terms NEWLINE

none: { $$ = nil }

op_asgn: MODASSIGN | MULASSIGN | ADDASSIGN | SUBASSIGN | DIVASSIGN

private:
  PRIVATE
| PROTECTED
