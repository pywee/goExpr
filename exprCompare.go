package lit

import (
	"strconv"

	"github.com/pywee/lit/global"
)

func (r *expression) parseComparison(left, right []*global.Structure, tok string, innerVar global.InnerVar) (*global.Structure, error) {
	var (
		err     error
		leftRv  *global.Structure
		rightRv *global.Structure
	)

	if len(left) == 1 && left[0].Tok != "IDENT" {
		leftRv = left[0]
	} else if leftRv, err = r.parse(left, innerVar); err != nil {
		return nil, err
	}

	if len(right) == 1 && right[0].Tok != "IDENT" {
		rightRv = right[0]
	} else if rightRv, err = r.parse(right, innerVar); err != nil {
		return nil, err
	}

	if tok == "==" {
		if compareEqual(leftRv, rightRv) {
			return &global.Structure{Tok: "BOOL", Lit: "true"}, nil
		}
	}

	if tok == "!=" {
		if compareNotEqual(leftRv, rightRv) {
			return &global.Structure{Tok: "BOOL", Lit: "true"}, nil
		}
	}

	var ok bool
	if ok, err = compareGreaterLessEqual(tok, leftRv, rightRv); err != nil {
		return nil, err
	}
	if ok {
		return &global.Structure{Tok: "BOOL", Lit: "true"}, nil
	}
	return &global.Structure{Tok: "BOOL", Lit: "false"}, nil
}

// type parseComparisonStruct struct {
// 	tok      string
// 	innerVar global.InnerVar
// 	expr     []*global.Structure
// }

// // 解析比较运算 == >= <= != > < === !==
// func (r *expression) parseComparison2(i int, arg *parseComparisonStruct) (*global.Structure, error) {
// 	var (
// 		ok       bool
// 		err      error
// 		left     *global.Structure
// 		right    *global.Structure
// 		tok      = arg.tok
// 		expr     = arg.expr
// 		innerVar = arg.innerVar
// 	)

// 	if left, err = r.parse(expr, innerVar); err != nil {
// 		return nil, err
// 	}
// 	if right, err = r.parse(expr[i+1:], innerVar); err != nil {
// 		return nil, err
// 	}
// 	if tok == "==" && compareEqual(left, right) {
// 		return &global.Structure{Tok: "BOOL", Lit: "true"}, nil
// 	}
// 	if tok == "!=" && compareNotEqual(left, right) {
// 		return &global.Structure{Tok: "BOOL", Lit: "true"}, nil
// 	}
// 	if ok, err = compareGreaterLessEqual(tok, left, right); err != nil {
// 		return nil, err
// 	}
// 	if ok {
// 		return &global.Structure{Tok: "BOOL", Lit: "true"}, nil
// 	}
// 	return &global.Structure{Tok: "BOOL", Lit: "false"}, nil
// }

// compareEqual 比较符: ==
func compareEqual(left, right *global.Structure) bool {
	var (
		lTok = left.Tok
		rTok = right.Tok
		lLit = left.Lit
		rLit = right.Lit
	)

	// 两边都是布尔值
	if lTok == "BOOL" && lTok == rTok {
		return lLit == rLit
	}
	// 两边都是字符串
	if lTok == "STRING" && lTok == rTok {
		return lLit == rLit
	}
	// 两边都是整型
	if lTok == "INT" && lTok == rTok {
		return lLit == rLit
	}

	l, r, err := changeTypeToCompare(left, right)
	if err != nil {
		return false
	}

	// FIXME 未做类型断言
	return l == r
}

// compareNotEqual 比较符: !=
func compareNotEqual(left, right *global.Structure) bool {
	var (
		lTok = left.Tok
		rTok = right.Tok
		lLit = left.Lit
		rLit = right.Lit
	)

	// 两边都是布尔值
	if lTok == "BOOL" && lTok == rTok {
		return lLit != rLit
	}
	// 两边都是字符串
	if lTok == "STRING" && lTok == rTok {
		return lLit != rLit
	}
	// 两边都是整型
	if lTok == "INT" && lTok == rTok {
		return lLit != rLit
	}

	l, r, err := changeTypeToCompare(left, right)
	if err != nil {
		return false
	}

	// 未做类型断言
	return l != r
}

// compareGreaterLessEqual
// 比较符: > < >= <=
func compareGreaterLessEqual(syn string, left, right *global.Structure) (bool, error) {
	l, r, err := changeTypeToCompare(left, right)
	if err != nil {
		return false, err
	}

	if left.Tok != right.Tok || (left.Tok != "INT" && left.Tok != "FLOAT") {
		return false, nil
	}

	var ok bool
	if syn == ">" {
		if left.Tok == "FLOAT" {
			ok = l.(float64) > r.(float64)
		} else if left.Tok == "INT" {
			ok = l.(int64) > r.(int64)
		}
	} else if syn == "<" {
		if left.Tok == "FLOAT" {
			ok = l.(float64) < r.(float64)
		} else if left.Tok == "INT" {
			ok = l.(int64) < r.(int64)
		}
	} else if syn == ">=" {
		if left.Tok == "FLOAT" {
			ok = l.(float64) >= r.(float64)
		} else if left.Tok == "INT" {
			ok = l.(int64) >= r.(int64)
		}
	} else if syn == "<=" {
		if left.Tok == "FLOAT" {
			ok = l.(float64) <= r.(float64)
		} else if left.Tok == "INT" {
			ok = l.(int64) <= r.(int64)
		}
	}
	return ok, nil
}

