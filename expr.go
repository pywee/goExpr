package lit

import (
	"go/scanner"
	"go/token"
	"strconv"
	"strings"

	fn "github.com/pywee/lit/function"
	"github.com/pywee/lit/global"
	"github.com/pywee/lit/types"
)

// 2022.11.13
// 重写核心代码 加快解析速度
// 完成 for 循环第一种循环形式
// 修复一些小 bug
// 加入新字段标记自定义函数是否有返回值

// cfn 代码体内的自定义函数
var cfn *fn.CustomFunctions

type expression struct {
	funcBlocks     []*fn.FunctionInfo
	codeBlocks     []*global.Block
	publicVariable map[string]*global.Structure
}

type exprResult struct {
	Type  string
	Tok   string
	Value interface{}
}

func NewExpr(src []byte) (*expression, error) {
	var (
		err    error
		s      scanner.Scanner
		fset   = token.NewFileSet()
		expr   = make([]*global.Structure, 0, 100)
		result = new(expression)
	)

	file := fset.AddFile("", fset.Base(), len(src))
	s.Init(file, src, nil, scanner.ScanComments)

	negative := ""
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		stok := tok.String()
		if stok == "COMMENT" {
			continue
		}

		// 负数处理
		if stok == "-" {
			if eLen := len(expr); eLen > 0 && expr[eLen-1].Lit == "" && expr[eLen-1].Tok != ")" {
				negative = "-"
				continue
			}
		}

		if negative == "-" {
			negative = ""
			lit = "-" + lit
		}

		posString := fset.Position(pos).String()
		if sLit := strings.ToLower(lit); sLit == "false" || sLit == "true" {
			stok = "BOOL"
		}
		if stok == "STRING" || stok == "CHAR" {
			lit, err = global.FormatString(lit)
			if err != nil {
				return nil, err
			}
		}
		expr = append(expr, &global.Structure{
			Tok:      stok,
			Lit:      lit,
			Position: posString,
		})
	}

	innerVar := make(map[string]*global.Structure, 5)
	_, err = result.initExpr(expr, innerVar, nil)
	// global.Output(innerVar["a"])
	return nil, err
}

type parsing struct {
	isInLoop bool
	isInFunc bool
}

