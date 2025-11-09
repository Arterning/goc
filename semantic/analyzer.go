package semantic

import (
	"fmt"
	"goc/parser"
	"goc/types"
)

// Symbol represents a variable or function in the symbol table
type Symbol struct {
	Name string
	Type types.Type
	Kind string // "variable" or "function"
}

// Scope represents a lexical scope
type Scope struct {
	parent  *Scope
	symbols map[string]*Symbol
}

// NewScope creates a new scope
func NewScope(parent *Scope) *Scope {
	return &Scope{
		parent:  parent,
		symbols: make(map[string]*Symbol),
	}
}

// Define adds a symbol to the current scope
func (s *Scope) Define(name string, typ types.Type, kind string) error {
	if _, exists := s.symbols[name]; exists {
		return fmt.Errorf("variable '%s' already declared in this scope", name)
	}
	s.symbols[name] = &Symbol{Name: name, Type: typ, Kind: kind}
	return nil
}

// Lookup finds a symbol in the current or parent scopes
func (s *Scope) Lookup(name string) (*Symbol, bool) {
	if sym, ok := s.symbols[name]; ok {
		return sym, true
	}
	if s.parent != nil {
		return s.parent.Lookup(name)
	}
	return nil, false
}

// Analyzer performs semantic analysis
type Analyzer struct {
	globalScope *Scope
	currentScope *Scope
	errors      []string
	exprTypes   map[parser.Expression]types.Type // Track expression types
}

// New creates a new Analyzer
func New() *Analyzer {
	globalScope := NewScope(nil)
	return &Analyzer{
		globalScope:  globalScope,
		currentScope: globalScope,
		errors:       []string{},
		exprTypes:    make(map[parser.Expression]types.Type),
	}
}

// Errors returns the analyzer errors
func (a *Analyzer) Errors() []string {
	return a.errors
}

// GetExprType returns the type of an expression
func (a *Analyzer) GetExprType(expr parser.Expression) types.Type {
	return a.exprTypes[expr]
}

// error records an error
func (a *Analyzer) error(msg string) {
	a.errors = append(a.errors, msg)
}

// Analyze performs semantic analysis on the program
func (a *Analyzer) Analyze(program *parser.Program) bool {
	// First pass: collect function declarations
	for _, fn := range program.Functions {
		a.analyzeFunctionDecl(fn)
	}

	// Second pass: analyze function bodies
	for _, fn := range program.Functions {
		a.analyzeFunctionBody(fn)
	}

	return len(a.errors) == 0
}

// analyzeFunctionDecl analyzes a function declaration
func (a *Analyzer) analyzeFunctionDecl(fn *parser.FunctionDecl) {
	returnType := types.FromString(fn.ReturnType)
	if returnType == nil {
		a.error(fmt.Sprintf("unknown return type '%s' for function '%s'", fn.ReturnType, fn.Name))
		return
	}

	// Add function to global scope
	if err := a.globalScope.Define(fn.Name, returnType, "function"); err != nil {
		a.error(err.Error())
	}
}

// analyzeFunctionBody analyzes a function body
func (a *Analyzer) analyzeFunctionBody(fn *parser.FunctionDecl) {
	// Create a new scope for the function
	a.currentScope = NewScope(a.globalScope)

	// Add parameters to the function scope
	for _, param := range fn.Parameters {
		paramType := types.FromString(param.Type)
		if paramType == nil {
			a.error(fmt.Sprintf("unknown parameter type '%s' for parameter '%s'", param.Type, param.Name))
			continue
		}
		if err := a.currentScope.Define(param.Name, paramType, "variable"); err != nil {
			a.error(err.Error())
		}
	}

	// Analyze function body
	a.analyzeBlockStmt(fn.Body)

	// Return to global scope
	a.currentScope = a.globalScope
}

// analyzeBlockStmt analyzes a block statement
func (a *Analyzer) analyzeBlockStmt(block *parser.BlockStmt) {
	// Create a new scope for the block
	a.currentScope = NewScope(a.currentScope)

	for _, stmt := range block.Statements {
		a.analyzeStatement(stmt)
	}

	// Return to parent scope
	a.currentScope = a.currentScope.parent
}

// analyzeStatement analyzes a statement
func (a *Analyzer) analyzeStatement(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.VarDeclStmt:
		a.analyzeVarDeclStmt(s)
	case *parser.AssignStmt:
		a.analyzeAssignStmt(s)
	case *parser.ReturnStmt:
		a.analyzeReturnStmt(s)
	case *parser.IfStmt:
		a.analyzeIfStmt(s)
	case *parser.WhileStmt:
		a.analyzeWhileStmt(s)
	case *parser.ExprStmt:
		a.analyzeExpression(s.Expression)
	case *parser.BlockStmt:
		a.analyzeBlockStmt(s)
	}
}

// analyzeVarDeclStmt analyzes a variable declaration statement
func (a *Analyzer) analyzeVarDeclStmt(stmt *parser.VarDeclStmt) {
	varType := types.FromString(stmt.Type)
	if varType == nil {
		a.error(fmt.Sprintf("unknown type '%s' for variable '%s'", stmt.Type, stmt.Name))
		return
	}

	// Check if variable is already declared
	if err := a.currentScope.Define(stmt.Name, varType, "variable"); err != nil {
		a.error(err.Error())
		return
	}

	// If there's an initializer, check its type
	if stmt.Value != nil {
		valueType := a.analyzeExpression(stmt.Value)
		if valueType != nil && !types.CanAssign(varType, valueType) {
			a.error(fmt.Sprintf("cannot assign value of type '%s' to variable '%s' of type '%s'",
				valueType.String(), stmt.Name, varType.String()))
		}
	}
}

