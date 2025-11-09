// Package parser 提供抽象语法树（AST）的节点定义
// AST 是编译器前端的核心数据结构，用树形结构表示程序的语法结构
//
// 工作流程：
// Token 流 -> Parser -> AST -> 语义分析器
//
// AST 节点分类：
// 1. Statement（语句）：不产生值的程序单元
//    - 变量声明：int x = 10
//    - 赋值语句：x = 20
//    - 控制流：if, while, return
//
// 2. Expression（表达式）：产生值的程序单元
//    - 字面量：42, 3.14
//    - 标识符：x, myVar
//    - 运算：x + y, x * 2
//    - 函数调用：fib(10)
package parser

import (
	"goc/lexer"
)

// Node 是所有 AST 节点的基接口
// 所有 AST 节点都必须实现这个接口
type Node interface {
	// TokenLiteral 返回节点关联的 Token 字面值
	// 主要用于调试和错误报告
	TokenLiteral() string
}

// Statement 表示语句节点的接口
// 语句是执行某个动作的程序单元，不产生值
//
// 实现这个接口的节点：
// - VarDeclStmt: 变量声明
// - AssignStmt: 赋值语句
// - ReturnStmt: 返回语句
// - IfStmt: 条件语句
// - WhileStmt: 循环语句
// - BlockStmt: 代码块
type Statement interface {
	Node
	statementNode() // 标记方法，用于类型区分
}

// Expression 表示表达式节点的接口
// 表达式是产生值的程序单元
//
// 实现这个接口的节点：
// - IntLiteral: 整数字面量
// - FloatLiteral: 浮点数字面量
// - Identifier: 标识符
// - BinaryExpr: 二元运算
// - UnaryExpr: 一元运算
// - CallExpr: 函数调用
type Expression interface {
	Node
	expressionNode() // 标记方法，用于类型区分
}

// ========== 顶层节点 ==========

// Program 表示整个程序，是 AST 的根节点
// 一个程序由多个函数定义组成
//
// 例如：
//   int main() { return 0 }
//   int add(int a, int b) { return a + b }
//
// 会被解析为包含两个 FunctionDecl 的 Program
type Program struct {
	Functions []*FunctionDecl // 程序中的所有函数定义
}

// TokenLiteral 返回程序的 Token 字面值
// 如果有函数，返回第一个函数的 Token；否则返回空字符串
func (p *Program) TokenLiteral() string {
	if len(p.Functions) > 0 {
		return p.Functions[0].TokenLiteral()
	}
	return ""
}

// ========== 函数相关节点 ==========

// FunctionDecl 表示函数定义
// 函数定义包括返回类型、函数名、参数列表和函数体
//
// 语法格式：
//   <返回类型> <函数名>(<参数列表>) { <函数体> }
//
// 例如：
//   int add(int a, int b) {
//       return a + b
//   }
//
// 对应的 FunctionDecl:
//   ReturnType: "int"
//   Name: "add"
//   Parameters: [Parameter{Type:"int", Name:"a"}, Parameter{Type:"int", Name:"b"}]
//   Body: BlockStmt{...}
type FunctionDecl struct {
	ReturnType string        // 返回值类型（"int", "float"等）
	Name       string        // 函数名
	Parameters []*Parameter  // 参数列表
	Body       *BlockStmt    // 函数体（代码块）
	Token      lexer.Token   // 关联的 Token（返回类型的 Token）
}

func (fd *FunctionDecl) TokenLiteral() string { return fd.Token.Literal }

// Parameter 表示函数参数
// 参数包括类型和名称
//
// 例如：
//   int add(int a, float b)
//          ^^^^^^^  ^^^^^^^^
//        参数1: int a
//        参数2: float b
type Parameter struct {
	Type  string      // 参数类型（"int", "float"等）
	Name  string      // 参数名
	Token lexer.Token // 关联的 Token（类型的 Token）
}

func (p *Parameter) TokenLiteral() string { return p.Token.Literal }

// ========== 语句节点 ==========

// BlockStmt 表示代码块（一组语句的集合）
// 代码块用花括号 {} 包围
//
// 例如：
//   {
//       int x = 10
//       int y = 20
//       return x + y
//   }
//
// 对应的 BlockStmt:
//   Statements: [VarDeclStmt{...}, VarDeclStmt{...}, ReturnStmt{...}]
type BlockStmt struct {
	Statements []Statement // 代码块中的语句列表
	Token      lexer.Token // 关联的 Token（'{' 符号）
}

