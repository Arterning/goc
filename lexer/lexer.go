// Package lexer 提供词法分析器实现
// 词法分析器（Lexer）是编译器前端的第一个阶段，负责将源代码字符串
// 转换为一系列有意义的 Token（词法单元）
//
// 工作流程：
// 源代码字符串 -> Lexer -> Token 流 -> Parser
//
// 例如："int x = 10" 会被分解为：
// [INT, IDENT("x"), ASSIGN, INT_LIT("10")]
package lexer

import (
	"unicode"
)

// Lexer 词法分析器结构体
// 负责逐字符扫描源代码，识别并生成 Token
type Lexer struct {
	input        string // 输入的源代码字符串
	position     int    // 当前位置（指向当前字符的索引）
	readPosition int    // 下一个读取位置（position + 1，用于向前查看）
	ch           byte   // 当前正在检查的字符
	line         int    // 当前行号（用于错误报告，从 1 开始）
	column       int    // 当前列号（用于错误报告，从 0 开始）
}

// New 创建一个新的词法分析器实例
// 参数 input: 要分析的源代码字符串
// 返回值: 初始化好的 Lexer 指针
//
// 初始化流程：
// 1. 创建 Lexer 实例，设置初始行号为 1
// 2. 调用 readChar() 读取第一个字符
// 3. 返回准备好的词法分析器
func New(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar() // 读取第一个字符，初始化 ch 和 position
	return l
}

// readChar 读取下一个字符并推进位置指针
// 这是词法分析器的核心移动方法
//
// 工作流程：
// 1. 检查是否到达文件末尾
//    - 如果是，设置 ch 为 0（表示 EOF）
//    - 如果否，读取 readPosition 位置的字符
// 2. 更新 position 为当前 readPosition
// 3. readPosition 前进一位（为下次读取做准备）
// 4. column 列号加 1
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // ASCII code for NUL，表示文件结束
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.column++
}

// peekChar 查看下一个字符但不移动位置指针
// 这是"向前看"（lookahead）的实现，用于识别多字符运算符
//
// 使用场景：
// - 区分 '=' 和 '=='
// - 区分 '<' 和 '<='
// - 区分 '&' 和 '&&'
// - 区分 '/' 和 '//'（注释）
//
// 返回值: 下一个字符，如果到达文件末尾则返回 0
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// NextToken 返回下一个 Token
// 这是词法分析器的核心方法，被语法分析器反复调用以获取 Token 流
//
// 工作流程：
// 1. 跳过空白字符（空格、制表符、换行等）
// 2. 记录当前位置（行号和列号）
// 3. 根据当前字符类型进行匹配：
//    - 运算符：识别单字符或双字符运算符（如 =, ==, <, <=）
//    - 分隔符：括号、花括号、逗号、分号
//    - 标识符：字母开头的序列（变量名、关键字）
//    - 数字：整数或浮点数字面量
// 4. 返回识别出的 Token
//
// 返回值: 识别出的 Token 结构体
func (l *Lexer) NextToken() Token {
	var tok Token

	// 跳过所有空白字符（空格、制表符、换行符等）
	l.skipWhitespace()

	// 记录当前 Token 的位置信息（用于错误报告）
	tok.Line = l.line
	tok.Column = l.column

	// 根据当前字符类型进行模式匹配
	switch l.ch {
	case '=':
		// 需要向前看一位，区分 '=' 和 '=='
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar() // 读取第二个 '='
			tok = Token{Type: EQ, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = newToken(ASSIGN, l.ch, l.line, l.column)
		}
	case '+':
		tok = newToken(PLUS, l.ch, l.line, l.column)
	case '-':
		tok = newToken(MINUS, l.ch, l.line, l.column)
	case '*':
		tok = newToken(MULT, l.ch, l.line, l.column)
	case '/':
		// 检查是否为注释（'//' 开头的单行注释）
		if l.peekChar() == '/' {
			l.skipComment() // 跳过整行注释
			return l.NextToken() // 递归调用，返回注释后的下一个 Token
		}
		tok = newToken(DIV, l.ch, l.line, l.column)
	case '%':
		tok = newToken(MOD, l.ch, l.line, l.column)
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: NEQ, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = newToken(NOT, l.ch, l.line, l.column)
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: LE, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = newToken(LT, l.ch, l.line, l.column)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: GE, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = newToken(GT, l.ch, l.line, l.column)
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: AND, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = newToken(ILLEGAL, l.ch, l.line, l.column)
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: OR, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = newToken(ILLEGAL, l.ch, l.line, l.column)
		}
	case '(':
		tok = newToken(LPAREN, l.ch, l.line, l.column)
	case ')':
		tok = newToken(RPAREN, l.ch, l.line, l.column)
	case '{':
		tok = newToken(LBRACE, l.ch, l.line, l.column)
	case '}':
		tok = newToken(RBRACE, l.ch, l.line, l.column)
	case ',':
		tok = newToken(COMMA, l.ch, l.line, l.column)
	case ';':
		tok = newToken(SEMI, l.ch, l.line, l.column)
	case 0:
		// 到达文件末尾
		tok.Literal = ""
		tok.Type = EOF
	default:
		// 处理标识符（变量名、函数名、关键字）
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier() // 读取完整的标识符
			tok.Type = LookupIdent(tok.Literal) // 检查是否为关键字
			tok.Line = l.line
			tok.Column = l.column - len(tok.Literal)
			return tok // 注意：这里直接返回，因为 readIdentifier 已经移动了位置
		} else if isDigit(l.ch) {
			// 处理数字字面量（整数或浮点数）
			tok.Line = l.line
			tok.Column = l.column
			literal, isFloat := l.readNumber() // 读取完整的数字
			tok.Literal = literal
			if isFloat {
				tok.Type = FLOAT_LIT
			} else {
				tok.Type = INT_LIT
			}
			return tok // 注意：这里直接返回，因为 readNumber 已经移动了位置
		} else {
			// 无法识别的字符，标记为 ILLEGAL
			tok = newToken(ILLEGAL, l.ch, l.line, l.column)
		}
	}

	l.readChar() // 移动到下一个字符
	return tok
}

