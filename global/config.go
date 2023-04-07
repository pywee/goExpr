package global

var MathSym = []string{"=", "+=", "-=", "*=", "/=", "^=", "&=", "|=", "%="}

type Block struct {
	// Type [0-未知, 非法定义; 1.变量赋值; 2.函数声明; 3.函数调用; 4.if语句; 5.for; 6.return; 7.变量自增 n++; 8.变量自减 n--; 9.continue; 10.break]
	Type int8
	// 关键词 如果类型是函数 则显示函数名
	Name string
	// Code 代码块
	Code []*Structure
	// ArrayIdx 修改数组指定下标的值时此处不为空
	// 如 arr[0][0] = 1
	// ArrayIdx = []int{0, 3}
	// 使用此结构加快速度
	ArrayIdx [][]*Structure
	// IfExt if 流程控制语句
	IfExt []*ExIf
	// ForExt for流程控制语句
	ForExt *ForExpression
}

type ForExpression struct {
	// Type 循环方式 [1. n=1; n < x; n ++]; 2.range操作; 3.无限循环]
	Type uint8
	// Condtions 循环条件
	Conditions []*Structure
	// Code for语句内部代码块
	Code []*Structure
}

type ExIf struct {
	// Tok 标识
	Tok string
	// Condition if条件
	Condition []*Structure
	// ConditionLen 条件句子长度
	ConditionLen int
	// Body if句子内数据
	// 此处仍会出现if 需要通过递归层层解析
	Body []*Structure
	// BodyLen 数据体长度
	BodyLen int
}

// Structure 基础数据解析 每一个符号代码
type Structure struct {
	Position string
	Tok      string
	Lit      string
	Arr      *Array
}

// Array 数组结构
type Array struct {
	// Name 数组名称 当数组是一个多维数组时 它有可能为空
	// 因为有临时变量存在
	Name string
	// List 数组的每一个元素
	List []*ArrayIdent
}

type ArrayIdent struct {
	InnerKey int
	Name     string
	Values   []*Structure
	Child    *Array
}

type InnerVar map[string]*Structure

func (s *Structure) IsBoolTrue() bool {
	return s.Lit == "true"
}

func (s *Structure) IsBoolFalse() bool {
	return s.Lit == "false"
}

func (s *Structure) Val() string {
	return s.Lit
}

func (s *Structure) Type() string {
	return s.Tok
}

func Index(arr []*Structure, sep string) int {
	for k, v := range arr {
		if v.Lit == sep {
			return k
		}
	}
	return -1
}
