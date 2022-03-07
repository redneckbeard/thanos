package parser

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode"
)

var currentLineNo = 0

var Illegal int = 99999999

var validPuncts = map[string]int{
	"(":   LPAREN,
	")":   RPAREN,
	".":   DOT,
	"..":  DOT2,
	"...": DOT3,
	";":   SEMICOLON,
	"?":   QMARK,
	"[":   LBRACKET,
	"]":   RBRACKET,
	"{":   LBRACE,
	"}":   RBRACE,
	",":   COMMA,
	"!":   BANG,
	"!=":  NEQ,
	"!~":  NOTMATCH,
	"%":   MODULO,
	"%=":  MODASSIGN,
	"&&":  LOGICALAND,
	"&":   AND,
	"*":   ASTERISK,
	"**":  POW,
	"*=":  MULASSIGN,
	"+":   PLUS,
	"+=":  ADDASSIGN,
	"-":   MINUS,
	"-=":  SUBASSIGN,
	"/":   SLASH,
	"/=":  DIVASSIGN,
	":":   COLON,
	"::":  SCOPE,
	"<":   LT,
	"<<":  LSHIFT,
	"<<=": LSHIFTASSIGN,
	"<=":  LTE,
	"<=>": SPACESHIP,
	"=":   ASSIGN,
	"==":  EQ,
	"=>":  HASHROCKET,
	"=~":  MATCH,
	">":   GT,
	">=":  GTE,
	">>":  RSHIFT,
	">>=": RSHIFTASSIGN,
	"|":   PIPE,
	"||":  LOGICALOR,
}

var keywords = map[string]int{
	"alias":    ALIAS,
	"begin":    BEGIN,
	"break":    BREAK,
	"case":     CASE,
	"class":    CLASS,
	"def":      DEF,
	"do":       DO,
	"else":     ELSE,
	"elsif":    ELSIF,
	"end":      END,
	"ensure":   ENSURE,
	"false":    FALSE,
	"for":      FOR,
	"gauntlet": IDENT,
	"if":       IF,
	"module":   MODULE,
	"next":     NEXT,
	"nil":      NIL,
	// Neither private nor protected are keywords in Ruby but are rather methods.
	// However, they function effectively as keywords, and because of how Method
	// structs are being constructed here, it's much simpler to simply flip a bit
	// to change state on the class when we encounter either than to record the
	// order of method definition relative to the position of any seen
	// `private`/`protected` method calls.
	"private":   PRIVATE,
	"protected": PROTECTED,
	"rescue":    RESCUE,
	"return":    RETURN,
	"self":      SELF,
	"super":     SUPER,
	"then":      THEN,
	"true":      TRUE,
	"unless":    UNLESS,
	"until":     UNTIL,
	"when":      WHEN,
	"while":     WHILE,
	"yield":     YIELD,
}

type LexState int

const (
	Clear LexState = iota
	InSymbol
	InGlobal
	InFloat
	InNumber
	InCVar
	InIVar
	InPunct
	InInterpString
	InInterp
	InRawString
	InRegex
	OpenBrace
)

type Token struct {
	Type     int
	Literal  string
	LineNo   int
	RawBlock string
}

func (t Token) String() string {
	return fmt.Sprintf("%s[%q]", tokenNames[t.Type], t.Literal)
}

type Lexer struct {
	*bytes.Buffer
	Root              *Root
	lineNo            int
	stream            chan Token
	read              []rune
	stateStack        []LexState
	lastToken         int
	rawSource         []rune
	lastParsedToken   Token
	gauntlet          bool
	spaceConsumed     bool
	percentDelimStack []rune
}

func NewLexer(buf []byte) *Lexer {
	l := &Lexer{
		Buffer:        bytes.NewBuffer(buf),
		lineNo:        1,
		stream:        make(chan Token),
		Root:          NewRoot(),
		spaceConsumed: true,
	}
	go l.Tokenize()
	return l
}

func (l *Lexer) Lex(lval *yySymType) int {
	token := <-l.stream
	lval.str = token.Literal
	currentLineNo = token.LineNo
	l.lastParsedToken = token
	return token.Type
}

func (l *Lexer) Error(e string) {
	l.Root.AddError(errors.New(e))
}

func (l *Lexer) pushState(state LexState) {
	l.stateStack = append(l.stateStack, state)
}

func (l *Lexer) popState() {
	if len(l.stateStack) > 0 {
		l.stateStack = l.stateStack[:len(l.stateStack)-1]
	}
}

func (l *Lexer) currentState() LexState {
	if len(l.stateStack) == 0 {
		return Clear
	}
	return l.stateStack[len(l.stateStack)-1]
}

func (l *Lexer) Peek() (rune, error) {
	b, _, err := l.ReadRune()
	l.UnreadRune()
	return b, err
}