// parseExprs 解析代码块
func (r *expression) parseExprs(expr []*global.Structure, innerVar global.InnerVar, runtime *parsing) (*expression, error) {
	var (
		err        error
		foundElse  bool
		rlen       = len(expr)
		blocks     = make([]*global.Block, 0, 20)
		funcBlocks = make([]*fn.FunctionInfo, 0, 20)
	)
	for i := 0; i < rlen; i++ {
		thisExpr := expr[i]
		if thisExpr.Tok == ";" && thisExpr.Lit == "\n" {
			continue
		}

		// FIXME.
		// 负数处理 bug

		// TODO
		// For
		// 数组
		// 对象
		// 内置函数完善
		// 抛出错误时 明确返回出现错误的行数

		// for 流程控制语句
		// FIXME 未针对for语句的合法性做充分检查
		if thisExpr.Tok == "for" {
			blocks, i, err = r.parseIdentedFOR(expr, blocks, innerVar, i)
			if err != nil {
				return nil, err
			}
			continue
		}

		// continue 语句
		if thisExpr.Tok == "continue" {
			if runtime == nil || !runtime.isInLoop {
				return nil, types.ErrorForContinue
			}
			blocks = append(blocks, &global.Block{Type: types.CodeTypeContinue})
			return &expression{codeBlocks: blocks, funcBlocks: funcBlocks}, nil
		}

		// break 语句
		if thisExpr.Tok == "break" {
			if runtime == nil || !runtime.isInLoop {
				return nil, types.ErrorForBreak
			}
			blocks = append(blocks, &global.Block{Type: types.CodeTypeBreak})
			return &expression{codeBlocks: blocks, funcBlocks: funcBlocks}, nil
		}

		// 变量赋值
		// 数组赋值
		if thisExpr.Tok == "IDENT" && i+1 < rlen {
			tok := expr[i+1].Tok
			if global.InArrayString(tok, global.MathSym) {
				if blocks, i = parseIdentedVAR(r, blocks, expr, innerVar, tok, rlen, i); i == -1 {
					return nil, types.ErrorWrongVarOperation
				}
				continue
			}
			if tokIdx := global.IsTokInArray(expr[i:], "="); tokIdx != -1 && tok == "[" {
				if blocks, i = parseIdentedArrayVAR(r, blocks, expr, innerVar, i+tokIdx, rlen, i); i == -1 {
					return nil, types.ErrorIlligleVisitedOfArray
				}
				continue
			}
		}

		// return 语句
		// FIXME 函数外执行时未做判断 此时不应该通过
		if thisExpr.Tok == "return" {
			var returnExpr = make([]*global.Structure, 0, 5)
			for j := i + 1; j < rlen; j++ {
				exprJ := expr[j]
				if exprJ.Tok == ";" {
					blocks = append(blocks, &global.Block{Type: types.CodeTypeIdentRETURN, Code: returnExpr})
					return &expression{funcBlocks: funcBlocks, codeBlocks: blocks}, nil
				}
				returnExpr = append(returnExpr, exprJ)
			}
			continue
		}

		// 函数定义
		if expr[i].Tok == "func" {
			if !global.IsVariableOrFunction(expr[i+1]) {
				return nil, types.ErrorFunctionNameIrregular
			}
			if funcBlocks, i, err = parseIdentFUNC(funcBlocks, expr, i, rlen); err != nil {
				return nil, err
			}
			continue
		}

		// 函数调用
		if expr[i].Tok == "IDENT" && i+1 < rlen && expr[i+1].Tok == "(" {
			blocks, i = parseExecFUNC(blocks, expr, i, rlen)
			continue
		}

		// if 流程控制语句
		if expr[i].Tok == "if" && !foundElse {
			parsed, err := parseIdentedIF(blocks, expr, i, rlen)
			if err != nil {
				return nil, err
			}
			i = parsed.i
			blocks = parsed.blocks
			continue
		}

		// else 处理
		if expr[i].Tok == "else" {
			parsed, err := parseIdentELSE(blocks, expr, i, rlen)
			if err != nil {
				return nil, err
			}
			i = parsed.i
			blocks = parsed.blocks
			foundElse = parsed.foundElse

			// FIXME 不确定此处 continue 是否存在副作用
			continue
		}

		// 变量自增操作
		if thisExpr.Tok == "IDENT" && i+1 < rlen && expr[i+1].Tok == "++" {
			blocks, i = parseIdentedVarPLUS(blocks, expr, i, rlen)
			continue
		}

		// 变量自减操作
		if thisExpr.Tok == "IDENT" && i+1 < rlen && expr[i+1].Tok == "--" {
			blocks, i = parseIdentedVarREDUCE(blocks, expr, i, rlen)
			continue
		}

		// 此处如果返回 则 if 语句中针对 else 的解析会出问题
		return nil, types.ErrorWrongSentence
	}

	return &expression{codeBlocks: blocks, funcBlocks: funcBlocks}, nil
}

