package lit

import (
	"strconv"

	"github.com/pywee/lit/global"
	"github.com/pywee/lit/types"
)

// parseIdentedVAR 解析变量声明
func parseIdentedVAR(r *expression, blocks []*global.Block, expr []*global.Structure, innerVal global.InnerVar, tok string, rlen, i int) ([]*global.Block, int) {
	var (
		thisLit = expr[i].Lit
		thisTok = expr[i].Tok
		code    = make([]*global.Structure, 0, 4)
	)

	for j := i; j < rlen; j++ {
		exprJ := expr[j]
		if exprJ.Tok == ";" {
			i = j
			break
		}
		code = append(code, exprJ)
	}

	if len(code) < 3 {
		return nil, -1
	}

	if tok == "=" {
		blocks = append(blocks, &global.Block{Name: thisLit, Type: types.CodeTypeIdentVAR, Code: code[2:]})
		return blocks, i
	}

	varValue := code[2:]
	if tok == "+=" {
		varValue = append(varValue, &global.Structure{Tok: ")"})
		nexpr := append([]*global.Structure{{Tok: thisTok, Lit: thisLit}, {Tok: "+"}, {Tok: "("}}, varValue...)
		blocks = append(blocks, &global.Block{Name: thisLit, Type: types.CodeTypeIdentVAR, Code: nexpr})
		return blocks, i
	}

	if tok == "-=" {
		varValue = append(varValue, &global.Structure{Tok: ")"})
		nexpr := append([]*global.Structure{{Tok: thisTok, Lit: thisLit}, {Tok: "-"}, {Tok: "("}}, varValue...)
		blocks = append(blocks, &global.Block{Name: thisLit, Type: types.CodeTypeIdentVAR, Code: nexpr})
		return blocks, i
	}

	if tok == "*=" {
		varValue = append(varValue, &global.Structure{Tok: ")"})
		nexpr := append([]*global.Structure{{Tok: thisTok, Lit: thisLit}, {Tok: "*"}, {Tok: "("}}, varValue...)
		blocks = append(blocks, &global.Block{Name: thisLit, Type: types.CodeTypeIdentVAR, Code: nexpr})
		return blocks, i
	}

	if tok == "/=" {
		varValue = append(varValue, &global.Structure{Tok: ")"})
		nexpr := append([]*global.Structure{{Tok: thisTok, Lit: thisLit}, {Tok: "/"}, {Tok: "("}}, varValue...)
		blocks = append(blocks, &global.Block{Name: thisLit, Type: types.CodeTypeIdentVAR, Code: nexpr})
		return blocks, i
	}

	if tok == "%=" {
		varValue = append(varValue, &global.Structure{Tok: ")"})
		nexpr := append([]*global.Structure{{Tok: thisTok, Lit: thisLit}, {Tok: "%"}, {Tok: "("}}, varValue...)
		blocks = append(blocks, &global.Block{Name: thisLit, Type: types.CodeTypeIdentVAR, Code: nexpr})
		return blocks, i
	}

	if tok == "&=" {
		varValue = append(varValue, &global.Structure{Tok: ")"})
		nexpr := append([]*global.Structure{{Tok: thisTok, Lit: thisLit}, {Tok: "&"}, {Tok: "("}}, varValue...)
		blocks = append(blocks, &global.Block{Name: thisLit, Type: types.CodeTypeIdentVAR, Code: nexpr})
		return blocks, i
	}

	if tok == "|=" {
		varValue = append(varValue, &global.Structure{Tok: ")"})
		nexpr := append([]*global.Structure{{Tok: thisTok, Lit: thisLit}, {Tok: "|"}, {Tok: "("}}, varValue...)
		blocks = append(blocks, &global.Block{Name: thisLit, Type: types.CodeTypeIdentVAR, Code: nexpr})
		return blocks, i
	}

	if tok == "^=" {
		varValue = append(varValue, &global.Structure{Tok: ")"})
		nexpr := append([]*global.Structure{{Tok: thisTok, Lit: thisLit}, {Tok: "^"}, {Tok: "("}}, varValue...)
		blocks = append(blocks, &global.Block{Name: thisLit, Type: types.CodeTypeIdentVAR, Code: nexpr})
		return blocks, i
	}

	// FIXME 当前不确定此处返回是否存在副作用
	// 逻辑上而言不会走到这里
	return nil, -1
}