func (l *Lexer) Read() (rune, error) {
	b, _, err := l.ReadRune()
	return b, err
}

func (l *Lexer) Advance() (rune, rune, error) {
	chr, _ := l.Read()
	l.read = append(l.read, chr)
	next, err := l.Peek()
	return chr, next, err
}

func (l *Lexer) RewindBuffer() {
	l.read = l.read[:len(l.read)-1]
}

func (l *Lexer) ResetBuffer() {
	l.read = []rune{}
}

func (l *Lexer) Emit(t int) {
	if l.gauntlet {
		l.rawSource = append(l.rawSource, l.read...)
	} else {
		l.rawSource = []rune{}
	}
	tok := Token{t, string(l.read), l.lineNo, string(l.rawSource)}
	l.stream <- tok
	l.lastToken = t
	l.ResetBuffer()
	l.spaceConsumed = false
}

func (l *Lexer) AtExprStart() bool {
	midExprTokens := []int{
		NIL, SYMBOL, STRING, INT, FLOAT, TRUE, FALSE, DEF, END, SELF, CONSTANT,
		IVAR, CVAR, GVAR, METHODIDENT, IDENT,
		RBRACE, STRINGEND, RBRACKET, RPAREN,
	}

	for _, tok := range midExprTokens {
		if l.lastToken == tok {
			return false
		}
	}

	return true
}

func (l *Lexer) currentStringDelim() rune {
	return l.percentDelimStack[len(l.percentDelimStack)-1]
}

func (l *Lexer) pushStringDelim(delim rune) {
	switch delim {
	case '{':
		delim = '}'
	case '[':
		delim = ']'
	case '(':
		delim = ')'
	}
	l.percentDelimStack = append(l.percentDelimStack, delim)
}

func (l *Lexer) popStringDelim() {
	l.percentDelimStack = l.percentDelimStack[:len(l.percentDelimStack)-1]
}

func (l *Lexer) Tokenize() {
	chr, err := l.Peek()
	for err == nil {
		switch {
		case l.currentState() == InInterpString:
			err = l.lexString()
		case l.currentState() == InRegex:
			err = l.lexRegex()
		case unicode.IsSpace(chr):
			err = l.lexWhitespace()
		case unicode.IsPunct(chr) || unicode.IsSymbol(chr):
			err = l.lexPunct()
		case unicode.IsLetter(chr) || chr == '_':
			err = l.lexWord()
		case unicode.IsDigit(chr):
			err = l.lexNumber()
		default:
			panic("Unknown Unicode character category encountered")
		}
		chr, err = l.Peek()
	}
	close(l.stream)
}

func (l *Lexer) lexWhitespace() error {
	chr, _, err := l.Advance()
	if chr == '\n' {
		l.Emit(NEWLINE)
		l.lineNo++
	} else {
		if l.gauntlet {
			l.rawSource = append(l.rawSource, l.read...)
		}
		l.ResetBuffer()
	}
	l.spaceConsumed = true
	return err
}

func (l *Lexer) lexNumber() error {
	_, next, err := l.Advance()
	for err == nil && unicode.IsDigit(next) {
		_, next, err = l.Advance()
	}
	// If we've already seen a decimal point, we've come back. Any further
	// punctuation belongs to something else.
	if l.currentState() == InFloat {
		l.Emit(FLOAT)
		return err
	}
	next, err = l.Peek()
	if err != nil && err != io.EOF {
		return err
	}
	// If we see a decimal point, we hand over lexing to lexPunct. If we're looking
	// at a float, we'll end up back above in the correct state; if not, we'll emit
	// the int from there.
	if next == '.' {
		l.pushState(InNumber)
		defer l.popState()
		return l.lexPunct()
	}

	l.Emit(INT)
	return err
}

