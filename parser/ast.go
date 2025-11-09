package parser

import (
	"goc/lexer"
)

// Node is the base interface for all AST nodes
type Node interface {
	TokenLiteral() string
}

// Statement represents a statement node
type Statement interface {
	Node
	statementNode()
}

// Expression represents an expression node
type Expression interface {
	Node
	expressionNode()
}

// Program is the root node of the AST
type Program struct {
	Functions []*FunctionDecl
}

func (p *Program) TokenLiteral() string {
	if len(p.Functions) > 0 {
		return p.Functions[0].TokenLiteral()
	}
	return ""
}

// FunctionDecl represents a function declaration
type FunctionDecl struct {
	ReturnType string
	Name       string
	Parameters []*Parameter
	Body       *BlockStmt
	Token      lexer.Token // the return type token
}

func (fd *FunctionDecl) TokenLiteral() string { return fd.Token.Literal }

// Parameter represents a function parameter
type Parameter struct {
	Type  string
	Name  string
	Token lexer.Token
}

func (p *Parameter) TokenLiteral() string { return p.Token.Literal }

// BlockStmt represents a block of statements
type BlockStmt struct {
	Statements []Statement
	Token      lexer.Token // the '{' token
}

func (bs *BlockStmt) statementNode()       {}
func (bs *BlockStmt) TokenLiteral() string { return bs.Token.Literal }

// VarDeclStmt represents a variable declaration
type VarDeclStmt struct {
	Type  string      // "int", "float", or "" for inferred
	Name  string
	Value Expression  // can be nil for uninitialized
	Token lexer.Token // the type token or identifier token
}

func (vds *VarDeclStmt) statementNode()       {}
func (vds *VarDeclStmt) TokenLiteral() string { return vds.Token.Literal }

// AssignStmt represents an assignment statement
type AssignStmt struct {
	Name  string
	Value Expression
	Token lexer.Token // the identifier token
}

func (as *AssignStmt) statementNode()       {}
func (as *AssignStmt) TokenLiteral() string { return as.Token.Literal }

// ReturnStmt represents a return statement
type ReturnStmt struct {
	Value Expression
	Token lexer.Token // the 'return' token
}

func (rs *ReturnStmt) statementNode()       {}
func (rs *ReturnStmt) TokenLiteral() string { return rs.Token.Literal }

// IfStmt represents an if statement
type IfStmt struct {
	Condition  Expression
	ThenBranch *BlockStmt
	ElseBranch *BlockStmt // can be nil
	Token      lexer.Token // the 'if' token
}

func (is *IfStmt) statementNode()       {}
func (is *IfStmt) TokenLiteral() string { return is.Token.Literal }

// WhileStmt represents a while loop
type WhileStmt struct {
	Condition Expression
	Body      *BlockStmt
	Token     lexer.Token // the 'while' token
}

func (ws *WhileStmt) statementNode()       {}
func (ws *WhileStmt) TokenLiteral() string { return ws.Token.Literal }

// ExprStmt represents an expression statement
type ExprStmt struct {
	Expression Expression
	Token      lexer.Token
}

func (es *ExprStmt) statementNode()       {}
func (es *ExprStmt) TokenLiteral() string { return es.Token.Literal }

// BinaryExpr represents a binary expression
type BinaryExpr struct {
	Left     Expression
	Operator string
	Right    Expression
	Token    lexer.Token // the operator token
}

func (be *BinaryExpr) expressionNode()      {}
func (be *BinaryExpr) TokenLiteral() string { return be.Token.Literal }

// UnaryExpr represents a unary expression
type UnaryExpr struct {
	Operator string
	Operand  Expression
	Token    lexer.Token // the operator token
}

func (ue *UnaryExpr) expressionNode()      {}
func (ue *UnaryExpr) TokenLiteral() string { return ue.Token.Literal }

// IntLiteral represents an integer literal
type IntLiteral struct {
	Value int64
	Token lexer.Token
}

func (il *IntLiteral) expressionNode()      {}
func (il *IntLiteral) TokenLiteral() string { return il.Token.Literal }

// FloatLiteral represents a float literal
type FloatLiteral struct {
	Value float64
	Token lexer.Token
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }

// Identifier represents an identifier
type Identifier struct {
	Name  string
	Token lexer.Token
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }

// CallExpr represents a function call
type CallExpr struct {
	Function  string
	Arguments []Expression
	Token     lexer.Token // the identifier token
}

func (ce *CallExpr) expressionNode()      {}
func (ce *CallExpr) TokenLiteral() string { return ce.Token.Literal }