// changeTypeToCompare 针对 > < >= <= 做统一处理
// 处理逻辑是一样的
func changeTypeToCompare(left, right *global.Structure) (interface{}, interface{}, error) {
	var err error

	if left.Tok == "STRING" {
		if err = global.TransformTokTypeStringToTypeIntOrFloat(left); err != nil {
			return nil, nil, err
		}
	} else if left.Tok == "BOOL" {
		if left, err = global.TransformBoolToInt(left); err != nil {
			return nil, nil, err
		}
	}

	if right.Tok == "STRING" {
		if err = global.TransformTokTypeStringToTypeIntOrFloat(right); err != nil {
			return nil, nil, err
		}
	} else if right.Tok == "BOOL" {
		if right, err = global.TransformBoolToInt(right); err != nil {
			return nil, nil, err
		}
	}

	if left.Tok == "INT" && right.Tok == "FLOAT" {
		left.Tok = "FLOAT"
	} else if left.Tok == "FLOAT" && right.Tok == "INT" {
		right.Tok = "FLOAT"
	}

	var (
		l interface{}
		r interface{}
	)
	if l, err = formatValueTypeToCompare(left); err != nil {
		return nil, nil, err
	}
	if r, err = formatValueTypeToCompare(right); err != nil {
		return nil, nil, err
	}
	return l, r, nil
}

// formatValueTypeToCompare 进行弱类型转换用于比较运算
// 如 "1.1">1.1 正常返回
// "你好" > 1.1 将返回错误 不能将非数字类的字符串进行比较运算
func formatValueTypeToCompare(src *global.Structure) (interface{}, error) {
	var (
		v   interface{}
		err error
	)

	if src.Tok == "STRING" {
		if err = global.TransformTokTypeStringToTypeIntOrFloat(src); err != nil {
			return nil, err
		}
	} else if src.Tok == "BOOL" {
		if src, err = global.TransformBoolToInt(src); err != nil {
			return nil, err
		}
	}

	if src.Tok == "FLOAT" {
		if v, err = strconv.ParseFloat(src.Lit, 64); err != nil {
			return nil, err
		}
	}
	if src.Tok == "INT" {
		if v, err = strconv.ParseInt(src.Lit, 10, 64); err != nil {
			return nil, err
		}
	}
	return v, nil
}

// parseCompare 比较运算
// == >= <= != > < === !==
// func (r *expression) parseCompare(expr []*global.Structure, pos string) (*global.Structure, error) {
// 	var (
// 		err     error
// 		rvLeft  *global.Structure
// 		rvRight *global.Structure
// 	)
// 	for k, v := range expr {
// 		if v.Tok == "==" {
// 			if rvLeft, err = r.parse(expr[:k], pos, nil); err != nil {
// 				return nil, err
// 			}
// 			if rvRight, err = r.parse(expr[k+1:], pos, nil); err != nil {
// 				return nil, err
// 			}
// 			if compareEqual(rvLeft, rvRight) {
// 				return &global.Structure{Tok: "BOOL", Lit: "true"}, nil
// 			}
// 			return &global.Structure{Tok: "BOOL", Lit: "false"}, nil
// 		}
// 		if v.Tok == "!=" {
// 			if rvLeft, err = r.parse(expr[:k], pos, nil); err != nil {
// 				return nil, err
// 			}
// 			if rvRight, err = r.parse(expr[k+1:], pos, nil); err != nil {
// 				return nil, err
// 			}
// 			if compareNotEqual(rvLeft, rvRight) {
// 				return &global.Structure{Tok: "BOOL", Lit: "true"}, nil
// 			}
// 			return &global.Structure{Tok: "BOOL", Lit: "false"}, nil
// 		}
// 		if inArray(v.Tok, []string{">", "<", ">=", "<="}) != "" {
// 			if rvLeft, err = r.parse(expr[:k], pos, nil); err != nil {
// 				return nil, err
// 			}
// 			if rvRight, err = r.parse(expr[k+1:], pos, nil); err != nil {
// 				return nil, err
// 			}
// 			if ok, err := compareGreaterLessEqual(v.Tok, rvLeft, rvRight); err != nil {
// 				return nil, err
// 			} else if ok {
// 				return &global.Structure{Tok: "BOOL", Lit: "true"}, nil
// 			}
// 			return &global.Structure{Tok: "BOOL", Lit: "false"}, nil
// 		}
// 	}
// 	return nil, nil
// }