func (l *Lexer) lexPunct() error {
	curr, next, err := l.Advance()

	if curr == '.' && l.currentState() == InNumber {
		if unicode.IsDigit(next) {
			l.pushState(InFloat)
			defer l.popState()
			return l.lexNumber()
		} else {
			l.RewindBuffer()
			l.Emit(INT)
			l.popState()
			l.read = []rune{'.'}
		}
	}

	// Defer to other lexing methods when the token is prefixed with non-alpha but
	if unicode.IsLetter(next) {
		switch curr {
		case ':':
			return l.lexSymbol()
		case '$':
			return l.lexGlobal()
		case '@':
			return l.lexAttribute()
		case '%':
			if l.AtExprStart() {
				return l.lexPercentLiteral()
			}
		}
	}

	switch curr {
	case '@':
		if next == '@' {
			return l.lexAttribute()
		}
	case '"':
		if l.currentState() == InInterpString {
			l.popState()
			l.Emit(STRINGEND)
			return err
		} else {
			return l.lexString()
		}
	case '\'':
		return l.lexRawString()
	case '}':
		var exitedState bool
		switch l.currentState() {
		case InInterp:
			exitedState = true
			l.Emit(INTERPEND)
		case OpenBrace:
			exitedState = true
			l.Emit(RBRACE)
		}
		if exitedState {
			l.popState()
		}
		return err
	case '{':
		l.pushState(OpenBrace)
		l.Emit(LBRACE)
		return err
	case '#':
		return l.lexComment()
	case '/':
		if l.currentState() == InRegex {
			l.popState()
			l.Emit(REGEXEND)
			return err
		}
		switch l.lastToken {
		case RBRACKET, RBRACE, RPAREN, INT, FLOAT, IDENT, CONSTANT, METHODIDENT:
			// keep going
		default:
			return l.lexRegex()
		}
	}

	for err == nil && (unicode.IsPunct(next) || unicode.IsSymbol(next)) {
		next, err = l.Peek()
		nextPunct := string(append(l.read, next))
		tok, currDefined := validPuncts[string(l.read)]
		_, nextDefined := validPuncts[nextPunct]
		if currDefined && !nextDefined {
			switch tok {
			case PLUS, MINUS:
				if l.AtExprStart() {
					l.Emit(UNARY_NUM)
					return err
				}
			case LPAREN, LBRACKET:
				if l.spaceConsumed {
					l.Emit(exprStartTokens[tok])
					return err
				}
				switch l.lastToken {
				case RPAREN, RBRACKET, RBRACE, IDENT, CONSTANT, METHODIDENT, YIELD, IVAR:
				default:
					l.Emit(exprStartTokens[tok])
					return err
				}

			}

			l.Emit(tok)
			return nil
		}
		curr, next, err = l.Advance()
	}
	if err != nil && err != io.EOF {
		return err
	}
	if _, defined := validPuncts[string(l.read)]; !defined {
		l.Emit(Illegal)
		return err
	}
	validTok := validPuncts[string(l.read)]
	switch validTok {
	case PLUS, MINUS:
		if l.AtExprStart() {
			l.Emit(UNARY_NUM)
			return err
		}
	case LPAREN, LBRACKET:
		if l.spaceConsumed {
			l.Emit(exprStartTokens[validTok])
			return err
		}
		switch l.lastToken {
		case RPAREN, RBRACKET, RBRACE, IDENT, CONSTANT, METHODIDENT, IVAR:
		default:
			l.Emit(exprStartTokens[validTok])
			return err
		}

	}
	l.Emit(validTok)
	return err
}

func (l *Lexer) lexSymbol() error {
	l.pushState(InSymbol)
	defer l.popState()
	return l.lexWord()
}

func (l *Lexer) lexGlobal() error {
	l.pushState(InGlobal)
	defer l.popState()
	return l.lexWord()
}

func (l *Lexer) lexWord() error {
	_, next, err := l.Advance()
	for err == nil && (unicode.IsLetter(next) || next == '_') {
		_, next, err = l.Advance()
	}
	if err != nil && err != io.EOF {
		return err
	}
	if p, defined := keywords[string(l.read)]; !defined {
		if unicode.IsUpper(l.read[0]) {
			l.Emit(CONSTANT)
		} else {
			switch l.currentState() {
			case InSymbol:
				next, err = l.Peek()
				if next == '!' || next == '?' {
					l.Advance()
				}
				l.Emit(SYMBOL)
			case InGlobal:
				l.Emit(GVAR)
			case InIVar:
				l.Emit(IVAR)
			case InCVar:
				l.Emit(CVAR)
			default:
				next, err = l.Peek()
				if next == '!' || next == '?' {
					l.Advance()
					l.Emit(METHODIDENT)
				} else if next == ':' {
					l.Advance()
					l.Emit(LABEL)
				} else {
					l.Emit(IDENT)
				}
			}
		}
	} else {
		switch string(l.read) {
		case "if", "unless", "while", "until", "rescue":
			if !l.AtExprStart() {
				l.Emit(keywordModifierTokens[p])
				break
			}
			fallthrough
		// Hack to support rspec-lite syntax for tests. Goal is to capture the
		// raw source of the block argument to the method call. Current design
		// of the lexer means that putting this hack in the parser has to
		// contend with concurrency so it's easy to manage all the state here
		// and just include that string in the token wrapper.
		case "gauntlet":
			l.gauntlet = true
			l.Emit(p)
		default:
			l.Emit(p)
		}
	}
	return err
}

func (l *Lexer) lexAttribute() error {
	curr, _, err := l.Advance()
	if curr == '@' {
		if err != nil {
			return err
		}
		l.pushState(InCVar)
		defer l.popState()
		return l.lexWord()
	}
	l.pushState(InIVar)
	defer l.popState()
	return l.lexWord()
}

