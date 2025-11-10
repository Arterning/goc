# 嵌入式工具目录

请将以下工具放入此目录：

## Windows
1. **nasm.exe** - NASM 汇编器
   - 下载：https://www.nasm.us/pub/nasm/releasebuilds/2.16.03/win64/nasm-2.16.03-win64.zip
   - 解压后将 `nasm.exe` 复制到此目录

## Linux
1. **nasm** - NASM 汇编器
   - 下载：https://www.nasm.us/pub/nasm/releasebuilds/2.16.03/linux/
   - 或从系统安装后复制 `/usr/bin/nasm` 到此目录

## 目录结构

```
embedded/tools/
├── README.md       # 本文件
├── nasm.exe        # Windows NASM (需要下载)
└── nasm            # Linux NASM (需要下载)
```

## 下载说明

### 方法 1: 使用提供的脚本（推荐）

运行项目中的 `download-tools.sh` (Linux/Mac) 或 `download-tools.bat` (Windows)

### 方法 2: 手动下载

1. 访问 NASM 官网：https://www.nasm.us/
2. 进入 Downloads 页面
3. 下载对应平台的版本
4. 解压并将可执行文件复制到此目录

## 文件大小参考

- nasm.exe: ~400 KB
- nasm (Linux): ~500 KB

总共约 1 MB，不会显著增加编译器体积。
