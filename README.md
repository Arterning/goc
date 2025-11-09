# GoC - 类 C 语言编译器

GoC 是一个用 Go 语言编写的类 C 语言编译器，使用 LLVM 后端生成机器码。

## 特性

### MVP 阶段功能

- ✅ **类型推断**：支持变量类型自动推断（`x = 42` 自动推断为 `int`）
- ✅ **可选分号**：换行即语句结束，分号可选
- ✅ **基本类型**：`int`、`float`
- ✅ **算术运算**：`+`、`-`、`*`、`/`、`%`
- ✅ **比较运算**：`==`、`!=`、`<`、`<=`、`>`、`>=`
- ✅ **逻辑运算**：`&&`、`||`、`!`
- ✅ **函数定义和调用**（显式类型签名）
- ✅ **控制流**：`if`/`else`、`while`
- ✅ **LLVM IR 代码生成**

## 语法特点

### 1. 类型推断

```c
// 传统 C 语法（仍然支持）
int x = 10;

// 类型推断（新特性）
x = 10  // 自动推断为 int
y = 3.14  // 自动推断为 float
```

### 2. 可选分号

```c
// 换行即语句结束
x = 10
y = 20

// 同行多语句需要分号
x = 10; y = 20
```

### 3. 变量声明规则

- **只能声明一次**：`int x; int x = 10;` ❌ 错误
- **类型一致性**：`x = 10; x = 3.14;` ❌ 错误（类型不匹配）
- **先声明后使用**：必须 `int x` 或 `x = 值`，不能只写 `x`

### 4. 函数签名

函数必须显式声明参数和返回值类型：

```c
int add(int a, int b) {
    return a + b
}
```

## 安装

### 前置要求

1. **Go 1.18+**
2. **Clang**（用于将 LLVM IR 编译为可执行文件）

### 构建编译器

```bash
cd goc
go build -o goc.exe
```

## 使用方法

### 编译源文件

```bash
./goc examples/hello.goc
```

这将生成：
- `hello.ll` - LLVM IR 中间代码
- `hello.exe` - 可执行文件

### 运行程序

```bash
./examples/hello.exe
echo $?  # 查看返回值
```

## 示例程序

### Hello World

```c
// examples/hello.goc
int main() {
    x = 42
    return x
}
```

### 斐波那契数列

```c
// examples/fibonacci.goc
int fib(int n) {
    if (n <= 1) {
        return n
    }
    a = fib(n - 1)
    b = fib(n - 2)
    return a + b
}

int main() {
    result = fib(10)
    return result
}
```

### While 循环求和

```c
// examples/loop.goc
int main() {
    sum = 0
    i = 1

    while (i <= 10) {
        sum = sum + i
        i = i + 1
    }

    return sum
}
```

## 编译器架构

```
源代码 (.goc)
    ↓
词法分析 (Lexer)
    ↓
语法分析 (Parser)
    ↓
抽象语法树 (AST)
    ↓
语义分析 (Semantic Analyzer)
    ↓
代码生成 (LLVM IR Generator)
    ↓
LLVM IR (.ll)
    ↓
Clang 编译
    ↓
可执行文件 (.exe)
```

## 项目结构

```
goc/
├── main.go           # 编译器入口
├── lexer/            # 词法分析
│   ├── token.go      # Token 定义
│   └── lexer.go      # 词法分析器
├── parser/           # 语法分析
│   ├── ast.go        # AST 节点定义
│   └── parser.go     # 语法分析器
├── semantic/         # 语义分析
│   └── analyzer.go   # 语义分析器
├── codegen/          # LLVM IR 代码生成
│   └── generator.go  # 代码生成器
├── types/            # 类型系统
│   └── types.go      # 类型定义和检查
└── examples/         # 示例程序
    ├── hello.goc
    ├── arithmetic.goc
    ├── fibonacci.goc
    ├── loop.goc
    └── float.goc
```

## 开发计划

### 已完成 ✅
- [x] 词法分析
- [x] 语法分析
- [x] 类型系统
- [x] 语义分析
- [x] LLVM IR 代码生成
- [x] 基本类型（int、float）
- [x] 类型推断
- [x] 控制流（if/else、while）
- [x] 函数定义和调用

### 未来扩展 🚀
- [ ] 数组支持
- [ ] 指针支持
- [ ] 结构体（struct）
- [ ] for 循环
- [ ] 更多类型（char、bool、double）
- [ ] 标准库集成
- [ ] 优化 LLVM IR
- [ ] 更好的错误提示

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License