func (l *Lexer) lexComment() error {
	_, next, err := l.Advance()
	for {
		var lastLoop bool
		if err == io.EOF {
			lastLoop = true
		}
		if next == '\n' {
			break
		}
		if lastLoop {
			break
		}
		_, next, err = l.Advance()
	}
	l.Emit(COMMENT)
	return err
}

func (l *Lexer) lexRegex() error {
	if l.currentState() != InRegex {
		l.Emit(REGEXBEG)
		l.pushState(InRegex)
	}

	curr, next, err := l.Advance()
	if curr == '/' {
		l.popState()
		l.Emit(REGEXEND)
		return err
	}
	for {
		var lastLoop bool
		if err == io.EOF {
			lastLoop = true
		}
		switch {
		case curr == '/':
			l.popState()
			l.Emit(REGEXEND)
		case next == '/':
			if len(l.read) > 0 {
				l.Emit(STRINGBODY)
			}
			l.Advance()
			l.popState()
			l.Emit(REGEXEND)
			return err
		case curr == '#' && next == '{':
			if len(l.read) > 1 {
				l.RewindBuffer()
				l.Emit(STRINGBODY)
				l.read = []rune{'#'}
			}
			l.Advance()
			l.pushState(InInterp)
			l.Emit(INTERPBEG)
			return err
		}
		if lastLoop {
			break
		}
		curr, next, err = l.Advance()
	}
	return err
}

func (l *Lexer) lexString() error {
	if l.currentState() != InInterpString {
		l.Emit(STRINGBEG)
		l.pushState(InInterpString)
		l.pushStringDelim('"')
	}
	curr, next, err := l.Advance()
	// empty string, should be handled better
	if curr == l.currentStringDelim() {
		l.popState()
		l.popStringDelim()
		l.Emit(STRINGEND)
		return err
	}
	for {
		var lastLoop bool
		if err == io.EOF {
			lastLoop = true
		}
		switch {
		case curr == l.currentStringDelim():
			l.popState()
			l.Emit(STRINGEND)
			l.popStringDelim()
		case next == l.currentStringDelim():
			if len(l.read) > 0 {
				l.Emit(STRINGBODY)
			}
			l.Advance()
			l.popState()
			l.Emit(STRINGEND)
			l.popStringDelim()
			return err
		case curr == '#' && next == '{':
			if len(l.read) > 1 {
				l.RewindBuffer()
				l.Emit(STRINGBODY)
				l.read = []rune{'#'}
			}
			l.Advance()
			l.pushState(InInterp)
			l.Emit(INTERPBEG)
			return err
		}
		if lastLoop {
			break
		}
		curr, next, err = l.Advance()
	}
	return err
}

func (l *Lexer) lexRawString() error {
	if l.currentState() != InRawString {
		l.Emit(RAWSTRINGBEG)
		l.pushState(InRawString)
		l.pushStringDelim('\'')
	}
	curr, next, err := l.Advance()
	// empty string, should be handled better
	if curr == l.currentStringDelim() {
		l.Emit(RAWSTRINGEND)
		l.popState()
		l.popStringDelim()
		return err
	}
	for {
		var lastLoop bool
		if err == io.EOF {
			lastLoop = true
		}
		switch {
		case curr == l.currentStringDelim():
			l.Emit(RAWSTRINGEND)
			l.popState()
			l.popStringDelim()
		case next == l.currentStringDelim():
			if len(l.read) > 0 {
				l.Emit(STRINGBODY)
			}
			l.Advance()
			l.Emit(RAWSTRINGEND)
			l.popState()
			l.popStringDelim()
			return err
		}
		if lastLoop {
			break
		}
		curr, next, err = l.Advance()
	}
	return err
}

func (l *Lexer) lexPercentLiteral() error {
	curr, next, err := l.Advance()
	if err != nil {
		return err
	}
	if !unicode.IsPunct(next) && !unicode.IsSymbol(next) {
		return fmt.Errorf("'%c' is not a valid delimiter for a percent literal", next)
	}
	l.pushStringDelim(next)
	l.Advance()
	switch curr {
	case 'w':
		l.Emit(RAWWORDSBEG)
		l.pushState(InRawString)
		return l.lexRawString()
	case 'W':
		l.Emit(WORDSBEG)
		l.pushState(InInterpString)
		return l.lexString()
	case 'q', 'Q', 'r', 'R', 'i', 'I', 's', 'S', 'x', 'X':
		return fmt.Errorf("'%%%c' literals are not supported", curr)
	default:
		return fmt.Errorf("'%c' is not a valid type of percent literal", curr)
	}
}
