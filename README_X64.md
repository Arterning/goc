# GoC x86-64 Native Backend

## 概述

GoC 编译器现在支持直接生成 x86-64 机器码的原生后端！你可以选择两种后端：

1. **LLVM 后端**（默认）：生成 LLVM IR，使用 Clang 编译
2. **x64 后端**（新增）：直接生成 x86-64 汇编代码

## 优点

### x64 原生后端的优势
- ✅ **完全用 Go 实现**：无需安装 LLVM/Clang
- ✅ **编译器体积小**：生成的编译器只有几 MB
- ✅ **完全可控**：所有代码生成逻辑都在你的掌控之中
- ✅ **跨平台支持**：自动适配 Windows 和 Linux 调用约定
- ✅ **学习友好**：可以查看生成的汇编代码，理解编译器工作原理

## 使用方法

### 编译程序

#### 使用默认的 LLVM 后端：
```bash
goc examples/hello.goc
```

#### 使用 x64 原生后端：
```bash
goc -backend=x64 examples/hello.goc
```

### 输出文件

- **LLVM 后端**：
  - `hello.ll` - LLVM IR 中间代码
  - `hello.exe` - 可执行文件

- **x64 后端**：
  - `hello.asm` - NASM 格式的汇编代码
  - `hello.exe` - 可执行文件（需要 NASM 和链接器）

## 系统要求

### LLVM 后端
- **必需**：Clang 编译器

### x64 后端
- **必需**：NASM 汇编器
- **必需**：链接器
  - Windows：`ld` (MinGW) 或 `link.exe` (MSVC)
  - Linux：`ld`

### 安装 NASM

#### Windows (使用 Chocolatey)：
```bash
choco install nasm
```

#### Windows (手动安装)：
1. 下载 NASM：https://www.nasm.us/
2. 解压并添加到 PATH

#### Linux (Ubuntu/Debian)：
```bash
sudo apt-get install nasm
```

#### Linux (Fedora/RHEL)：
```bash
sudo dnf install nasm
```

## 生成的汇编代码示例

对于这个简单的程序：

```c
int main() {
    x = 42
    return x
}
```

x64 后端生成的汇编代码：

```asm
bits 64
default rel

section .text
global main

; Function: main
main:
    push rbp
    mov rbp, rsp
    sub rsp, 256

    ; x = 42
    mov eax, 42
    mov [rbp-8], eax

    ; return x
    mov eax, [rbp-8]

    ; 函数尾声
    mov rsp, rbp
    pop rbp
    ret
```

## 技术特性

### 已实现的功能

✅ **表达式**
- 整数和浮点字面量
- 变量引用
- 二元运算（+, -, *, /, %, ==, !=, <, <=, >, >=, &&, ||）
- 一元运算（-, !）
- 函数调用

✅ **语句**
- 变量声明（显式类型和类型推断）
- 赋值语句
- If/Else 控制流
- While 循环
- Return 语句
- 代码块

✅ **函数**
- 函数定义
- 函数参数（最多 4 个，Windows x64 调用约定）
- 函数调用

✅ **类型系统**
- int (32-bit)
- float (32-bit，使用 SSE 指令）

✅ **调用约定**
- Windows x64：RCX, RDX, R8, R9
- System V AMD64 (Linux)：RDI, RSI, RDX, RCX, R8, R9

## 架构设计

### 编译流程

```
源代码 (.goc)
    ↓
词法分析 (Lexer)
    ↓
语法分析 (Parser)
    ↓
语义分析 (Semantic Analyzer)
    ↓
代码生成 (x64 CodeGenerator)
    ↓
NASM 汇编 (.asm)
    ↓
汇编器 (NASM)
    ↓
目标文件 (.o)
    ↓
链接器 (ld/link.exe)
    ↓
可执行文件 (.exe)
```

### 寄存器分配策略

当前实现使用**简单栈式分配**：
- 所有变量存储在栈上
- 寄存器仅用于临时计算：
  - `RAX/EAX`：整数运算结果
  - `XMM0`：浮点运算结果
  - `XMM1`：浮点运算临时值
  - `RBP`：栈帧基址
  - `RSP`：栈指针

### 目录结构

```
goc/
├── main.go           # 编译器入口
├── lexer/            # 词法分析器
├── parser/           # 语法分析器
├── semantic/         # 语义分析器
├── codegen/          # LLVM IR 代码生成器
├── x64/              # x86-64 原生后端（新增）
│   ├── asm.go        # 汇编指令表示
│   ├── codegen.go    # x86-64 代码生成器
│   └── executable.go # 可执行文件构建器
└── types/            # 类型系统
```

## 性能对比

### 编译器大小
- **LLVM 后端**：需要安装完整的 Clang/LLVM（数百 MB）
- **x64 后端**：仅需 GoC 编译器（<10 MB）+ NASM（<5 MB）

### 编译速度
- **LLVM 后端**：中等（LLVM 优化需要时间）
- **x64 后端**：快速（直接生成代码，无优化）

### 生成代码质量
- **LLVM 后端**：高度优化
- **x64 后端**：未优化（适合学习和调试）

## 未来改进

可能的增强功能：

1. **寄存器分配优化**
   - 实现基于使用计数的寄存器分配
   - 实现图着色算法

2. **直接生成机器码**
   - 跳过 NASM，直接编码 x86-64 指令
   - 直接生成 PE/ELF 文件格式

3. **优化**
   - 常量折叠
   - 死代码消除
   - 寄存器分配优化

4. **更多特性**
   - 支持更多参数（栈传参）
   - 支持数组和指针
   - 支持结构体

## 示例程序

### 1. 简单返回值

```c
int main() {
    return 42
}
```

编译：`goc -backend=x64 examples/hello.goc`

### 2. 算术运算

```c
int main() {
    int x = 10
    int y = 32
    int result = x + y
    return result
}
```

### 3. 斐波那契数列

```c
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

## 常见问题

### Q: 为什么还需要 NASM？

A: 当前实现生成 NASM 汇编代码作为中间步骤。未来版本可以直接生成机器码，完全消除对外部工具的依赖。

### Q: 支持哪些操作系统？

A: 当前支持 Windows 和 Linux x86-64。会自动检测操作系统并使用正确的调用约定。

### Q: 生成的代码有多快？

A: 生成的代码未经优化，性能不如 LLVM 后端。但对于学习编译器原理和小型项目来说已经足够。

### Q: 如何查看生成的汇编代码？

A: 汇编代码保存在 `.asm` 文件中，你可以用任何文本编辑器查看。

## 贡献

欢迎贡献！特别是：
- 优化改进
- 错误修复
- 文档完善
- 示例程序

## 许可证

[根据你的项目指定]

## 致谢

本项目实现了一个完整的编译器后端，包括：
- x86-64 指令生成
- 调用约定处理
- 栈帧管理
- SSE 浮点支持
