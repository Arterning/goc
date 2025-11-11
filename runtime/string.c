// string.c - GoC 字符串类型运行时函数库
// 提供字符串连接、比较、索引等操作的运行时支持

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// 字符串结构体定义（与 LLVM IR 中的定义对应）
// struct { i8* data, i32 length }
typedef struct {
    char* data;    // 指向 null-terminated 字符数组的指针
    int length;    // 字符串长度（不包括 \0）
} goc_string;

// str_concat - 字符串连接
// 参数: s1, s2 - 要连接的两个字符串
// 返回值: 新的字符串（s1 + s2）
// 注意: 使用 malloc 分配内存，调用者需要管理内存
goc_string str_concat(goc_string s1, goc_string s2) {
    goc_string result;

    // 计算新字符串的长度
    result.length = s1.length + s2.length;

    // 分配内存（包括 null 终止符）
    result.data = (char*)malloc(result.length + 1);
    if (result.data == NULL) {
        fprintf(stderr, "Error: Failed to allocate memory for string concatenation\n");
        exit(1);
    }

    // 复制第一个字符串
    memcpy(result.data, s1.data, s1.length);

    // 复制第二个字符串
    memcpy(result.data + s1.length, s2.data, s2.length);

    // 添加 null 终止符
    result.data[result.length] = '\0';

    return result;
}

// str_eq - 字符串相等比较
// 参数: s1, s2 - 要比较的两个字符串
// 返回值: 1 表示相等，0 表示不相等
int str_eq(goc_string s1, goc_string s2) {
    // 长度不同，必然不相等
    if (s1.length != s2.length) {
        return 0;
    }

    // 使用 memcmp 比较内容
    return memcmp(s1.data, s2.data, s1.length) == 0 ? 1 : 0;
}

// str_ne - 字符串不等比较
// 参数: s1, s2 - 要比较的两个字符串
// 返回值: 1 表示不相等，0 表示相等
int str_ne(goc_string s1, goc_string s2) {
    return !str_eq(s1, s2);
}

// str_index - 字符串索引访问
// 参数: s - 字符串, index - 索引位置
// 返回值: 指定位置的字符（作为 int 返回）
// 注意: 不进行边界检查，调用者需要确保索引有效
int str_index(goc_string s, int index) {
    if (index < 0 || index >= s.length) {
        fprintf(stderr, "Error: String index out of bounds: %d (length: %d)\n", index, s.length);
        exit(1);
    }

    return (int)(unsigned char)s.data[index];
}
