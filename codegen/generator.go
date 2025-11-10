// Package codegen 提供 LLVM IR 代码生成器实现
// 代码生成器（Code Generator）是编译器前端的最后一个阶段
// 负责将 AST 转换为 LLVM IR（中间表示）
//
// 工作流程：
// 类型标注的 AST -> 代码生成器 -> LLVM IR -> Clang -> 机器码
//
// 主要功能：
// 1. 将高级语法结构转换为 LLVM IR 指令
// 2. 管理变量存储（使用 alloca 分配栈空间）
// 3. 生成控制流图（CFG）
// 4. 处理类型转换和运算符
//
// LLVM IR 简介：
// - SSA 形式（Static Single Assignment）：每个变量只赋值一次
// - 基本块（Basic Block）：顺序执行的指令序列
// - 控制流（Control Flow）：基本块之间的跳转
//
// 关键概念：
// - alloca: 在栈上分配内存
// - load: 从内存加载值
// - store: 向内存存储值
// - phi: SSA 形式中的选择指令
package codegen

import (
	"fmt"
	"goc/parser"
	"goc/semantic"
	"goc/types"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	llvmTypes "github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

// Generator LLVM IR 代码生成器结构体
// 负责遍历 AST 并生成对应的 LLVM IR 代码
type Generator struct {
	module       *ir.Module             // LLVM 模块（包含所有函数和全局变量）
	builder      *ir.Block              // 当前正在构建的基本块
	currentFunc  *ir.Func               // 当前正在生成的函数
	analyzer     *semantic.Analyzer     // 语义分析器（用于获取表达式类型）
	variables    map[string]value.Value // 变量存储映射（变量名 -> alloca 指令）
	functions    map[string]*ir.Func    // 函数映射（函数名 -> LLVM 函数）
	stringCount  int                    // 字符串常量计数器
	printfFunc   *ir.Func               // printf 函数声明
	putsFunc     *ir.Func               // puts 函数声明
	putcharFunc  *ir.Func               // putchar 函数声明
}

// New 创建一个新的代码生成器
// 参数 analyzer: 语义分析器（用于获取类型信息）
// 返回值: 初始化好的 Generator
func New(analyzer *semantic.Analyzer) *Generator {
	return &Generator{
		module:    ir.NewModule(),
		analyzer:  analyzer,
		variables: make(map[string]value.Value),
		functions: make(map[string]*ir.Func),
	}
}

// ========== C 标准库函数声明 ==========

// declareCLibFunctions 声明 C 标准库函数
// 声明 printf, puts, putchar 用于 print 功能
func (g *Generator) declareCLibFunctions() {
	// 声明 printf: int printf(ptr, ...)
	// 用于格式化输出
	g.printfFunc = g.module.NewFunc("printf", llvmTypes.I32,
		ir.NewParam("format", llvmTypes.NewPointer(llvmTypes.I8)))
	g.printfFunc.Sig.Variadic = true

	// 声明 puts: int puts(ptr)
	// 用于输出字符串（自动添加换行）
	g.putsFunc = g.module.NewFunc("puts", llvmTypes.I32,
		ir.NewParam("s", llvmTypes.NewPointer(llvmTypes.I8)))

	// 声明 putchar: int putchar(int)
	// 用于输出单个字符
	g.putcharFunc = g.module.NewFunc("putchar", llvmTypes.I32,
		ir.NewParam("c", llvmTypes.I32))
}

// ========== 代码生成入口 ==========

// Generate 为整个程序生成 LLVM IR
// 参数 program: 经过语义分析的 AST
// 返回值: (LLVM 模块, 错误)
//
// 生成策略（三步）：
// 第一步：声明 C 标准库函数（printf, puts, putchar）
//   - 这样可以在代码中调用这些函数
// 第二步：声明所有函数签名
//   - 这样可以支持函数的前向引用
// 第三步：生成函数体
//   - 生成详细的 LLVM IR 指令
//
// 生成的 LLVM IR 特点：
// - 使用内存模型（alloca/load/store）而不是 SSA 寄存器
// - 每个变量对应一个栈上的内存位置
// - 简化了代码生成，但可能不是最优的 IR
func (g *Generator) Generate(program *parser.Program) (*ir.Module, error) {
	// 第一步：声明 C 标准库函数
	g.declareCLibFunctions()

	// 第二步：声明所有函数签名
	for _, fn := range program.Functions {
		if err := g.declareFunctionSignature(fn); err != nil {
			return nil, err
		}
	}

	// 第三步：生成函数体
	for _, fn := range program.Functions {
		if err := g.generateFunction(fn); err != nil {
			return nil, err
		}
	}

	return g.module, nil
}

// DeclareExternalFunction 声明外部函数（用于多文件编译）
// 参数 name: 函数名
// 参数 returnType: 返回类型
// 参数 paramTypes: 参数类型列表
func (g *Generator) DeclareExternalFunction(name string, returnType string, paramTypes []string) error {
	// 检查函数是否已经声明
	if _, exists := g.functions[name]; exists {
		return nil // 已经声明过了
	}

	// 获取返回类型
	retType, err := g.getLLVMType(returnType)
	if err != nil {
		return err
	}

	// 获取参数类型
	var params []*ir.Param
	for i, paramType := range paramTypes {
		pType, err := g.getLLVMType(paramType)
		if err != nil {
			return err
		}
		params = append(params, ir.NewParam(fmt.Sprintf("arg%d", i), pType))
	}

	// 声明函数（不定义函数体）
	llvmFunc := g.module.NewFunc(name, retType, params...)
	g.functions[name] = llvmFunc

	return nil
}

// declareFunctionSignature declares a function signature
func (g *Generator) declareFunctionSignature(fn *parser.FunctionDecl) error {
	// Get return type
	returnType, err := g.getLLVMType(fn.ReturnType)
	if err != nil {
		return err
	}

	// Get parameter types
	var params []*ir.Param
	for _, param := range fn.Parameters {
		paramType, err := g.getLLVMType(param.Type)
		if err != nil {
			return err
		}
		params = append(params, ir.NewParam(param.Name, paramType))
	}

	// Create function
	llvmFunc := g.module.NewFunc(fn.Name, returnType, params...)
	g.functions[fn.Name] = llvmFunc

	return nil
}

// generateFunction generates LLVM IR for a function
func (g *Generator) generateFunction(fn *parser.FunctionDecl) error {
	llvmFunc := g.functions[fn.Name]
	g.currentFunc = llvmFunc

	// Create entry block
	entry := llvmFunc.NewBlock("entry")
	g.builder = entry

	// Clear variable map for new function
	g.variables = make(map[string]value.Value)

	// Allocate storage for parameters
	for i, param := range fn.Parameters {
		paramType, _ := g.getLLVMType(param.Type)
		alloca := entry.NewAlloca(paramType)
		// 不要给 alloca 指定名字，避免与参数名冲突
		// alloca.LocalIdent = ir.NewLocalIdent(param.Name)
		entry.NewStore(llvmFunc.Params[i], alloca)
		g.variables[param.Name] = alloca
	}

	// Generate function body
	if err := g.generateBlockStmt(fn.Body); err != nil {
		return err
	}

	// Add default return if missing
	if g.builder.Term == nil {
		if fn.ReturnType == "int" {
			g.builder.NewRet(constant.NewInt(llvmTypes.I32, 0))
		} else if fn.ReturnType == "float" {
			g.builder.NewRet(constant.NewFloat(llvmTypes.Float, 0.0))
		}
	}

	return nil
}

// generateBlockStmt generates LLVM IR for a block statement
func (g *Generator) generateBlockStmt(block *parser.BlockStmt) error {
	for _, stmt := range block.Statements {
		if err := g.generateStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

// generateStatement generates LLVM IR for a statement
func (g *Generator) generateStatement(stmt parser.Statement) error {
	switch s := stmt.(type) {
	case *parser.VarDeclStmt:
		return g.generateVarDeclStmt(s)
	case *parser.AssignStmt:
		return g.generateAssignStmt(s)
	case *parser.ReturnStmt:
		return g.generateReturnStmt(s)
	case *parser.IfStmt:
		return g.generateIfStmt(s)
	case *parser.WhileStmt:
		return g.generateWhileStmt(s)
	case *parser.ExprStmt:
		_, err := g.generateExpression(s.Expression)
		return err
	case *parser.BlockStmt:
		return g.generateBlockStmt(s)
	default:
		return fmt.Errorf("unknown statement type")
	}
}

// generateVarDeclStmt generates LLVM IR for a variable declaration
func (g *Generator) generateVarDeclStmt(stmt *parser.VarDeclStmt) error {
	// Get LLVM type
	llvmType, err := g.getLLVMType(stmt.Type)
	if err != nil {
		return err
	}

	// Allocate storage
	alloca := g.builder.NewAlloca(llvmType)
	// 让 LLVM 自动生成唯一的变量名，避免冲突
	// alloca.LocalIdent = ir.NewLocalIdent(stmt.Name)
	g.variables[stmt.Name] = alloca

	// Initialize if value provided
	if stmt.Value != nil {
		value, err := g.generateExpression(stmt.Value)
		if err != nil {
			return err
		}
		g.builder.NewStore(value, alloca)
	}

	return nil
}

// generateAssignStmt generates LLVM IR for an assignment statement
func (g *Generator) generateAssignStmt(stmt *parser.AssignStmt) error {
	// Check if variable exists (for type inference)
	alloca, exists := g.variables[stmt.Name]

	// Generate value
	val, err := g.generateExpression(stmt.Value)
	if err != nil {
		return err
	}

	if !exists {
		// Type-inferred declaration
		exprType := g.analyzer.GetExprType(stmt.Value)
		llvmType, err := g.typeToLLVMType(exprType)
		if err != nil {
			return err
		}
		newAlloca := g.builder.NewAlloca(llvmType)
		// 让 LLVM 自动生成唯一的变量名
		// newAlloca.LocalIdent = ir.NewLocalIdent(stmt.Name)
		alloca = newAlloca
		g.variables[stmt.Name] = alloca
	}

	g.builder.NewStore(val, alloca)
	return nil
}

// generateReturnStmt generates LLVM IR for a return statement
func (g *Generator) generateReturnStmt(stmt *parser.ReturnStmt) error {
	if stmt.Value != nil {
		val, err := g.generateExpression(stmt.Value)
		if err != nil {
			return err
		}
		g.builder.NewRet(val)
	} else {
		g.builder.NewRet(nil)
	}
	return nil
}

// generateIfStmt generates LLVM IR for an if statement
func (g *Generator) generateIfStmt(stmt *parser.IfStmt) error {
	// Generate condition
	cond, err := g.generateExpression(stmt.Condition)
	if err != nil {
		return err
	}

	// Convert condition to boolean (non-zero = true)
	condType := g.analyzer.GetExprType(stmt.Condition)
	var boolCond value.Value
	if condType.Equals(types.IntType) {
		zero := constant.NewInt(llvmTypes.I32, 0)
		boolCond = g.builder.NewICmp(enum.IPredNE, cond, zero)
	} else if condType.Equals(types.FloatType) {
		zero := constant.NewFloat(llvmTypes.Float, 0.0)
		boolCond = g.builder.NewFCmp(enum.FPredONE, cond, zero)
	}

	// Create basic blocks
	thenBlock := g.currentFunc.NewBlock("then")
	mergeBlock := g.currentFunc.NewBlock("merge")
	var elseBlock *ir.Block
	if stmt.ElseBranch != nil {
		elseBlock = g.currentFunc.NewBlock("else")
		g.builder.NewCondBr(boolCond, thenBlock, elseBlock)
	} else {
		g.builder.NewCondBr(boolCond, thenBlock, mergeBlock)
	}

	// Generate then branch
	g.builder = thenBlock
	if err := g.generateBlockStmt(stmt.ThenBranch); err != nil {
		return err
	}
	if g.builder.Term == nil {
		g.builder.NewBr(mergeBlock)
	}

	// Generate else branch if exists
	if stmt.ElseBranch != nil {
		g.builder = elseBlock
		if err := g.generateBlockStmt(stmt.ElseBranch); err != nil {
			return err
		}
		if g.builder.Term == nil {
			g.builder.NewBr(mergeBlock)
		}
	}

	// Continue with merge block
	g.builder = mergeBlock
	return nil
}

// generateWhileStmt generates LLVM IR for a while loop
func (g *Generator) generateWhileStmt(stmt *parser.WhileStmt) error {
	// Create basic blocks
	condBlock := g.currentFunc.NewBlock("while.cond")
	bodyBlock := g.currentFunc.NewBlock("while.body")
	endBlock := g.currentFunc.NewBlock("while.end")

	// Branch to condition
	g.builder.NewBr(condBlock)

	// Generate condition
	g.builder = condBlock
	cond, err := g.generateExpression(stmt.Condition)
	if err != nil {
		return err
	}

	// Convert condition to boolean
	condType := g.analyzer.GetExprType(stmt.Condition)
	var boolCond value.Value
	if condType.Equals(types.IntType) {
		zero := constant.NewInt(llvmTypes.I32, 0)
		boolCond = g.builder.NewICmp(enum.IPredNE, cond, zero)
	} else if condType.Equals(types.FloatType) {
		zero := constant.NewFloat(llvmTypes.Float, 0.0)
		boolCond = g.builder.NewFCmp(enum.FPredONE, cond, zero)
	}

	g.builder.NewCondBr(boolCond, bodyBlock, endBlock)

	// Generate body
	g.builder = bodyBlock
	if err := g.generateBlockStmt(stmt.Body); err != nil {
		return err
	}
	if g.builder.Term == nil {
		g.builder.NewBr(condBlock)
	}

	// Continue with end block
	g.builder = endBlock
	return nil
}

// generateExpression generates LLVM IR for an expression
func (g *Generator) generateExpression(expr parser.Expression) (value.Value, error) {
	switch e := expr.(type) {
	case *parser.IntLiteral:
		return constant.NewInt(llvmTypes.I32, e.Value), nil
	case *parser.FloatLiteral:
		return constant.NewFloat(llvmTypes.Float, e.Value), nil
	case *parser.StringLiteral:
		return g.generateStringConstant(e.Value), nil
	case *parser.Identifier:
		return g.generateIdentifier(e)
	case *parser.BinaryExpr:
		return g.generateBinaryExpr(e)
	case *parser.UnaryExpr:
		return g.generateUnaryExpr(e)
	case *parser.CallExpr:
		return g.generateCallExpr(e)
	default:
		return nil, fmt.Errorf("unknown expression type")
	}
}

// generateStringConstant generates a global string constant
// 参数 str: 字符串内容
// 返回值: 指向字符串常量的指针（i8*）
func (g *Generator) generateStringConstant(str string) value.Value {
	// 创建字符串常量名（例如：.str.0, .str.1, ...）
	name := fmt.Sprintf(".str.%d", g.stringCount)
	g.stringCount++

	// 创建全局字符串常量（需要添加 \00 结尾）
	strWithNull := str + "\x00"
	arrayType := llvmTypes.NewArray(uint64(len(strWithNull)), llvmTypes.I8)
	globalStr := g.module.NewGlobalDef(name, constant.NewCharArrayFromString(strWithNull))
	globalStr.Linkage = enum.LinkagePrivate
	globalStr.UnnamedAddr = enum.UnnamedAddrUnnamedAddr

	// 返回指向字符串首元素的指针（getelementptr）
	zero := constant.NewInt(llvmTypes.I64, 0)
	return constant.NewGetElementPtr(arrayType, globalStr, zero, zero)
}

// generateIdentifier generates LLVM IR for an identifier
func (g *Generator) generateIdentifier(ident *parser.Identifier) (value.Value, error) {
	alloca, exists := g.variables[ident.Name]
	if !exists {
		return nil, fmt.Errorf("undefined variable: %s", ident.Name)
	}
	return g.builder.NewLoad(alloca.Type().(*llvmTypes.PointerType).ElemType, alloca), nil
}

// generateBinaryExpr generates LLVM IR for a binary expression
func (g *Generator) generateBinaryExpr(expr *parser.BinaryExpr) (value.Value, error) {
	left, err := g.generateExpression(expr.Left)
	if err != nil {
		return nil, err
	}

	right, err := g.generateExpression(expr.Right)
	if err != nil {
		return nil, err
	}

	exprType := g.analyzer.GetExprType(expr.Left)

	if exprType.Equals(types.IntType) {
		return g.generateIntBinaryOp(left, right, expr.Operator)
	} else if exprType.Equals(types.FloatType) {
		return g.generateFloatBinaryOp(left, right, expr.Operator)
	}

	return nil, fmt.Errorf("unsupported binary operation type")
}

// generateIntBinaryOp generates LLVM IR for integer binary operations
func (g *Generator) generateIntBinaryOp(left, right value.Value, op string) (value.Value, error) {
	switch op {
	case "+":
		return g.builder.NewAdd(left, right), nil
	case "-":
		return g.builder.NewSub(left, right), nil
	case "*":
		return g.builder.NewMul(left, right), nil
	case "/":
		return g.builder.NewSDiv(left, right), nil
	case "%":
		return g.builder.NewSRem(left, right), nil
	case "==":
		return g.builder.NewICmp(enum.IPredEQ, left, right), nil
	case "!=":
		return g.builder.NewICmp(enum.IPredNE, left, right), nil
	case "<":
		return g.builder.NewICmp(enum.IPredSLT, left, right), nil
	case "<=":
		return g.builder.NewICmp(enum.IPredSLE, left, right), nil
	case ">":
		return g.builder.NewICmp(enum.IPredSGT, left, right), nil
	case ">=":
		return g.builder.NewICmp(enum.IPredSGE, left, right), nil
	case "&&":
		return g.builder.NewAnd(left, right), nil
	case "||":
		return g.builder.NewOr(left, right), nil
	default:
		return nil, fmt.Errorf("unknown integer binary operator: %s", op)
	}
}

// generateFloatBinaryOp generates LLVM IR for float binary operations
func (g *Generator) generateFloatBinaryOp(left, right value.Value, op string) (value.Value, error) {
	switch op {
	case "+":
		return g.builder.NewFAdd(left, right), nil
	case "-":
		return g.builder.NewFSub(left, right), nil
	case "*":
		return g.builder.NewFMul(left, right), nil
	case "/":
		return g.builder.NewFDiv(left, right), nil
	case "==":
		return g.builder.NewFCmp(enum.FPredOEQ, left, right), nil
	case "!=":
		return g.builder.NewFCmp(enum.FPredONE, left, right), nil
	case "<":
		return g.builder.NewFCmp(enum.FPredOLT, left, right), nil
	case "<=":
		return g.builder.NewFCmp(enum.FPredOLE, left, right), nil
	case ">":
		return g.builder.NewFCmp(enum.FPredOGT, left, right), nil
	case ">=":
		return g.builder.NewFCmp(enum.FPredOGE, left, right), nil
	default:
		return nil, fmt.Errorf("unsupported float binary operator: %s", op)
	}
}

// generateUnaryExpr generates LLVM IR for a unary expression
func (g *Generator) generateUnaryExpr(expr *parser.UnaryExpr) (value.Value, error) {
	operand, err := g.generateExpression(expr.Operand)
	if err != nil {
		return nil, err
	}

	exprType := g.analyzer.GetExprType(expr.Operand)

	switch expr.Operator {
	case "-":
		if exprType.Equals(types.IntType) {
			zero := constant.NewInt(llvmTypes.I32, 0)
			return g.builder.NewSub(zero, operand), nil
		} else if exprType.Equals(types.FloatType) {
			zero := constant.NewFloat(llvmTypes.Float, 0.0)
			return g.builder.NewFSub(zero, operand), nil
		}
	case "!":
		if exprType.Equals(types.IntType) {
			zero := constant.NewInt(llvmTypes.I32, 0)
			return g.builder.NewICmp(enum.IPredEQ, operand, zero), nil
		}
	}

	return nil, fmt.Errorf("unsupported unary operator: %s", expr.Operator)
}

// generateCallExpr generates LLVM IR for a function call
func (g *Generator) generateCallExpr(expr *parser.CallExpr) (value.Value, error) {
	// 特殊处理内建的 print 函数
	if expr.Function == "print" {
		return g.generatePrintCall(expr)
	}

	fn, exists := g.functions[expr.Function]
	if !exists {
		return nil, fmt.Errorf("undefined function: %s", expr.Function)
	}

	var args []value.Value
	for _, arg := range expr.Arguments {
		val, err := g.generateExpression(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, val)
	}

	return g.builder.NewCall(fn, args...), nil
}

// generatePrintCall 生成 print 函数调用的 LLVM IR
// print 是内建函数，根据参数类型自动选择调用 printf 或 puts
func (g *Generator) generatePrintCall(expr *parser.CallExpr) (value.Value, error) {
	if len(expr.Arguments) == 0 {
		return nil, fmt.Errorf("print requires at least one argument")
	}

	// 生成第一个参数
	firstArg, err := g.generateExpression(expr.Arguments[0])
	if err != nil {
		return nil, err
	}

	// 如果只有一个参数且是字符串，使用 puts
	if len(expr.Arguments) == 1 {
		if _, ok := expr.Arguments[0].(*parser.StringLiteral); ok {
			return g.builder.NewCall(g.putsFunc, firstArg), nil
		}
		// 如果是整数，使用 printf 格式化输出
		if _, ok := expr.Arguments[0].(*parser.IntLiteral); ok {
			formatStr := g.generateStringConstant("%d\n")
			return g.builder.NewCall(g.printfFunc, formatStr, firstArg), nil
		}
		// 如果是浮点数，使用 printf 格式化输出
		if _, ok := expr.Arguments[0].(*parser.FloatLiteral); ok {
			formatStr := g.generateStringConstant("%f\n")
			return g.builder.NewCall(g.printfFunc, formatStr, firstArg), nil
		}
		// 对于变量，根据类型判断
		if ident, ok := expr.Arguments[0].(*parser.Identifier); ok {
			exprType := g.analyzer.GetExprType(ident)
			if exprType.Equals(types.IntType) {
				formatStr := g.generateStringConstant("%d\n")
				return g.builder.NewCall(g.printfFunc, formatStr, firstArg), nil
			} else if exprType.Equals(types.FloatType) {
				formatStr := g.generateStringConstant("%f\n")
				return g.builder.NewCall(g.printfFunc, formatStr, firstArg), nil
			}
		}
	}

	// 多个参数时，使用 printf（第一个参数必须是格式字符串）
	if _, ok := expr.Arguments[0].(*parser.StringLiteral); ok {
		var args []value.Value
		args = append(args, firstArg)
		for i := 1; i < len(expr.Arguments); i++ {
			arg, err := g.generateExpression(expr.Arguments[i])
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		}
		return g.builder.NewCall(g.printfFunc, args...), nil
	}

	return nil, fmt.Errorf("unsupported print arguments")
}

// getLLVMType converts a type string to an LLVM type
func (g *Generator) getLLVMType(typeStr string) (llvmTypes.Type, error) {
	switch typeStr {
	case "int":
		return llvmTypes.I32, nil
	case "float":
		return llvmTypes.Float, nil
	case "void":
		return llvmTypes.Void, nil
	default:
		return nil, fmt.Errorf("unknown type: %s", typeStr)
	}
}

// typeToLLVMType converts a Type to an LLVM type
func (g *Generator) typeToLLVMType(t types.Type) (llvmTypes.Type, error) {
	if t.Equals(types.IntType) {
		return llvmTypes.I32, nil
	} else if t.Equals(types.FloatType) {
		return llvmTypes.Float, nil
	}
	return nil, fmt.Errorf("unknown type: %s", t.String())
}