// initExpr 全局表达式入口
// 代码块中如果带有 if 等复杂语句 则需要从这里进入递归
// 调用此函数时 expr 结尾处必须带入分号 ";"
func (r *expression) initExpr(expr []*global.Structure, innerVar global.InnerVar, runtime *parsing) (*global.Structure, error) {
	bs, err := r.parseExprs(expr, innerVar, runtime)
	if err != nil {
		// global.Output(expr)
		return nil, err
	}

	// for _, v := range bs.codeBlocks {
	// 	global.Output(v.ArrayIdx)
	// }

	if len(r.funcBlocks) == 0 {
		r.funcBlocks = bs.funcBlocks
	}

	for _, block := range bs.codeBlocks {
		if block.Type == types.CodeTypeContinue {
			return &global.Structure{Tok: "continue", Lit: "continue"}, nil
		}

		if block.Type == types.CodeTypeBreak {
			return &global.Structure{Tok: "break", Lit: "break"}, nil
		}

		if block.Type == types.CodeTypeIdentRETURN {
			return r.parse(block.Code, innerVar)
		}

		if block.Type == types.CodeTypeIdentVAR {
			if block.Name == "" {
				return nil, types.ErrorWrongSentence
			}
			// global.Output(block.Code)
			rv, err := r.parse(block.Code, innerVar)
			if err != nil {
				return nil, err
			}
			if rv == nil {
				rv = &global.Structure{Tok: "NULL", Lit: "NULL"}
			}
			innerVar[block.Name] = rv
			// return &global.Structure{Tok: "BOOL", Lit: "true"}, nil
		} else if block.Type == types.CodeTypeIdentArrayVAR {
			v, ok := innerVar[block.Name]
			if !ok || v.Tok != "ARRAY" {
				return nil, types.ErrorNotFoundIdentedArray
			}

			// arrList 当前维度数组数据
			arrList := v.Arr.List
			// idxs 所有访问下标
			idxs := block.ArrayIdx
			// arrLen 当前维度下的数组长度
			arrLen := len(arrList)
			for _, idx := range idxs {
				parsedIdx, err := r.parse(idx, innerVar)
				if err != nil {
					return nil, err
				}
				idxINT, err := checkArrayIdx(parsedIdx)
				if err != nil {
					return nil, err
				}
				// global.Output(innerVar["b"].Arr.List[1].Values)
				if idxINT >= arrLen {
					return nil, types.ErrorOutOfArrayRange
				}

				// 定义数组并赋值
				// 修改指定下标值
				// a = [123]
				// a[0] = "hello"
				// println(a[0]) 得到 hello

				thisArr := arrList[idxINT]
				if thisArrValuesLen := len(thisArr.Values); thisArrValuesLen > 0 {
					arrList[idxINT].Values = block.Code
					// v.Arr.List[idxINT].Values = []*global.Structure{{Tok: "INT", Lit: "1"}}
				} else if thisArrValuesLen == 0 && thisArr.Child != nil {
					arrList = thisArr.Child.List
				}
			}
		} else if block.Type == types.CodeTypeFunctionExec {
			rv, err := r.parse(block.Code, innerVar)
			if err != nil {
				return nil, err
			}
			if rv != nil {
				return rv, nil
			}
			// return &global.Structure{Tok: "NULL", Lit: "NULL"}, nil
		} else if block.Type == types.CodeTypeIdentIF {
			// 检查if语句合法性
			if err = checkLegitimateIF(block.IfExt); err != nil {
				return nil, err
			}

			var conditionResult bool
			for _, v := range block.IfExt {
				if v.ConditionLen == 0 {
					conditionResult = true
				} else if v.ConditionLen > 0 {
					rv, err := r.parse(v.Condition, innerVar)
					if err != nil {
						return nil, err
					}
					conditionResult = global.TransformAllToBool(rv)
				}
				if !conditionResult {
					continue
				}

				blen := len(v.Body)
				if blen < 2 {
					return nil, types.ErrorIfExpression
				}
				ret, err := r.initExpr(v.Body[1:blen-1], innerVar, runtime)
				if err != nil {
					return nil, err
				}
				if ret != nil {
					return ret, nil
				}
				break
			}
		} else if block.Type == types.CodeTypeVariablePlus {
			if len(block.Code) != 2 {
				return nil, types.ErrorWrongSentence
			}
			if _, err = execVarPlusReduce(block, innerVar, true); err != nil {
				return nil, err
			}
		} else if block.Type == types.CodeTypeVariableReduce {
			if len(block.Code) != 2 {
				return nil, types.ErrorWrongSentence
			}
			if _, err = execVarPlusReduce(block, innerVar, false); err != nil {
				return nil, err
			}
		} else if block.Type == types.CodeTypeIdentFOR {
			forExpr := block.ForExt
			if fType := forExpr.Type; fType == types.TypeForExpressionIteration {
				if err = r.execFORType1(forExpr, innerVar); err != nil {
					return nil, err
				}
			} else if fType == types.TypeForExpressionRange {
				if err = r.execForTypeRange(forExpr, innerVar); err != nil {
					return nil, err
				}
			}
		}
	}
	return nil, nil
}

