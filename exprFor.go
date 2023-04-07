package lit

import (
	"fmt"
	"regexp"

	"github.com/pywee/lit/global"
	"github.com/pywee/lit/types"
)

// FIXME 未针对for语句的合法性做充分检查
// parseIdentedFOR 解析for语句
func (r *expression) parseIdentedFOR(expr []*global.Structure, blocks []*global.Block, innerVar global.InnerVar, i int) ([]*global.Block, int, error) {
	var (
		// rlen 数据长度
		rlen = len(expr)
		// foundCurlyBracket 花括号标记
		foundCurlyBracket bool
		// curlyBracketCount 计算花括号
		curlyBracketCount uint8
		// conditions for 条件
		conditions = make([]*global.Structure, 0, 10)
		// curlyBracketCode 花括号内的代码块
		curlyBracketCode = make([]*global.Structure, 0, 10)
		// forType 默认循环类型 [1.循环迭代; 2.range; 3.死循环]
		forType uint8 = types.TypeForExpressionIteration
	)

	for j := i; j < rlen; j++ {
		exprJ := expr[j]
		if exprJ.Tok == "{" {
			curlyBracketCount++
			foundCurlyBracket = true
		} else if exprJ.Tok == "}" {
			curlyBracketCount--
		}
		if exprJ.Tok == "range" {
			forType = types.TypeForExpressionRange
		}
		if foundCurlyBracket {
			curlyBracketCode = append(curlyBracketCode, exprJ)
			if curlyBracketCount == 0 {
				i = j
				break
			}
		} else {
			conditions = append(conditions, exprJ)
		}
	}

	blocks = append(blocks, &global.Block{
		Name: "FOR",
		ForExt: &global.ForExpression{
			Type:       forType,
			Conditions: conditions,
			Code:       curlyBracketCode[1 : len(curlyBracketCode)-1],
		},
		Type: types.CodeTypeIdentFOR,
	})
	return blocks, i, nil
}

// execForTypeRange 针对 for range
// for k,v = range arr {}
func (r *expression) execForTypeRange(forExpr *global.ForExpression, innerVar global.InnerVar) error {
	cds := forExpr.Conditions
	clen := len(cds)
	if clen < 3 {
		return types.ErrorForExpression
	}

	rangeIdx := global.Index(cds, "range")
	if rangeIdx == -1 || rangeIdx+1 >= clen {
		return types.ErrorForExpression
	}

	var (
		err    error
		rv     *global.Structure
		arr    = cds[rangeIdx+1:]
		kvs    = cds[1 : rangeIdx-1]
		arrLen = len(arr)
	)

	_ = kvs

	if arrLen == 1 && arr[0].Tok == "IDENT" {
		rv, err = r.parse(arr, innerVar)
	} else {
		arr = append(arr, &global.Structure{Tok: ";", Lit: ";"})
		rv, err = r.initExpr(arr, innerVar, &parsing{isInLoop: true})
	}
	if err != nil {
		return err
	}
	if rv.Tok != "ARRAY" {
		return types.ErrorNotSupportToRange
	}

	// TODO 暂时先写到这里
	thisArrList := rv.Arr.List
	if code := forExpr.Code; len(code) > 0 {
		nkv, err := forKvs(kvs)
		if err != nil {
			return err
		}
		kvLen := len(nkv)
		for i := 0; i < len(thisArrList); i++ {
			if kvLen == 2 {
				innerVar[nkv[0]] = &global.Structure{Lit: fmt.Sprintf("%d", i), Tok: "INT"}
				// innerVar[nkv[1]] = &global.Structure{Lit: fmt.Sprintf("%d", i)}
			}
			// global.Output(code)
			// _, err := r.initExpr(code, innerVar, &parsing{isInLoop: true})
			// if err != nil {
			// 	return err
			// }
		}
	}

	return nil
}

// forKvs 检查 for range 循环中的 key=>value 是否合法
func forKvs(kvs []*global.Structure) ([]string, error) {
	var expr string
	var kv = make([]string, 0, 2)
	re, _ := regexp.Compile(`^[_a-zA-Z]+[_a-zA-Z0-9]*$`)
	for _, v := range kvs {
		if v.Tok == "," {
			if ok := re.MatchString(expr); !ok {
				return nil, types.ErrorForExpression
			}
			kv = append(kv, expr)
			expr = ""
			continue
		}
		if v.Lit == "" {
			expr += v.Tok
		} else {
			expr += v.Lit
		}
	}

	if expr != "" {
		if ok := re.MatchString(expr); !ok {
			return nil, types.ErrorForExpression
		}
		kv = append(kv, expr)
	}

	if len(kv) > 2 {
		return nil, types.ErrorForExpression
	}

	return kv, nil
}

// execFORType1 解析以下形式的 for 流程控制:
// n = 0; n < y; n ++
func (r *expression) execFORType1(forExpr *global.ForExpression, innerVar global.InnerVar) error {
	var (
		lf         int
		cd1        = make([]*global.Structure, 0, 5)
		cd2        = make([]*global.Structure, 0, 5)
		conditions = forExpr.Conditions
	)

	for i := 0; i < len(conditions); i++ {
		if conditions[i].Tok == ";" {
			if len(cd1) == 0 {
				cd1 = conditions[1 : i+1]
			} else {
				cd2 = conditions[len(cd1)+1 : i]
				lf = i
				break
			}
		}
	}

	// n = 0
	if _, err := r.initExpr(cd1, innerVar, nil); err != nil {
		return err
	}

	// n ++
	cd3 := conditions[lf+1:]
	cLen := len(cd3)
	if cLen < 2 {
		return types.ErrorForExpression
	}

	if cLen == 2 {
		if tok := cd3[1].Tok; tok != "++" && tok != "--" {
			return types.ErrorForExpression
		}
	}

	// FIXME 暂未基于此类表达式做通用处理
	if cLen >= 3 {
		if tok := cd3[1].Tok; tok != "+=" && tok != "-=" {
			return types.ErrorForExpression
		}
	}

	cd3 = append(cd3, &global.Structure{Tok: ";", Lit: ";"})
	var result *global.Structure
	for {
		// n < y
		rv, err := r.parse(cd2, innerVar)
		if err != nil {
			return err
		}
		if !global.TransformAllToBool(rv) {
			break
		}

		if result, err = r.initExpr(forExpr.Code, innerVar, &parsing{isInLoop: true}); err != nil {
			return err
		}

		if _, err = r.initExpr(cd3, innerVar, &parsing{isInLoop: true}); err != nil {
			return err
		}

		// 发现 continue 则跳出当前循环
		if result != nil && result.Tok == "continue" {
			continue
		}

		// 发现 break 则跳出当前循环
		if result != nil && result.Tok == "break" {
			break
		}
	}
	return nil
}