// newToken 创建一个新的 Token
// 参数 tokenType: Token 类型
// 参数 ch: 当前字符
// 参数 line, column: 位置信息
// 返回值: 构造好的 Token 结构体
//
// 这是一个辅助函数，用于快速创建单字符 Token
func newToken(tokenType TokenType, ch byte, line, column int) Token {
	return Token{Type: tokenType, Literal: string(ch), Line: line, Column: column}
}

// readIdentifier 读取一个完整的标识符
// 标识符规则：以字母或下划线开头，后面可以跟字母、数字或下划线
//
// 工作流程：
// 1. 记录起始位置
// 2. 持续读取字符，直到遇到非字母非数字字符
// 3. 返回完整的标识符字符串
//
// 例如：
// - "int" -> 读取 "int"
// - "myVar123" -> 读取 "myVar123"
// - "calculate_sum" -> 读取 "calculate_sum"
func (l *Lexer) readIdentifier() string {
	position := l.position
	// 持续读取字母和数字，直到遇到其他字符
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber 读取一个数字（整数或浮点数）
// 返回值: (数字字符串, 是否为浮点数)
//
// 支持的格式：
// - 整数：123, 0, 999
// - 浮点数：3.14, 2.5, 0.1
// - 带 'f' 后缀的浮点数：3.14f, 2.0f
//
// 工作流程：
// 1. 读取整数部分（连续的数字）
// 2. 检查是否有小数点 '.'
//    - 如果有且后面跟数字，读取小数部分，标记为浮点数
// 3. 检查是否有 'f' 或 'F' 后缀
//    - 如果有，标记为浮点数
// 4. 返回完整的数字字符串和类型标记
func (l *Lexer) readNumber() (string, bool) {
	position := l.position
	isFloat := false

	// 读取整数部分
	for isDigit(l.ch) {
		l.readChar()
	}

	// 检查小数点（需要确保小数点后有数字，避免将 "3." 识别为浮点数）
	if l.ch == '.' && isDigit(l.peekChar()) {
		isFloat = true
		l.readChar() // 消耗小数点 '.'
		// 读取小数部分
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	// 检查 'f' 后缀（例如：3.14f 或 2f）
	if l.ch == 'f' || l.ch == 'F' {
		isFloat = true
		l.readChar() // 消耗 'f'
	}

	return l.input[position:l.position], isFloat
}

// skipWhitespace 跳过所有空白字符
// 空白字符包括：空格、制表符、换行符、回车符
//
// 特殊处理：
// - 遇到换行符 '\n' 时，行号加 1，列号重置为 0
// - 这样可以正确追踪 Token 的位置信息
//
// 注意：虽然跳过了换行符，但语法分析器可以通过其他方式识别语句结束
// （本编译器支持可选分号，换行即可表示语句结束）
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		if l.ch == '\n' {
			l.line++    // 行号加 1
			l.column = 0 // 列号重置
		}
		l.readChar()
	}
}

// skipComment 跳过单行注释
// 注释格式：// 开头，到行尾结束
//
// 工作流程：
// 1. 持续读取字符，直到遇到换行符 '\n' 或文件结束符 0
// 2. 不处理注释内容，直接跳过
//
// 例如：
// "// this is a comment\nx = 10"
// 会跳过 "// this is a comment"，从 '\n' 后继续
func (l *Lexer) skipComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

// isLetter 检查字符是否为字母或下划线
// 参数 ch: 要检查的字符
// 返回值: true 表示是字母或下划线，false 表示不是
//
// 用途：
// - 识别标识符的起始字符
// - 识别标识符的后续字符
//
// 支持的字符：
// - 所有 Unicode 字母（包括中文等，但通常不建议使用）
// - 下划线 '_'
func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}

// isDigit 检查字符是否为数字
// 参数 ch: 要检查的字符
// 返回值: true 表示是 0-9 的数字，false 表示不是
//
// 用途：
// - 识别数字字面量
// - 识别标识符中的数字部分
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
