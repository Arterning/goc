// Package semantic 提供语义分析器实现
// 语义分析器（Semantic Analyzer）是编译器前端的第三个阶段
// 负责检查程序的语义正确性
//
// 工作流程：
// AST -> 语义分析器 -> 类型标注的 AST -> 代码生成器
//
// 主要功能：
// 1. 符号表管理：跟踪变量和函数的声明和作用域
// 2. 类型检查：确保表达式的类型正确
// 3. 类型推断：根据初始值推断变量类型
// 4. 错误检测：检测未声明变量、类型不匹配、重复声明等错误
//
// 关键概念：
// - 作用域（Scope）：变量的可见性范围
// - 符号表（Symbol Table）：存储标识符信息的数据结构
// - 词法作用域（Lexical Scoping）：作用域嵌套结构
package semantic

import (
	"fmt"
	"goc/parser"
	"goc/types"
)

// ========== 符号表数据结构 ==========

// Symbol 表示符号表中的一个符号（变量或函数）
// 存储标识符的名称、类型和种类信息
type Symbol struct {
	Name string      // 符号名称（变量名或函数名）
	Type types.Type  // 符号类型（int, float 等）
	Kind string      // 符号种类（"variable" 或 "function"）
}

// Scope 表示一个词法作用域
// 使用链表结构实现作用域嵌套：每个 Scope 指向其父 Scope
//
// 作用域层次示例：
//   全局作用域 (parent=nil)
//     ├─ 函数 main 作用域 (parent=全局)
//     │   ├─ if 块作用域 (parent=main)
//     │   └─ while 块作用域 (parent=main)
//     └─ 函数 add 作用域 (parent=全局)
type Scope struct {
	parent  *Scope              // 父作用域（nil 表示全局作用域）
	symbols map[string]*Symbol  // 当前作用域的符号表
}

// NewScope 创建一个新的作用域
// 参数 parent: 父作用域（nil 表示全局作用域）
// 返回值: 新创建的 Scope
func NewScope(parent *Scope) *Scope {
	return &Scope{
		parent:  parent,
		symbols: make(map[string]*Symbol),
	}
}

// Define 在当前作用域中定义一个新符号
// 参数 name: 符号名称
// 参数 typ: 符号类型
// 参数 kind: 符号种类（"variable" 或 "function"）
// 返回值: 如果符号已存在则返回错误，否则返回 nil
//
// 注意：不会查找父作用域，允许在内层作用域中定义与外层同名的变量（遮蔽）
func (s *Scope) Define(name string, typ types.Type, kind string) error {
	// 检查当前作用域是否已存在同名符号
	if _, exists := s.symbols[name]; exists {
		return fmt.Errorf("variable '%s' already declared in this scope", name)
	}
	s.symbols[name] = &Symbol{Name: name, Type: typ, Kind: kind}
	return nil
}

// Lookup 查找符号
// 参数 name: 符号名称
// 返回值: (找到的 Symbol, 是否找到)
//
// 查找策略：
// 1. 先在当前作用域查找
// 2. 如果没找到，递归到父作用域查找
// 3. 直到找到符号或到达全局作用域
//
// 这实现了词法作用域（Lexical Scoping）规则
func (s *Scope) Lookup(name string) (*Symbol, bool) {
	// 在当前作用域查找
	if sym, ok := s.symbols[name]; ok {
		return sym, true
	}
	// 递归到父作用域查找
	if s.parent != nil {
		return s.parent.Lookup(name)
	}
	return nil, false
}

// ========== 语义分析器 ==========

// Analyzer 语义分析器结构体
// 负责遍历 AST，进行类型检查和符号表管理
type Analyzer struct {
	globalScope  *Scope                            // 全局作用域（存储函数定义）
	currentScope *Scope                            // 当前作用域指针（随着分析过程移动）
	errors       []string                          // 错误信息列表
	exprTypes    map[parser.Expression]types.Type // 表达式类型映射表（用于代码生成阶段）
}

// New 创建一个新的语义分析器
// 返回值: 初始化好的 Analyzer
func New() *Analyzer {
	globalScope := NewScope(nil) // 创建全局作用域

	// 注册内建函数
	// print 函数：接受任意参数，返回 void（用 int 代替）
	globalScope.Define("print", types.IntType, "function")

	return &Analyzer{
		globalScope:  globalScope,
		currentScope: globalScope, // 初始时，当前作用域就是全局作用域
		errors:       []string{},
		exprTypes:    make(map[parser.Expression]types.Type),
	}
}

// Errors 返回分析过程中收集的所有错误
func (a *Analyzer) Errors() []string {
	return a.errors
}

// GetExprType 获取表达式的类型
// 参数 expr: 表达式节点
// 返回值: 表达式的类型
//
// 用途：代码生成阶段需要知道每个表达式的类型
// 以便生成正确的 LLVM IR 指令
func (a *Analyzer) GetExprType(expr parser.Expression) types.Type {
	return a.exprTypes[expr]
}

// error 记录错误信息
func (a *Analyzer) error(msg string) {
	a.errors = append(a.errors, msg)
}

// ========== 分析入口 ==========

// Analyze 对整个程序进行语义分析（单文件编译）
// 参数 program: 语法分析生成的 AST
// 返回值: true 表示分析成功，false 表示有错误
//
// 分析策略（两遍扫描）：
// 第一遍：收集所有函数声明，添加到全局符号表
//   - 这样可以支持函数的前向引用（调用在定义之前的函数）
// 第二遍：分析函数体
//   - 进行详细的类型检查和符号表管理
//
// 例如：
//   int main() {
//       return fib(10);  // 第一遍已经收集了 fib 的声明
//   }
//   int fib(int n) { ... }  // 定义在后面也可以
func (a *Analyzer) Analyze(program *parser.Program) bool {
	// 第一遍：收集所有函数声明
	for _, fn := range program.Functions {
		a.analyzeFunctionDecl(fn)
	}

	// 第二遍：分析函数体
	for _, fn := range program.Functions {
		a.analyzeFunctionBody(fn)
	}

	// 返回是否有错误
	return len(a.errors) == 0
}

// DeclareFunctionFromExternal 从外部声明函数（用于多文件编译）
// 参数 fn: 函数声明节点
// 这个方法只声明函数，不分析函数体
func (a *Analyzer) DeclareFunctionFromExternal(fn *parser.FunctionDecl) {
	a.analyzeFunctionDecl(fn)
}

// AnalyzeFunctions 分析函数体（用于多文件编译）
// 参数 program: 语法分析生成的 AST
// 返回值: true 表示分析成功，false 表示有错误
// 注意：假设所有函数声明已经通过 DeclareFunctionFromExternal 收集
func (a *Analyzer) AnalyzeFunctions(program *parser.Program) bool {
	// 清空之前的错误（如果有）
	startErrorCount := len(a.errors)

	// 分析所有函数体
	for _, fn := range program.Functions {
		a.analyzeFunctionBody(fn)
	}

	// 只返回本次分析是否有新错误
	return len(a.errors) == startErrorCount
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
	case *parser.StringLiteral:
		// 字符串字面量类型（暂时用 IntType 作为占位符）
		a.exprTypes[expr] = types.IntType
		return types.IntType
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
