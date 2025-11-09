// Package lexer 提供词法分析器的 Token 定义
// 词法分析器的职责是将源代码字符串分解为一系列有意义的词法单元（Token）
package lexer

// TokenType 表示 Token 的类型
// 使用 int 类型作为底层类型，通过 iota 枚举所有可能的 Token 类型
type TokenType int

const (
	// ========== 特殊 Token ==========
	// EOF 表示文件结束（End Of File）
	EOF TokenType = iota
	// ILLEGAL 表示非法字符或无法识别的 Token
	ILLEGAL

	// ========== 标识符和字面量 ==========
	// IDENT 表示标识符（变量名、函数名等）
	// 例如：main, x, myVariable, calculateSum
	IDENT
	// INT_LIT 表示整数字面量
	// 例如：123, 42, 0, 999
	INT_LIT
	// FLOAT_LIT 表示浮点数字面量
	// 例如：3.14, 2.5, 0.1
	FLOAT_LIT

	// ========== 算术运算符 ==========
	// ASSIGN 表示赋值运算符 =
	ASSIGN
	// PLUS 表示加法运算符 +
	PLUS
	// MINUS 表示减法运算符 -（也可作为负号）
	MINUS
	// MULT 表示乘法运算符 *
	MULT
	// DIV 表示除法运算符 /
	DIV
	// MOD 表示取模运算符 %（整数取余）
	MOD

	// ========== 比较运算符 ==========
	// EQ 表示相等比较运算符 ==
	EQ
	// NEQ 表示不等比较运算符 !=
	NEQ
	// LT 表示小于比较运算符 <
	LT
	// LE 表示小于等于比较运算符 <=
	LE
	// GT 表示大于比较运算符 >
	GT
	// GE 表示大于等于比较运算符 >=
	GE

	// ========== 逻辑运算符 ==========
	// AND 表示逻辑与运算符 &&
	AND
	// OR 表示逻辑或运算符 ||
	OR
	// NOT 表示逻辑非运算符 !
	NOT

	// ========== 分隔符 ==========
	// LPAREN 表示左括号 (
	LPAREN
	// RPAREN 表示右括号 )
	RPAREN
	// LBRACE 表示左花括号 {
	LBRACE
	// RBRACE 表示右花括号 }
	RBRACE
	// COMMA 表示逗号 ,（用于分隔函数参数等）
	COMMA
	// SEMI 表示分号 ;（语句结束符，可选）
	SEMI

	// ========== 关键字 ==========
	// INT 表示整数类型关键字 int
	INT
	// FLOAT 表示浮点数类型关键字 float
	FLOAT
	// RETURN 表示返回语句关键字 return
	RETURN
	// IF 表示条件语句关键字 if
	IF
	// ELSE 表示条件语句关键字 else
	ELSE
	// WHILE 表示循环语句关键字 while
	WHILE
)

// keywords 关键字映射表
// 用于在词法分析时快速判断一个标识符是否为保留关键字
// 例如："int" -> INT, "while" -> WHILE
var keywords = map[string]TokenType{
	"int":    INT,    // 整数类型
	"float":  FLOAT,  // 浮点数类型
	"return": RETURN, // 返回语句
	"if":     IF,     // 条件判断
	"else":   ELSE,   // 条件分支
	"while":  WHILE,  // 循环语句
}

// Token 表示一个词法单元（Token）
// 词法分析器将源代码分解为 Token 序列，每个 Token 包含以下信息：
type Token struct {
	Type    TokenType // Token 的类型（例如：IDENT, INT_LIT, PLUS）
	Literal string    // Token 的字面值（例如："main", "42", "+"）
	Line    int       // Token 在源代码中的行号（用于错误报告）
	Column  int       // Token 在源代码中的列号（用于错误报告）
}

// LookupIdent 检查一个标识符是否为关键字
// 参数 ident: 要检查的标识符字符串
// 返回值: 如果是关键字则返回对应的 TokenType，否则返回 IDENT
//
// 工作原理：
// 1. 在 keywords 映射表中查找该标识符
// 2. 如果找到，说明是关键字，返回对应的 TokenType（如 INT, WHILE 等）
// 3. 如果没找到，说明是普通标识符，返回 IDENT
//
// 例如：
// - LookupIdent("int") -> INT（关键字）
// - LookupIdent("myVar") -> IDENT（普通标识符）
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

// String 返回 TokenType 的字符串表示
// 这是一个实现 fmt.Stringer 接口的方法，用于调试和错误输出
// 将 TokenType 常量转换为可读的字符串形式
//
// 例如：
// - EOF.String() -> "EOF"
// - PLUS.String() -> "+"
// - INT.String() -> "int"
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