// analyzeAssignStmt analyzes an assignment statement (could be inferred declaration)
func (a *Analyzer) analyzeAssignStmt(stmt *parser.AssignStmt) {
	// Check if this is a first assignment (type inference)
	sym, exists := a.currentScope.Lookup(stmt.Name)

	valueType := a.analyzeExpression(stmt.Value)
	if valueType == nil {
		return
	}

	if !exists {
		// This is a type-inferred declaration (first assignment)
		if err := a.currentScope.Define(stmt.Name, valueType, "variable"); err != nil {
			a.error(err.Error())
		}
	} else {
		// This is a reassignment
		if !types.CanAssign(sym.Type, valueType) {
			a.error(fmt.Sprintf("cannot assign value of type '%s' to variable '%s' of type '%s'",
				valueType.String(), stmt.Name, sym.Type.String()))
		}
	}
}

// analyzeReturnStmt analyzes a return statement
func (a *Analyzer) analyzeReturnStmt(stmt *parser.ReturnStmt) {
	if stmt.Value != nil {
		a.analyzeExpression(stmt.Value)
	}
}

// analyzeIfStmt analyzes an if statement
func (a *Analyzer) analyzeIfStmt(stmt *parser.IfStmt) {
	// Analyze condition
	condType := a.analyzeExpression(stmt.Condition)
	if condType != nil && !types.IsNumeric(condType) {
		a.error("if condition must be numeric")
	}

	// Analyze then branch
	a.analyzeBlockStmt(stmt.ThenBranch)

	// Analyze else branch if it exists
	if stmt.ElseBranch != nil {
		a.analyzeBlockStmt(stmt.ElseBranch)
	}
}

// analyzeWhileStmt analyzes a while statement
func (a *Analyzer) analyzeWhileStmt(stmt *parser.WhileStmt) {
	// Analyze condition
	condType := a.analyzeExpression(stmt.Condition)
	if condType != nil && !types.IsNumeric(condType) {
		a.error("while condition must be numeric")
	}

	// Analyze body
	a.analyzeBlockStmt(stmt.Body)
}

// analyzeExpression analyzes an expression and returns its type
func (a *Analyzer) analyzeExpression(expr parser.Expression) types.Type {
	switch e := expr.(type) {
	case *parser.IntLiteral:
		a.exprTypes[expr] = types.IntType
		return types.IntType
	case *parser.FloatLiteral:
		a.exprTypes[expr] = types.FloatType
		return types.FloatType
	case *parser.Identifier:
		return a.analyzeIdentifier(e)
	case *parser.BinaryExpr:
		return a.analyzeBinaryExpr(e)
	case *parser.UnaryExpr:
		return a.analyzeUnaryExpr(e)
	case *parser.CallExpr:
		return a.analyzeCallExpr(e)
	default:
		a.error("unknown expression type")
		return nil
	}
}

// analyzeIdentifier analyzes an identifier
func (a *Analyzer) analyzeIdentifier(ident *parser.Identifier) types.Type {
	sym, exists := a.currentScope.Lookup(ident.Name)
	if !exists {
		a.error(fmt.Sprintf("undefined variable '%s'", ident.Name))
		return nil
	}
	a.exprTypes[ident] = sym.Type
	return sym.Type
}

// analyzeBinaryExpr analyzes a binary expression
func (a *Analyzer) analyzeBinaryExpr(expr *parser.BinaryExpr) types.Type {
	leftType := a.analyzeExpression(expr.Left)
	rightType := a.analyzeExpression(expr.Right)

	if leftType == nil || rightType == nil {
		return nil
	}

	resultType, err := types.GetBinaryOpResultType(leftType, rightType, expr.Operator)
	if err != nil {
		a.error(err.Error())
		return nil
	}

	a.exprTypes[expr] = resultType
	return resultType
}

// analyzeUnaryExpr analyzes a unary expression
func (a *Analyzer) analyzeUnaryExpr(expr *parser.UnaryExpr) types.Type {
	operandType := a.analyzeExpression(expr.Operand)
	if operandType == nil {
		return nil
	}

	resultType, err := types.GetUnaryOpResultType(operandType, expr.Operator)
	if err != nil {
		a.error(err.Error())
		return nil
	}

	a.exprTypes[expr] = resultType
	return resultType
}

// analyzeCallExpr analyzes a function call expression
func (a *Analyzer) analyzeCallExpr(expr *parser.CallExpr) types.Type {
	// Look up function
	sym, exists := a.globalScope.Lookup(expr.Function)
	if !exists {
		a.error(fmt.Sprintf("undefined function '%s'", expr.Function))
		return nil
	}

	if sym.Kind != "function" {
		a.error(fmt.Sprintf("'%s' is not a function", expr.Function))
		return nil
	}

	// Analyze arguments
	for _, arg := range expr.Arguments {
		a.analyzeExpression(arg)
	}

	// For MVP, we don't do detailed argument type checking
	// Just return the function's return type
	a.exprTypes[expr] = sym.Type
	return sym.Type
}