// parseIdentedArrayVAR 解析数组赋值
// 当出现错误时返回变量 i 的值为 -1
func parseIdentedArrayVAR(r *expression, blocks []*global.Block, expr []*global.Structure, innerVal global.InnerVar, tokIdx, rlen, i int) ([]*global.Block, int) {
	var (
		brkCount  int
		arrayName = expr[i].Lit
		brkExpr   = make([]*global.Structure, 0, 1)
		idxExprs  = make([][]*global.Structure, 0, 2)
	)

	for j := i + 1; j < rlen; j++ {
		exprJ := expr[j]
		// global.Output(exprJ)
		if exprJ.Tok == ";" {
			i = j
			break
		}

		brkExpr = append(brkExpr, exprJ)
		if exprJ.Tok == "[" {
			brkCount++
			// 异常处理 避免出现非法的赋值方式
			// 如 a[[1]] = 1
			if j > 0 && brkCount > 1 && expr[j-1].Tok != "IDENT" {
				return nil, -1
			}
		} else if exprJ.Tok == "]" {
			brkCount--
			if brkCount == 0 {
				brkLen := len(brkExpr)
				if brkLen < 3 {
					return nil, -1
				}
				idxExprs = append(idxExprs, brkExpr[1:brkLen-1])
				brkExpr = nil
			}
		}
	}

	blocks = append(blocks, &global.Block{
		Name:     arrayName,
		Type:     types.CodeTypeIdentArrayVAR,
		Code:     expr[tokIdx+1 : i],
		ArrayIdx: idxExprs,
	})
	return blocks, i
}

// parseIdentedVarPLUS 解析变量自增
func parseIdentedVarPLUS(blocks []*global.Block, expr []*global.Structure, i int, rlen int) ([]*global.Block, int) {
	vplus := make([]*global.Structure, 0, 3)
	for j := i; j < rlen; j++ {
		exprJ := expr[j]
		if exprJ.Tok == ";" {
			blocks = append(blocks, &global.Block{
				Name: expr[i].Lit,
				Type: types.CodeTypeVariablePlus,
				Code: vplus,
			})
			i = j
			break
		}
		vplus = append(vplus, exprJ)
	}
	return blocks, i
}

// parseIdentedVarREDUCE 解析变量自减
func parseIdentedVarREDUCE(blocks []*global.Block, expr []*global.Structure, i int, rlen int) ([]*global.Block, int) {
	vreduce := make([]*global.Structure, 0, 3)
	for j := i; j < rlen; j++ {
		exprJ := expr[j]
		if exprJ.Tok == ";" {
			blocks = append(blocks, &global.Block{
				Name: expr[i].Lit,
				Type: types.CodeTypeVariableReduce,
				Code: vreduce,
			})
			i = j
			break
		}
		vreduce = append(vreduce, exprJ)
	}
	return blocks, i
}

// execVarPlusReduce 执行变量递增、递减操作
func execVarPlusReduce(block *global.Block, innerVar map[string]*global.Structure, plus bool) (*global.Structure, error) {
	blockName := block.Name
	v1, ok := innerVar[blockName]
	if !ok {
		return nil, types.ErrorNotFoundVariable
	}

	vTok := v1.Tok
	if vTok != "INT" && vTok != "FLOAT" && vTok != "STRING" {
		return nil, types.ErrorHandleUnsupported
	}

	var rv *global.Structure
	if vTok == "INT" {
		retParsed, err := strconv.ParseInt(v1.Lit, 10, 64)
		if err != nil {
			return nil, err
		}
		if plus {
			retParsed++
		} else {
			retParsed--
		}
		rv = &global.Structure{Tok: "INT", Lit: strconv.FormatInt(retParsed, 10)}
	} else if vTok == "FLOAT" {
		retParsed, err := strconv.ParseFloat(v1.Lit, 64)
		if err != nil {
			return nil, err
		}
		if plus {
			retParsed++
		} else {
			retParsed--
		}
		// rv = &global.Structure{Tok: "FLOAT", Lit: strconv.FormatFloat(retParsed, 'e', -1, 64)}
		rv = &global.Structure{Tok: "FLOAT", Lit: strconv.FormatFloat(retParsed, 'f', -1, 64)}
	} else if vTok == "STRING" {
		if ok, _ := global.IsInt(v1.Lit); ok {
			retParsed, err := strconv.ParseInt(v1.Lit, 10, 64)
			if err != nil {
				return nil, err
			}
			if plus {
				retParsed++
			} else {
				retParsed--
			}
			rv = &global.Structure{Tok: "INT", Lit: strconv.FormatInt(retParsed, 10)}
		} else if ok, _ := global.IsFloat(v1.Lit); ok {
			retParsed, err := strconv.ParseFloat(v1.Lit, 64)
			if err != nil {
				return nil, err
			}
			if plus {
				retParsed++
			} else {
				retParsed--
			}
			rv = &global.Structure{Tok: "FLOAT", Lit: strconv.FormatFloat(retParsed, 'f', -1, 64)}
		}
	}
	if rv != nil {
		innerVar[blockName] = rv
	}
	return rv, nil
}
