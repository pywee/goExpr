# Lit 轻量级解释型弱类型语言 (迭代中...)


Lit 是一个解释型编程语言，目前仍在持续开发中。基于 Golang 开发，还没有<b>正式的名字</b>，暂且叫作 <b>Lit</b> 好了。原本我只打算用于实现对文本中包含的算术表达式进行计算，后来逐步地扩充了更多的特性。索性将它定位为一个解释型编程语言了。

目前我已经实现了一些编程语言必备的基础功能，尤其是比较重要的算术表达式、弱类型转换、内置函数、变量声明、自定义函数等。仍然需要进一步完善。许多的特性参考了 Go、PHP 以及 Node.js。

最终我要将它实现为一个解释型的弱类型编程语言，希望它能用于日常的 Web 开发。至于能否顺利完成，还需要很长时间才能下定论。

关于目前已实现的特性，请看如下文档示例，我将定期更新：

当前文档更新日期是 2022.11.02

---

### 当前已支持的特性

**一、变量 (声明、赋值、拼接、参与算术计算）**

**二、完整的算术表达式 [+ - * / & | ^ %]**

**三、弱类型**

**四、关系运算符 [> < >= <= != == ===]**

**五、逻辑运算符 [&& ||]**

**六、内置函数 (试验阶段)**

**七、自定义函数 (试验阶段)**

---

### 将在后期支持的特性

**一、if else 语句**

**二、数组**

**三、for**

**四、对象或结构体 (从中选择其一实现 会参考golang)**

**五、递归**

**六、基础的动态库 (如 tcp/http/mysql等)**

**七、异常处理**

---

#### 使用方法


```
go get github.com/pywee/lit
```

**一、变量声明**
```golang
import "github.com/pywee/lit"

func main() {
    // 执行以下句子，最终会输出
    src := []byte(`
        a = 123;
        b = a + 456;
        print(b); // 579
    `)
    _, err := lit.NewExpr(src)
}

```

---

**二、算术表达式的计算。算术符号的优先级保持与 Go 语言相同。示例：**

```golang
// 不同的语言符号优先级是不完全一样的，Lit 的算术符号优先级保持与 Golang 一致
// 首先我们执行 Go 语言原生函数进行数学表达式计算
// 以下句子最终会输出 
// +1.600000e+001
// 125
println((2 + 100 ^ 2 - (10*1.1 - 22 + (22 | 11))) / 10 * 2)
println(12/333+31+(5/10)-6|100)

// 使用 Lit 计算文本中的数据
// 表达式文本
// 执行下面的句子 最终会输出
exprs := []byte(`
    a = (2 + 100 ^ 2 - (10*1.1 - 22 + (22 | 11))) / 10 * 2;
    b = 12 / 333 + 31 + (5 / 10) - 6 | 100;
    print(a); // 16
    print(b); // 125
`)
_, err = lit.NewExpr(exprs)

// *** 同样的表达式放在 PHP 中，会输出 -24 ***

```

---

**三、当前已支持部分常用内置函数(测试阶段)，更多的内置函数将在接下来继续完成**

```golang

// 下面的句子调用了两个函数 
// IsInt(arg) 用来检查 arg 是否为整型 
// Replace(arg1, arg2, arg3, arg4) 用来做字符串替换

// 执行下面语句 最终会输出
    exprs := []byte(`
    a = replace("hello word111", "1", "", 2-isInt((1+(1 + isInt(123+(1+2)))-1)+2)-2);
    varDump(a); // STRING hello word
`)
_, err = lit.NewExpr(exprs)

```
---
**四、弱类型转换。弱类型的特性我把它设计为与 PHP 基本一样**
```golang
// 当布尔值参与运算时，底层会将 true 转为 1, false 转为 0
// 执行下面句子 将输出
src := []byte(`
    a = true - 1;
    b = isInt(1);
    c = isFloat(1.0);
    d = false == 0.0;
    e = "false" == 0.0;
    print(a); // 0
    print(b); // true
    print(c); // true
    print(d); // true
    print(e); // true
`)
_, err := lit.NewExpr(src)

src := []byte(`
    a = true - 1;
    b = isInt(1);
    c = isFloat(1.0);
    d = false == 0.0;
    e = "false" == 0.0;
    print(a); // 0
    print(b); // true
    print(c); // true
    print(d); // true
    print(e); // true
`)

exprs := []byte(`
    a = false;
    b = true;
    c = "123";
    d = 456;
    print(a + b); // 1
    print(a >= b); // false
    print(a + b + c + d); // 580
`)
_, err := lit.NewExpr(src)


// 与其他弱类型语言一样
// 字符串数字与整型相操作，在 Lit 的底层会将字符串数字转换为整型
// 执行下面句子 将输出
src := []byte(`
    a = "1" - 1;
    b =  0.0 >= false+1 || (1<=21 && 1==1);
    print(a); // 0
    print(b); // true
`)
_, err := lit.NewExpr(src)


// 与其他弱类型语言一样
// 字符串数字与整型相操作，在 Lit 的底层会将字符串数字转换为整型
// 执行下面句子 将输出
src := []byte(`
    a = "1" - 1;
    print(a); // 0
`)
_, err := lit.NewExpr(src)


// 字符串与字符串相加时 将进行字符串的拼接
// 执行以下句子，将会输出
src := []byte(`
    a = "abc" + "def";
    print(a); // abcdef
`)
_, err := lit.NewExpr(src)


// 但如果当两个字符串都为数字时 对他们进行相加 则会被底层转换为数字
// 执行如下句子，将会输出
    src := []byte(`
    a = "123" + "456";
    print(a); // 579
`)
_, err := lit.NewExpr(src)


// 其他字符串+整型将会报错
// 执行如下句子会报错
src := []byte(`
    a = "abcwwww1230"+0.01;
    print(a); // 报错
`)
_, err := lit.NewExpr(src)

// 在 php 里面，执行上面的句子不但不会报错，还会将字符串 "abcwwww1230" 中的 1230 单独提取出来
// 并且还会将 1230 转换成整型与后面的 0.01 进行计算
// 我不太清楚支持这种操作的目的是什么，个人感觉，这显然没有一点点运用场景
// 所以在 Lit 里面 不允许这样操作 既损耗效率又没有一点点意义

```

---

**五、"并且" 与 "或者" 符号处理**
```golang
// 执行如下句子，将会输出
src := []byte(`
    a = isInt(1) && 72+(11-2) || 1-false;
    varDump(a); // BOOL true
`)
_, err := lit.NewExpr(src)

```
---

**六、自定义函数处理 (试验阶段)**
```golang
// 执行如下句子，将会输出
// 8 123
src := []byte(`
    a = "123";
    func demo(b = 10, m = "2") {
        print(b - m, a);
    }
    demo(); // 8 123
`)
_, err := lit.NewExpr(src)


// 执行如下句子，将会输出 6
src := []byte(`
    func a() {
        return 1;
    }
    func b() {
        return a() + 2;
    }
    print(b()+3); // 6
`)
_, err := lit.NewExpr(src)


// 指定形参默认数据
// 执行如下句子，将会输出
src := []byte(`
    func demo(a, b = 10) {
        return a+1+b;
    }

    a = demo(9);
    print(a); // 20
    print(b); // 找不到变量
`)
_, err := lit.NewExpr(src)

```


---
**请注意，Lit 的算术符号优先级向 Golang 看齐。每个语言对算术符号的优先级处理都有一定区别，如，针对以下表达式进行计算时：**

``` golang
// 2 + 100 ^ 2 - (10*1.1 - 22 + (22 | 11)) / 10 * 2
// PHP 输出 -104
// Node.js 输出 -104
// Golang 输出 96
// Lit 输出 96
```

---

**Lit 算术符号优先级**

第一级  ``` () && || ```

第二级  ``` > < >= <= == != ===```

第三级  ``` * / % ```

第四级 ```| &``` 

第五级  ``` + - ^ ```

---

**当前支持的内置函数有如下，更多函数将会在逐步补充 (当前仍然存在bug）**
通用处理函数
```golang
    print
    varDump
```
---
字符串处理函数
函数的命名基本参考了 Go 语言
除了个别函数有差别，如 
utf8Len 用于检测字符串字数的函数
isNumeric 用于判断当前输入是否为数字

```
trim
trimLeft
trimRight
trimSpace
len
utf8Len
md5
replace
contains
index
lastIndex
toLower
toUpper
toTitle
repeat
```
---
其他函数
```
isNumeric
isBool
isInt
isFloat
```

---

##### 由于它将是一个解释型语言，出于效率考虑，我不会让它支持任何的语法糖，不支持多余的、景上添花的特性，避免在运行时过多影响效率。

##### 得益于 Go 语言原生的一些高级特性，例如 管道、协程 这些可轻松实现异步操作的特性，Lit 会有更多扩展空间，它会比 php 更灵活、更小巧。

##### 敬请期待...