func (bs *BlockStmt) statementNode()       {}
func (bs *BlockStmt) TokenLiteral() string { return bs.Token.Literal }

// VarDeclStmt 表示变量声明语句
// 支持两种形式：
// 1. 显式类型声明：int x = 10
// 2. 类型推断声明：x = 10（通过初始值推断类型）
//
// 例如：
//   int x = 10    -> Type="int", Name="x", Value=IntLiteral{10}
//   int y         -> Type="int", Name="y", Value=nil（未初始化）
//   z = 3.14      -> Type="", Name="z", Value=FloatLiteral{3.14}（类型推断）
type VarDeclStmt struct {
	Type  string      // 变量类型（"int", "float"，或 "" 表示类型推断）
	Name  string      // 变量名
	Value Expression  // 初始值表达式（可以为 nil 表示未初始化）
	Token lexer.Token // 关联的 Token（类型的 Token 或标识符 Token）
}

func (vds *VarDeclStmt) statementNode()       {}
func (vds *VarDeclStmt) TokenLiteral() string { return vds.Token.Literal }

// AssignStmt 表示赋值语句
// 用于给已存在的变量赋新值，或进行类型推断的首次赋值
//
// 注意：这个节点在语法分析阶段无法区分是"首次赋值（类型推断）"还是"重新赋值"
// 需要在语义分析阶段通过符号表判断
//
// 例如：
//   x = 10      -> Name="x", Value=IntLiteral{10}
//   y = x + 5   -> Name="y", Value=BinaryExpr{...}
type AssignStmt struct {
	Name  string      // 变量名
	Value Expression  // 赋值的表达式
	Token lexer.Token // 关联的 Token（标识符 Token）
}

func (as *AssignStmt) statementNode()       {}
func (as *AssignStmt) TokenLiteral() string { return as.Token.Literal }

// ReturnStmt 表示返回语句
// 用于从函数中返回一个值
//
// 例如：
//   return 42      -> Value=IntLiteral{42}
//   return x + y   -> Value=BinaryExpr{...}
type ReturnStmt struct {
	Value Expression  // 返回值表达式
	Token lexer.Token // 关联的 Token（'return' 关键字）
}

func (rs *ReturnStmt) statementNode()       {}
func (rs *ReturnStmt) TokenLiteral() string { return rs.Token.Literal }

// IfStmt 表示条件语句（if-else）
// 根据条件的真假选择执行不同的代码分支
//
// 语法格式：
//   if (条件) {
//       then分支
//   } else {
//       else分支（可选）
//   }
//
// 例如：
//   if (x > 0) {
//       return 1
//   } else {
//       return -1
//   }
//
// 对应的 IfStmt:
//   Condition: BinaryExpr{x > 0}
//   ThenBranch: BlockStmt{return 1}
//   ElseBranch: BlockStmt{return -1}
type IfStmt struct {
	Condition  Expression  // 条件表达式
	ThenBranch *BlockStmt  // if 为真时执行的代码块
	ElseBranch *BlockStmt  // else 分支（可以为 nil，表示没有 else）
	Token      lexer.Token // 关联的 Token（'if' 关键字）
}

func (is *IfStmt) statementNode()       {}
func (is *IfStmt) TokenLiteral() string { return is.Token.Literal }

// WhileStmt 表示循环语句
// 当条件为真时，重复执行循环体
//
// 语法格式：
//   while (条件) {
//       循环体
//   }
//
// 例如：
//   while (i < 10) {
//       i = i + 1
//   }
//
// 对应的 WhileStmt:
//   Condition: BinaryExpr{i < 10}
//   Body: BlockStmt{i = i + 1}
type WhileStmt struct {
	Condition Expression  // 循环条件表达式
	Body      *BlockStmt  // 循环体
	Token     lexer.Token // 关联的 Token（'while' 关键字）
}

func (ws *WhileStmt) statementNode()       {}
func (ws *WhileStmt) TokenLiteral() string { return ws.Token.Literal }

