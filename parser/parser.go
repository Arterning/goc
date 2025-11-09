package parser

import (
	"fmt"
	"goc/lexer"
	"strconv"
)

// Parser performs syntax analysis
type Parser struct {
	lexer     *lexer.Lexer
	curToken  lexer.Token
	peekToken lexer.Token
	errors    []string
}

// New creates a new Parser
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		lexer:  l,
		errors: []string{},
	}
	// Read two tokens to initialize curToken and peekToken
	p.nextToken()
	p.nextToken()
	return p
}

// Errors returns the parser errors
func (p *Parser) Errors() []string {
	return p.errors
}

// nextToken advances to the next token
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

// curTokenIs checks if current token is of given type
func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

// peekTokenIs checks if peek token is of given type
func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

// expectPeek advances if peek token matches, otherwise records error
func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

// peekError records a peek error
func (p *Parser) peekError(t lexer.TokenType) {
	msg := fmt.Sprintf("Line %d:%d: expected next token to be %s, got %s instead",
		p.peekToken.Line, p.peekToken.Column, t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

// error records an error
func (p *Parser) error(msg string) {
	fullMsg := fmt.Sprintf("Line %d:%d: %s", p.curToken.Line, p.curToken.Column, msg)
	p.errors = append(p.errors, fullMsg)
}

// ParseProgram parses the entire program
func (p *Parser) ParseProgram() *Program {
	program := &Program{
		Functions: []*FunctionDecl{},
	}

	for !p.curTokenIs(lexer.EOF) {
		fn := p.parseFunction()
		if fn != nil {
			program.Functions = append(program.Functions, fn)
		}
		p.nextToken()
	}

	return program
}

// parseFunction parses a function declaration
func (p *Parser) parseFunction() *FunctionDecl {
	fn := &FunctionDecl{Token: p.curToken}

	// Parse return type
	if !p.isType(p.curToken.Type) {
		p.error(fmt.Sprintf("expected type, got %s", p.curToken.Literal))
		return nil
	}
	fn.ReturnType = p.curToken.Literal

	// Parse function name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	fn.Name = p.curToken.Literal

	// Parse parameters
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	fn.Parameters = p.parseParameters()

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	// Parse body
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
	case lexer.INT, lexer.FLOAT:
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

// Operator precedence
const (
	_ int = iota
	LOWEST
	OR_PREC       // ||
	AND_PREC      // &&
	EQUALS        // == !=
	LESSGREATER   // < <= > >=
	SUM           // + -
	PRODUCT       // * / %
	PREFIX        // -x !x
	CALL          // function()
)

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

func (p *Parser) peekPrecedence() int {
	if prec, ok := precedences[p.peekToken.Type]; ok {
		return prec
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if prec, ok := precedences[p.curToken.Type]; ok {
		return prec
	}
	return LOWEST
}

// parseExpression parses an expression using precedence climbing
func (p *Parser) parseExpression(precedence int) Expression {
	// Parse prefix expression
	left := p.parsePrefixExpression()
	if left == nil {
		return nil
	}

	// Parse infix expressions
	for !p.peekTokenIs(lexer.SEMI) && !p.peekTokenIs(lexer.RBRACE) &&
	      !p.peekTokenIs(lexer.RPAREN) && !p.peekTokenIs(lexer.COMMA) &&
	      precedence < p.peekPrecedence() {

		if p.peekTokenIs(lexer.LPAREN) {
			// Function call
			p.nextToken()
			left = p.parseCallExpr(left)
		} else {
			// Binary expression
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