// parse 解析算术表达式、逻辑运算等
// 用于最小粒度处理
func (r *expression) parse(expr []*global.Structure, innerVar global.InnerVar) (*global.Structure, error) {
	exLen := len(expr)
	if exLen == 1 {
		// 变量解析
		rv := expr[0]
		if rv != nil && rv.Tok == "IDENT" {
			if innerRv, ok := innerVar[rv.Lit]; ok {
				return innerRv, nil
			}
			return nil, types.ErrorNotFoundVariable
		}
		return rv, nil
	}

	// 解析数组定义 将元素放入树结构
	if exLen > 1 && expr[0].Tok == "[" && expr[exLen-1].Tok == "]" {
		return &global.Structure{Tok: "ARRAY", Lit: "Array", Arr: parseIdentARRAY(expr)}, nil
	}

	var innerExpr = make([]*global.Structure, 0, 10)
	for i := 0; i < exLen; i++ {
		if expr[i].Tok == "IDENT" && i+1 < exLen && expr[i+1].Tok == "(" {
			var (
				brCount  uint8
				funcExec = make([]*global.Structure, 0, 3)
			)
			for j := i; j < exLen; j++ {
				funcExec = append(funcExec, expr[j])
				if expr[j].Tok == "(" {
					brCount++
				}
				if expr[j].Tok == ")" {
					brCount--
					if brCount == 0 {
						// global.Output(innerVar["a"])
						// global.Output(expr[i+2 : j])
						rv, err := r.execFUNC(funcExec, expr[i+2:j], innerVar)
						if err != nil {
							return nil, err
						}
						if rv != nil {
							innerExpr = append(innerExpr, rv)
						}
						i = j
						break
					}
				}
			}
			continue
		}

		if expr[i].Tok == "(" {
			var (
				bracketCount uint8
				bracketExprs = make([]*global.Structure, 0, 5)
			)
			for j := i; j < exLen; j++ {
				exprJ := expr[j]
				bracketExprs = append(bracketExprs, exprJ)
				if exprJ.Tok == "(" {
					bracketCount++
				} else if exprJ.Tok == ")" {
					bracketCount--
					if bracketCount == 0 {
						// global.Output(bracketExprs[1 : len(bracketExprs)-1])
						rv, err := r.parse(bracketExprs[1:len(bracketExprs)-1], innerVar)
						if err != nil {
							return nil, err
						}
						i = j
						innerExpr = append(innerExpr, rv)
						break
					}
				}
			}
			continue
		}

		var (
			// arrayType 判断当前是否为访问数组下标
			// 如果当前是外部变量 则标识为1 如访问 a[0] 此时 a is array
			// 如果当前是临时变量 则标识为2 如访问 a[0][1] 的时候，a[0]返回的是一个临时变量 重新进行递归
			// 此则 a[0]的结果就是 arrayType=2
			arrayType uint8
			// thisArrayVar 当前访问到下标的数组结果
			thisArrayVar *global.Structure
		)

		if exLen > 3 {
			if i+1 < exLen && expr[i].Tok == "IDENT" && expr[i+1].Tok == "[" {
				var ok bool
				arrayType = 1
				thisArrayVar, ok = innerVar[expr[i].Lit]
				if !ok {
					return nil, types.ErrorNotFoundVariable
				}
				if thisArrayVar.Tok != "ARRAY" {
					return nil, types.ErrorVariableIsNotAndArray
				}
			} else if expr[i].Tok == "ARRAY" && expr[i+1].Tok == "[" {
				arrayType = 2
				thisArrayVar = expr[i]
			}
		}

		// 访问数组下标
		if arrayType > 0 {
			var (
				mBracket  int8
				indexExpr = make([]*global.Structure, 0, 5)
			)

			for j := i + 1; j < exLen; j++ {
				exprJ := expr[j]
				if exprJ.Tok == "[" {
					mBracket++
				} else if exprJ.Tok == "]" {
					mBracket--
					if len(indexExpr) <= 1 {
						return nil, types.ErrorArrayIndexVisitingIlligle
					}

					// 解析下标表达式
					idxResult, err := r.parse(indexExpr[1:], innerVar)
					if err != nil {
						return nil, err
					}

					// 针对整型类字符串下标访问的处理
					if idxResult.Tok == "STRING" {
						var ok bool
						if ok, err = global.IsInt(idxResult.Lit); err != nil {
							return nil, err
						}
						if !ok {
							return nil, types.ErrorInvalidArrayIndexType
						}
						if idxResult, err = global.TransformAllToInt(idxResult); err != nil {
							return nil, err
						}
					}

					if idxResult.Tok != "INT" {
						return nil, types.ErrorInvalidArrayIndexType
					}
					intIndex, err := strconv.ParseInt(idxResult.Lit, 10, 64)
					if err != nil {
						return nil, err
					}

					thisArrayVarArrList := thisArrayVar.Arr.List
					if int(intIndex) >= len(thisArrayVarArrList) {
						return nil, types.ErrorOutOfArrayRange
					}
					thisIndex := thisArrayVarArrList[intIndex]
					if thisIndex == nil {
						return nil, types.ErrorArrayIndexNotExists
					}

					// 获得 array 值
					listLen := len(thisIndex.Values)
					if listLen == 0 && thisIndex.Child == nil {
						return &global.Structure{Tok: "NULL", Lit: "NULL"}, nil
					}

					if listLen > 0 {
						arrValue, err := r.parse(thisIndex.Values, innerVar)
						if err != nil {
							return nil, err
						}
						innerExpr = append(innerExpr, arrValue)
					} else if thisIndex.Child != nil {
						innerExpr = append(innerExpr, &global.Structure{Tok: "ARRAY", Lit: "Array", Arr: thisIndex.Child})
					}
					i = j
					break
				}
				indexExpr = append(indexExpr, exprJ)
			}
			// return nil, types.ErrorArrayIndexNotExists
			continue
		}

		// 获取变量的值
		if expr[i].Tok == "IDENT" {
			if value, exists := innerVar[expr[i].Lit]; exists {
				innerExpr = append(innerExpr, value)
				continue
			}
			return nil, types.ErrorNotFoundVariable
		}

		// 逻辑运算 ||
		if expr[i].Tok == "||" {
			return r.parseOr(expr, innerExpr, innerVar, i)
		}

		// 逻辑运算 &&
		if expr[i].Tok == "&&" {
			return r.parseAnd(expr, innerExpr, innerVar, i)
		}
		innerExpr = append(innerExpr, expr[i])
	}

	if iLen := len(innerExpr); iLen >= 3 {
		for n := 0; n < iLen; n++ {
			// 比较运算
			// 考虑后期再支持 ===
			if tok := inArray(innerExpr[n].Tok, []string{"==", "!=", ">", "<", ">=", "<="}); tok != "" {
				rv, err := r.parseComparison(innerExpr[:n], innerExpr[n+1:], tok, innerVar)
				if err != nil {
					return nil, err
				}
				return rv, nil
			}
		}
	}

	innerLen := len(innerExpr)
	if innerLen == 1 {
		return innerExpr[0], nil
	}

	// 访问多维数组操作
	// a[0][0][0]
	if innerLen >= 4 && innerExpr[0].Tok == "ARRAY" {
		// FIXME 检测合法性
		if innerExpr[1].Tok != "[" {
			return nil, types.ErrorWrongSentence
		}
		return r.parse(innerExpr, innerVar)
	}

	// 数学计算
	innerExpr, err := r.parseExpr(innerExpr, "")
	if err != nil {
		return nil, err
	}
	if len(innerExpr) == 1 {
		return innerExpr[0], nil
	}
	return nil, nil
}

// getIdentedCustomFunc 获取定义好的自定义函数代码块
func (r *expression) getIdentedCustomFunc(fname string) *fn.FunctionInfo {
	for _, v := range r.funcBlocks {
		if v.FunctionName == fname {
			return v
		}
	}
	return nil
}

func inArray(sep string, arr []string) string {
	for _, v := range arr {
		if sep == v {
			return v
		}
	}
	return ""
}

func (r *expression) Get(vName string) (*global.Structure, error) {
	ret, ok := r.publicVariable[vName]
	if !ok {
		return nil, types.ErrorNotFoundVariable
	}
	return ret, nil
}

func (r *expression) GetVal(vName string) interface{} {
	ret, ok := r.publicVariable[vName]
	if !ok {
		return types.ErrorNotFoundVariable
	}
	if len(ret.Lit) > 1 && (ret.Tok == "STRING" || ret.Tok == "CHAR") {
		lit, _ := global.FormatString(ret.Lit)
		return lit
	}
	return ret.Lit
}
