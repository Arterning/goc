// Package types 提供类型系统的定义和类型检查功能
// 类型系统是静态类型语言的核心，负责：
// 1. 定义语言支持的类型
// 2. 检查类型兼容性
// 3. 推断表达式的类型
//
// 本编译器支持的类型：
// - int: 32 位整数
// - float: 32 位浮点数
// - void: 空类型（用于无返回值的函数）
package types

// Type 表示类型的接口
// 所有类型都必须实现这个接口
type Type interface {
	String() string    // 返回类型的字符串表示
	Equals(Type) bool  // 检查两个类型是否相等
}

// BasicType 表示基本类型（int, float, void）
// 使用结构体而不是字符串，便于扩展（如添加指针、数组等）
type BasicType struct {
	Name string // 类型名称
}

// String 返回类型的字符串表示
func (bt *BasicType) String() string {
	return bt.Name
}

// Equals 检查两个类型是否相等
// 参数 other: 要比较的类型
// 返回值: true 表示类型相同，false 表示不同
func (bt *BasicType) Equals(other Type) bool {
	if otherBasic, ok := other.(*BasicType); ok {
		return bt.Name == otherBasic.Name
	}
	return false
}

// ========== 预定义类型常量 ==========
// 使用单例模式，确保每种类型只有一个实例
// 这样可以使用指针比较来检查类型相等

var (
	IntType   = &BasicType{Name: "int"}   // 32 位整数类型
	FloatType = &BasicType{Name: "float"} // 32 位浮点数类型
	VoidType  = &BasicType{Name: "void"}  // 空类型（无返回值）
)

// ========== 类型转换和查询函数 ==========

// FromString 将字符串转换为 Type
// 参数 s: 类型字符串（"int", "float", "void"）
// 返回值: 对应的 Type，如果无法识别则返回 nil
//
// 用途：解析源代码中的类型声明
// 例如："int x" 中的 "int" -> IntType
func FromString(s string) Type {
	switch s {
	case "int":
		return IntType
	case "float":
		return FloatType
	case "void":
		return VoidType
	default:
		return nil // 未知类型
	}
}

// InferFromLiteral 从字面量推断类型
// 参数 tokenType: Token 类型字符串（"INT_LIT", "FLOAT_LIT"）
// 返回值: 推断出的 Type
//
// 用途：类型推断功能
// 例如："x = 42" 中，从 42（INT_LIT）推断 x 为 int 类型
func InferFromLiteral(tokenType string) Type {
	switch tokenType {
	case "INT_LIT":
		return IntType
	case "FLOAT_LIT":
		return FloatType
	default:
		return nil
	}
}

// IsNumeric 检查类型是否为数值类型（int 或 float）
// 参数 t: 要检查的类型
// 返回值: true 表示是数值类型，false 表示不是
//
// 用途：
// - 检查表达式是否可以进行数值运算
// - 检查条件表达式是否有效（本编译器中，条件必须是数值类型）
func IsNumeric(t Type) bool {
	return t.Equals(IntType) || t.Equals(FloatType)
}

// CanAssign 检查是否可以将 from 类型的值赋给 to 类型的变量
// 参数 to: 目标类型（变量的类型）
// 参数 from: 源类型（值的类型）
// 返回值: true 表示可以赋值，false 表示类型不兼容
//
// 类型检查策略：
// - MVP 版本：严格类型检查，类型必须完全匹配
// - 不支持隐式类型转换（例如 int -> float）
//
// 例如：
// - int x = 10;    // OK: int -> int
// - int y = 3.14;  // ERROR: float -> int（类型不匹配）
func CanAssign(to, from Type) bool {
	// 严格类型检查：类型必须完全相等
	return to.Equals(from)
}

// ========== 运算符类型检查 ==========

// GetBinaryOpResultType 确定二元运算的结果类型
// 参数 left, right: 左右操作数的类型
// 参数 op: 运算符（+, -, *, /, %, ==, !=, <, <=, >, >=, &&, ||）
// 返回值: (结果类型, 错误)
//
// 类型规则：
// 1. 操作数类型必须完全匹配（不支持隐式转换）
// 2. 比较运算符（==, !=, <, <=, >, >=）返回 int 类型（0 或 1）
// 3. 算术和逻辑运算符返回操作数的类型
//
// 例如：
// - 1 + 2        -> int（算术运算，返回操作数类型）
// - 3.14 * 2.0   -> float（算术运算，返回操作数类型）
// - x > 10       -> int（比较运算，总是返回 int）
// - 1 + 3.14     -> ERROR（类型不匹配）
func GetBinaryOpResultType(left, right Type, op string) (Type, error) {
	// 检查操作数类型是否匹配
	if !left.Equals(right) {
		return nil, &TypeError{
			Message: "type mismatch in binary operation: " + left.String() + " and " + right.String(),
		}
	}

	// 比较运算符总是返回 int（0 表示 false，1 表示 true，类似 C 语言）
	if isComparisonOp(op) {
		return IntType, nil
	}

	// 算术运算符（+, -, *, /, %）和逻辑运算符（&&, ||）
	// 返回操作数的类型
	return left, nil
}

// GetUnaryOpResultType 确定一元运算的结果类型
// 参数 operand: 操作数的类型
// 参数 op: 运算符（-, !）
// 返回值: (结果类型, 错误)
//
// 类型规则：
// - 一元运算符要求操作数为数值类型（int 或 float）
// - 结果类型与操作数类型相同
//
// 例如：
// - -x     -> 与 x 的类型相同
// - !flag  -> 与 flag 的类型相同
func GetUnaryOpResultType(operand Type, op string) (Type, error) {
	if !IsNumeric(operand) {
		return nil, &TypeError{
			Message: "unary operator " + op + " requires numeric type",
		}
	}
	return operand, nil
}

// isComparisonOp 检查运算符是否为比较运算符
func isComparisonOp(op string) bool {
	return op == "==" || op == "!=" || op == "<" || op == "<=" || op == ">" || op == ">="
}

// ========== 错误类型 ==========

// TypeError 表示类型错误
// 用于报告类型不匹配、类型不兼容等错误
type TypeError struct {
	Message string // 错误消息
}

// Error 实现 error 接口
func (te *TypeError) Error() string {
	return te.Message
}
