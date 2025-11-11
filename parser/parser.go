// Package parser 提供语法分析器实现
// 语法分析器（Parser）是编译器前端的第二个阶段，负责将 Token 流
// 转换为抽象语法树（AST）
//
// 工作流程：
// Token 流 -> Parser -> AST -> 语义分析器
//
// 解析技术：
// - 递归下降解析（Recursive Descent Parsing）
// - 运算符优先级解析（Pratt Parsing / Precedence Climbing）
//
// 主要功能：
// 1. 解析函数定义
// 2. 解析语句（声明、赋值、控制流等）
// 3. 解析表达式（字面量、运算、函数调用等）
// 4. 错误检测和报告
package parser

import (
	"fmt"
	"goc/lexer"
	"strconv"
)

// Parser 语法分析器结构体
// 使用双 Token 缓冲（current + peek）实现向前看（lookahead）
type Parser struct {
	lexer     *lexer.Lexer // 词法分析器
	curToken  lexer.Token  // 当前 Token
	peekToken lexer.Token  // 下一个 Token（用于向前看）
	errors    []string     // 错误信息列表
}

// New 创建一个新的语法分析器
// 参数 l: 词法分析器实例
// 返回值: 初始化好的 Parser
//
// 初始化流程：
// 1. 创建 Parser 实例
// 2. 调用两次 nextToken() 填充 curToken 和 peekToken
// 3. 返回准备好的解析器
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		lexer:  l,
		errors: []string{},
	}
	// 读取两个 Token 来初始化 curToken 和 peekToken
	// 这样可以实现向前看一个 Token 的功能
	p.nextToken()
	p.nextToken()
	return p
}

// Errors 返回解析过程中收集的所有错误
func (p *Parser) Errors() []string {
	return p.errors
}

// ========== 辅助方法 ==========

// nextToken 推进到下一个 Token
// 将 peekToken 移动到 curToken，并从词法分析器读取新的 peekToken
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

// curTokenIs 检查当前 Token 是否为指定类型
func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

// peekTokenIs 检查下一个 Token 是否为指定类型
// 用于向前看（lookahead）
func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

// expectPeek 检查下一个 Token 是否为指定类型
// 如果是，推进到该 Token 并返回 true
// 如果不是，记录错误并返回 false
//
// 这是语法分析中最常用的模式：
// 期望某个特定的 Token（如 '('、'{'、';' 等）
func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