// ExprStmt 表示表达式语句
// 将一个表达式作为语句使用（通常用于函数调用）
//
// 例如：
//   print(x)       -> ExprStmt{CallExpr{...}}
//   calculate()    -> ExprStmt{CallExpr{...}}
type ExprStmt struct {
	Expression Expression  // 表达式
	Token      lexer.Token // 关联的 Token
}

func (es *ExprStmt) statementNode()       {}
func (es *ExprStmt) TokenLiteral() string { return es.Token.Literal }

// ========== 表达式节点 ==========

// BinaryExpr 表示二元表达式
// 由左操作数、运算符、右操作数组成
//
// 支持的运算符：
// - 算术运算：+, -, *, /, %
// - 比较运算：==, !=, <, <=, >, >=
// - 逻辑运算：&&, ||
//
// 例如：
//   x + y      -> BinaryExpr{Left: Identifier{"x"}, Operator: "+", Right: Identifier{"y"}}
//   a * 2      -> BinaryExpr{Left: Identifier{"a"}, Operator: "*", Right: IntLiteral{2}}
//   x > 10     -> BinaryExpr{Left: Identifier{"x"}, Operator: ">", Right: IntLiteral{10}}
type BinaryExpr struct {
	Left     Expression  // 左操作数
	Operator string      // 运算符（"+", "-", "*", "/", "%", "==", "!=", "<", "<=", ">", ">=", "&&", "||"）
	Right    Expression  // 右操作数
	Token    lexer.Token // 关联的 Token（运算符 Token）
}

func (be *BinaryExpr) expressionNode()      {}
func (be *BinaryExpr) TokenLiteral() string { return be.Token.Literal }

// UnaryExpr 表示一元表达式
// 由运算符和操作数组成
//
// 支持的运算符：
// - 负号：-（取负）
// - 逻辑非：!（取反）
//
// 例如：
//   -x         -> UnaryExpr{Operator: "-", Operand: Identifier{"x"}}
//   !flag      -> UnaryExpr{Operator: "!", Operand: Identifier{"flag"}}
type UnaryExpr struct {
	Operator string      // 运算符（"-", "!"）
	Operand  Expression  // 操作数
	Token    lexer.Token // 关联的 Token（运算符 Token）
}

func (ue *UnaryExpr) expressionNode()      {}
func (ue *UnaryExpr) TokenLiteral() string { return ue.Token.Literal }

// IntLiteral 表示整数字面量
// 存储整数常量的值
//
// 例如：
//   42         -> IntLiteral{Value: 42}
//   0          -> IntLiteral{Value: 0}
//   999        -> IntLiteral{Value: 999}
type IntLiteral struct {
	Value int64       // 整数值
	Token lexer.Token // 关联的 Token
}

func (il *IntLiteral) expressionNode()      {}
func (il *IntLiteral) TokenLiteral() string { return il.Token.Literal }

// FloatLiteral 表示浮点数字面量
// 存储浮点数常量的值
//
// 例如：
//   3.14       -> FloatLiteral{Value: 3.14}
//   2.5        -> FloatLiteral{Value: 2.5}
//   0.1        -> FloatLiteral{Value: 0.1}
type FloatLiteral struct {
	Value float64     // 浮点数值
	Token lexer.Token // 关联的 Token
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }

// Identifier 表示标识符（变量名、函数名等）
// 标识符是对已声明的变量或函数的引用
//
// 例如：
//   x          -> Identifier{Name: "x"}
//   myVar      -> Identifier{Name: "myVar"}
//   result     -> Identifier{Name: "result"}
type Identifier struct {
	Name  string      // 标识符名称
	Token lexer.Token // 关联的 Token
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }

// CallExpr 表示函数调用表达式
// 由函数名和参数列表组成
//
// 例如：
//   add(1, 2)      -> CallExpr{Function: "add", Arguments: [IntLiteral{1}, IntLiteral{2}]}
//   fib(n)         -> CallExpr{Function: "fib", Arguments: [Identifier{"n"}]}
//   calculate()    -> CallExpr{Function: "calculate", Arguments: []}（无参数）
type CallExpr struct {
	Function  string       // 函数名
	Arguments []Expression // 参数列表
	Token     lexer.Token  // 关联的 Token（函数名 Token）
}

func (ce *CallExpr) expressionNode()      {}
func (ce *CallExpr) TokenLiteral() string { return ce.Token.Literal }
