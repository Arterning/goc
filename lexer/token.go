package lexer

// TokenType represents the type of a token
type TokenType int

const (
	// Special tokens
	EOF TokenType = iota
	ILLEGAL

	// Identifiers and literals
	IDENT   // variable names, function names
	INT_LIT // 123
	FLOAT_LIT // 3.14

	// Operators
	ASSIGN  // =
	PLUS    // +
	MINUS   // -
	MULT    // *
	DIV     // /
	MOD     // %

	// Comparison operators
	EQ      // ==
	NEQ     // !=
	LT      // <
	LE      // <=
	GT      // >
	GE      // >=

	// Logical operators
	AND     // &&
	OR      // ||
	NOT     // !

	// Delimiters
	LPAREN  // (
	RPAREN  // )
	LBRACE  // {
	RBRACE  // }
	COMMA   // ,
	SEMI    // ;

	// Keywords
	INT     // int
	FLOAT   // float
	RETURN  // return
	IF      // if
	ELSE    // else
	WHILE   // while
)

var keywords = map[string]TokenType{
	"int":    INT,
	"float":  FLOAT,
	"return": RETURN,
	"if":     IF,
	"else":   ELSE,
	"while":  WHILE,
}

// Token represents a lexical token
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

// LookupIdent checks if an identifier is a keyword
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

// String returns a string representation of the token type
func (t TokenType) String() string {
	switch t {
	case EOF:
		return "EOF"
	case ILLEGAL:
		return "ILLEGAL"
	case IDENT:
		return "IDENT"
	case INT_LIT:
		return "INT_LIT"
	case FLOAT_LIT:
		return "FLOAT_LIT"
	case ASSIGN:
		return "="
	case PLUS:
		return "+"
	case MINUS:
		return "-"
	case MULT:
		return "*"
	case DIV:
		return "/"
	case MOD:
		return "%"
	case EQ:
		return "=="
	case NEQ:
		return "!="
	case LT:
		return "<"
	case LE:
		return "<="
	case GT:
		return ">"
	case GE:
		return ">="
	case AND:
		return "&&"
	case OR:
		return "||"
	case NOT:
		return "!"
	case LPAREN:
		return "("
	case RPAREN:
		return ")"
	case LBRACE:
		return "{"
	case RBRACE:
		return "}"
	case COMMA:
		return ","
	case SEMI:
		return ";"
	case INT:
		return "int"
	case FLOAT:
		return "float"
	case RETURN:
		return "return"
	case IF:
		return "if"
	case ELSE:
		return "else"
	case WHILE:
		return "while"
	default:
		return "UNKNOWN"
	}
}