// peekError 记录"期望 Token 不匹配"的错误
// 参数 t: 期望的 Token 类型
func (p *Parser) peekError(t lexer.TokenType) {
	msg := fmt.Sprintf("Line %d:%d: expected next token to be %s, got %s instead",
		p.peekToken.Line, p.peekToken.Column, t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

// error 记录一般性错误
// 参数 msg: 错误消息
func (p *Parser) error(msg string) {
	fullMsg := fmt.Sprintf("Line %d:%d: %s", p.curToken.Line, p.curToken.Column, msg)
	p.errors = append(p.errors, fullMsg)
}

// ========== 核心解析方法 ==========

// ParseProgram 解析整个程序
// 返回值: 包含所有函数定义的 Program 节点
//
// 工作流程：
// 1. 创建空的 Program 节点
// 2. 循环解析所有函数定义，直到遇到 EOF
// 3. 将解析成功的函数添加到 Program 中
// 4. 返回完整的 AST
//
// 错误处理：
// - 如果解析某个函数失败，会记录错误但继续解析下一个函数
// - 所有错误可通过 Errors() 方法获取
func (p *Parser) ParseProgram() *Program {
	program := &Program{
		Functions: []*FunctionDecl{},
	}

	// 循环解析所有顶层函数定义
	for !p.curTokenIs(lexer.EOF) {
		fn := p.parseFunction()
		if fn != nil {
			program.Functions = append(program.Functions, fn)
		}
		p.nextToken()
	}

	return program
}

// parseFunction 解析函数定义
// 返回值: FunctionDecl 节点或 nil（解析失败）
//
// 语法格式：
//   <返回类型> <函数名> ( <参数列表> ) { <函数体> }
//
// 解析步骤：
// 1. 解析返回类型（int, float 等）
// 2. 解析函数名（标识符）
// 3. 期望 '('，解析参数列表，期望 ')'
// 4. 期望 '{'，解析函数体，期望 '}'
func (p *Parser) parseFunction() *FunctionDecl {
	fn := &FunctionDecl{Token: p.curToken}

	// 1. 解析返回类型
	if !p.isType(p.curToken.Type) {
		p.error(fmt.Sprintf("expected type, got %s", p.curToken.Literal))
		return nil
	}
	fn.ReturnType = p.curToken.Literal

	// 2. 解析函数名
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	fn.Name = p.curToken.Literal

	// 3. 解析参数列表：( <参数列表> )
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	fn.Parameters = p.parseParameters()

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	// 4. 解析函数体：{ <语句列表> }
	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	fn.Body = p.parseBlockStmt()

	return fn
}

// parseParameters parses function parameters
func (p *Parser) parseParameters() []*Parameter {
	params := []*Parameter{}

	if p.peekTokenIs(lexer.RPAREN) {
		return params
	}

	p.nextToken()

	// First parameter
	param := &Parameter{Token: p.curToken}
	if !p.isType(p.curToken.Type) {
		p.error(fmt.Sprintf("expected type, got %s", p.curToken.Literal))
		return nil
	}
	param.Type = p.curToken.Literal

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	param.Name = p.curToken.Literal
	params = append(params, param)

	// Additional parameters
	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to type

		param := &Parameter{Token: p.curToken}
		if !p.isType(p.curToken.Type) {
			p.error(fmt.Sprintf("expected type, got %s", p.curToken.Literal))
			return nil
		}
		param.Type = p.curToken.Literal

		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		param.Name = p.curToken.Literal
		params = append(params, param)
	}

	return params
}

// parseBlockStmt parses a block statement
func (p *Parser) parseBlockStmt() *BlockStmt {
	block := &BlockStmt{Token: p.curToken}
	block.Statements = []Statement{}

	p.nextToken()

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

// parseStatement parses a statement
func (p *Parser) parseStatement() Statement {
	switch p.curToken.Type {
	case lexer.INT, lexer.FLOAT, lexer.STR:
		return p.parseVarDeclStmt()
	case lexer.RETURN:
		return p.parseReturnStmt()
	case lexer.IF:
		return p.parseIfStmt()
	case lexer.WHILE:
		return p.parseWhileStmt()
	case lexer.IDENT:
		// Could be assignment or type-inferred declaration
		if p.peekTokenIs(lexer.ASSIGN) {
			return p.parseAssignOrInferredDecl()
		}
		return p.parseExprStmt()
	default:
		return p.parseExprStmt()
	}
}

// parseVarDeclStmt parses a variable declaration with explicit type
func (p *Parser) parseVarDeclStmt() *VarDeclStmt {
	stmt := &VarDeclStmt{Token: p.curToken}
	stmt.Type = p.curToken.Literal

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	// Check for initialization
	if p.peekTokenIs(lexer.ASSIGN) {
		p.nextToken() // consume identifier
		p.nextToken() // move to value
		stmt.Value = p.parseExpression(LOWEST)
	}

	return stmt
}

// parseAssignOrInferredDecl parses assignment or type-inferred declaration
func (p *Parser) parseAssignOrInferredDecl() Statement {
	// This could be either:
	// 1. x = 10 (first assignment, inferred type)
	// 2. x = 20 (reassignment)
	// We'll treat it as assignment here, and let semantic analysis
	// determine if it's a declaration or reassignment

	stmt := &AssignStmt{Token: p.curToken}
	stmt.Name = p.curToken.Literal

	p.nextToken() // consume identifier
	p.nextToken() // move to value

	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

// parseReturnStmt parses a return statement
func (p *Parser) parseReturnStmt() *ReturnStmt {
	stmt := &ReturnStmt{Token: p.curToken}

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

// parseIfStmt parses an if statement
func (p *Parser) parseIfStmt() *IfStmt {
	stmt := &IfStmt{Token: p.curToken}

	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	stmt.ThenBranch = p.parseBlockStmt()

	// Check for else branch
	if p.peekTokenIs(lexer.ELSE) {
		p.nextToken() // consume '}'
		p.nextToken() // consume 'else'

		if !p.expectPeek(lexer.LBRACE) {
			return nil
		}

		stmt.ElseBranch = p.parseBlockStmt()
	}

	return stmt
}

// parseWhileStmt parses a while loop
func (p *Parser) parseWhileStmt() *WhileStmt {
	stmt := &WhileStmt{Token: p.curToken}

	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStmt()

	return stmt
}

// parseExprStmt parses an expression statement
func (p *Parser) parseExprStmt() *ExprStmt {
	stmt := &ExprStmt{Token: p.curToken}
	stmt.Expression = p.parseExpression(LOWEST)
	return stmt
}

// ========== 运算符优先级定义 ==========
// 使用 Pratt Parsing（优先级爬升法）解析表达式
// 数值越大，优先级越高

const (
	_ int = iota
	LOWEST        // 最低优先级
	OR_PREC       // 1: || （逻辑或）
	AND_PREC      // 2: && （逻辑与）
	EQUALS        // 3: == != （相等比较）
	LESSGREATER   // 4: < <= > >= （大小比较）
	SUM           // 5: + - （加减）
	PRODUCT       // 6: * / % （乘除取模）
	PREFIX        // 7: -x !x （一元运算符）
	CALL          // 8: function() （函数调用，最高优先级）
)

// precedences 运算符优先级映射表
// 将 TokenType 映射到优先级数值
// 用于在解析表达式时确定运算顺序
//
// 例如：
// - x + y * z 会先解析 y * z（PRODUCT > SUM）
// - x == y && z 会先解析 x == y（EQUALS > AND_PREC）
var precedences = map[lexer.TokenType]int{
	lexer.OR:     OR_PREC,
	lexer.AND:    AND_PREC,
	lexer.EQ:     EQUALS,
	lexer.NEQ:    EQUALS,
	lexer.LT:     LESSGREATER,
	lexer.LE:     LESSGREATER,
	lexer.GT:     LESSGREATER,
	lexer.GE:     LESSGREATER,
	lexer.PLUS:   SUM,
	lexer.MINUS:  SUM,
	lexer.MULT:   PRODUCT,
	lexer.DIV:    PRODUCT,
	lexer.MOD:    PRODUCT,
	lexer.LPAREN: CALL,
}

// peekPrecedence 返回下一个 Token 的优先级
func (p *Parser) peekPrecedence() int {
	if prec, ok := precedences[p.peekToken.Type]; ok {
		return prec
	}
	return LOWEST
}

// curPrecedence 返回当前 Token 的优先级
func (p *Parser) curPrecedence() int {
	if prec, ok := precedences[p.curToken.Type]; ok {
		return prec
	}
	return LOWEST
}

// parseExpression 使用优先级爬升法解析表达式
// 参数 precedence: 当前的最低优先级
// 返回值: Expression 节点
//
// 工作原理（Pratt Parsing）：
// 1. 解析前缀表达式（数字、标识符、一元运算符、括号等）
// 2. 循环解析中缀表达式，只要下一个运算符的优先级高于当前优先级
//    - 如果是 '('，解析为函数调用
//    - 否则解析为二元运算
// 3. 返回最终的表达式树
//
// 例如解析 "1 + 2 * 3"：
// 1. 解析前缀: 1
// 2. 遇到 +（优先级 SUM），继续
// 3. 递归解析右侧，遇到 *（优先级 PRODUCT > SUM）
// 4. 先构建 (2 * 3)，再构建 (1 + ...)
// 5. 最终得到 BinaryExpr{1, +, BinaryExpr{2, *, 3}}
func (p *Parser) parseExpression(precedence int) Expression {
	// 1. 解析前缀表达式（表达式的开头部分）
	left := p.parsePrefixExpression()
	if left == nil {
		return nil
	}

	// 2. 循环解析中缀运算符，直到遇到：
	//    - 语句结束符（;, }, ), ,）
	//    - 优先级不高于当前优先级的运算符
	for !p.peekTokenIs(lexer.SEMI) && !p.peekTokenIs(lexer.RBRACE) &&
	      !p.peekTokenIs(lexer.RPAREN) && !p.peekTokenIs(lexer.COMMA) &&
	      precedence < p.peekPrecedence() {

		if p.peekTokenIs(lexer.LPAREN) {
			// 函数调用：func()
			p.nextToken()
			left = p.parseCallExpr(left)
		} else {
			// 二元运算：left <op> right
			p.nextToken()
			left = p.parseBinaryExpr(left)
		}
	}

	return left
}

// parsePrefixExpression parses a prefix expression
func (p *Parser) parsePrefixExpression() Expression {
	switch p.curToken.Type {
	case lexer.IDENT:
		return &Identifier{Name: p.curToken.Literal, Token: p.curToken}
	case lexer.INT_LIT:
		return p.parseIntLiteral()
	case lexer.FLOAT_LIT:
		return p.parseFloatLiteral()
	case lexer.STRING_LIT:
		return p.parseStringLiteral()
	case lexer.MINUS, lexer.NOT:
		return p.parseUnaryExpr()
	case lexer.LPAREN:
		return p.parseGroupedExpr()
	default:
		p.error(fmt.Sprintf("unexpected token: %s", p.curToken.Literal))
		return nil
	}
}

// parseIntLiteral parses an integer literal
func (p *Parser) parseIntLiteral() *IntLiteral {
	lit := &IntLiteral{Token: p.curToken}
	value, err := strconv.ParseInt(p.curToken.Literal, 10, 64)
	if err != nil {
		p.error(fmt.Sprintf("could not parse %s as integer", p.curToken.Literal))
		return nil
	}
	lit.Value = value
	return lit
}

// parseFloatLiteral parses a float literal
func (p *Parser) parseFloatLiteral() *FloatLiteral {
	lit := &FloatLiteral{Token: p.curToken}
	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.error(fmt.Sprintf("could not parse %s as float", p.curToken.Literal))
		return nil
	}
	lit.Value = value
	return lit
}

// parseStringLiteral parses a string literal
func (p *Parser) parseStringLiteral() *StringLiteral {
	return &StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

// parseUnaryExpr parses a unary expression
func (p *Parser) parseUnaryExpr() *UnaryExpr {
	expr := &UnaryExpr{Token: p.curToken, Operator: p.curToken.Literal}
	p.nextToken()
	expr.Operand = p.parseExpression(PREFIX)
	return expr
}

// parseGroupedExpr parses a grouped expression
func (p *Parser) parseGroupedExpr() Expression {
	p.nextToken()
	exp := p.parseExpression(LOWEST)
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}
	return exp
}

// parseBinaryExpr parses a binary expression
func (p *Parser) parseBinaryExpr(left Expression) Expression {
	expr := &BinaryExpr{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(precedence)

	return expr
}

// parseCallExpr parses a function call expression
func (p *Parser) parseCallExpr(function Expression) Expression {
	// function should be an Identifier
	ident, ok := function.(*Identifier)
	if !ok {
		p.error("function call target must be an identifier")
		return nil
	}

	expr := &CallExpr{Token: p.curToken, Function: ident.Name}
	expr.Arguments = p.parseCallArguments()
	return expr
}

// parseCallArguments parses function call arguments
func (p *Parser) parseCallArguments() []Expression {
	args := []Expression{}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return args
	}

	p.nextToken()
	args = append(args, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to next argument
		args = append(args, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return args
}

// isType checks if a token type represents a type keyword
func (p *Parser) isType(t lexer.TokenType) bool {
	return t == lexer.INT || t == lexer.FLOAT
}
