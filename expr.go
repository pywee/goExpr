package goExpr

import (
	"errors"
	"fmt"
	"go/scanner"
	"go/token"
	"log"
	"strconv"
	"strings"

	fn "github.com/pywee/goExpr/function"
	"github.com/pywee/goExpr/global"
	"github.com/pywee/goExpr/types"
)

type Expression struct {
	publicVariable map[string]*global.Structure
}

func NewExpr(src []byte) (*Expression, error) {
	var (
		s      scanner.Scanner
		fset   = token.NewFileSet()
		result = &Expression{publicVariable: make(map[string]*global.Structure, 10)}
		list   = make([]*global.Structure, 0, 100)
	)

	file := fset.AddFile("", fset.Base(), len(src))
	s.Init(file, src, nil, scanner.ScanComments)
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}

		if tok.String() == ";" {
			var vName string
			if vleft, vLeftListEndIdx := findStrInfrontSymbool("=", list); vLeftListEndIdx != -1 {
				vName = vleft[0].Lit
				list = list[vLeftListEndIdx+1:]
			}

			// 递归解析表达式
			posLine := strings.Split(fset.Position(pos).String(), ":")
			for _, v := range list {
				if sLit := strings.ToLower(v.Lit); sLit == "false" || sLit == "true" {
					v.Tok = "BOOL"
				}
			}

			rv, err := result.parse(list, "第"+posLine[0]+"行, ")
			if err != nil {
				return nil, errors.New("第" + posLine[0] + "行, " + err.Error())
			}
			global.Output(rv)

			// 变量赋值
			if vName != "" {
				// fmt.Printf("set %s to %v\n", vName, rv)
				result.publicVariable[vName] = rv
			}
			list = nil
			continue
		}

		tokString := tok.String()
		if tok.String() == "CHAR" {
			tokString = "STRING"
		}
		if tokString == "STRING" {
			lit = formatString(lit)
		}

		list = append(list, &global.Structure{
			Position: fset.Position(pos).String(),
			Tok:      tokString,
			Lit:      lit,
		})

		// fmt.Printf("[ %s ]\t[ %s ]\t [ %s ] \n", fset.Position(pos).String(), tok, lit)
	}

	return result, nil
}

func (r *Expression) parse(expr []*global.Structure, pos string) (*global.Structure, error) {
	rLen := len(expr)
	if rLen == 0 {
		return nil, nil
	}
	if rLen == 1 {
		return expr[0], nil
	}

	var (
		foundKuo   bool
		count      int
		firstKey   int = -1
		firstIdent *global.Structure
	)
	for k, v := range expr {
		if v.Tok == "&&" && firstKey == -1 {
			return r.parseAnd(expr, k, pos)
		}
		if v.Tok == "||" && firstKey == -1 {
			return r.parseOr(expr, k, pos)
		}

		if v.Tok == "(" {
			count++
			if firstKey == -1 {
				firstKey = k
			}
			if firstIdent == nil && k > 0 && !foundKuo {
				firstIdent = expr[k-1]
			}
			foundKuo = true
		} else if v.Tok == ")" {
			count--
		}

		// a = (("wwww") + 222))
		// a = IsInt(("wwww")+222);
		// a = "你"+Replace("你好", "2", "3", 4)+"xxx";
		// FIXME a = (1+IsInt(1+(3))+(11+2)); fixed
		if count == 0 && foundKuo && firstKey != -1 {
			first := []*global.Structure{}
			if firstKey > 0 {
				// 发现函数
				// 执行 IsInt(1) 和 1+Isint(1) 时
				// 得到的 first 不一样
				first = expr[:firstKey]
				if firstIdent != nil && firstIdent.Tok == "IDENT" && expr[firstKey-1] == firstIdent {
					first = expr[:firstKey-1]
				}
			}

			// end 必须放在 middle 前面
			// 避免变量 expr 被修改了长度
			end := []*global.Structure{}
			if k < len(expr) && len(expr[k+1:]) > 0 {
				end = expr[k+1:]
			}

			// 发现中间表达式为函数执行调用
			var (
				err    error
				middle *global.Structure
			)

			if global.IsVariableOrFunction(firstIdent) {
				if middle, err = r.execFunc(firstIdent, expr[firstKey+1:k], pos); err != nil {
					return nil, err
				}
			} else if middle, err = r.parse(expr[firstKey+1:k], pos); err != nil {
				return nil, err
			}

			// 左中右结合再次递归
			expr = append(first, middle)
			expr = append(expr, end...)

			return r.parse(expr, pos)
		}
	}

	// 最小粒度进入到算术表达式中计算
	rv, err := r.parseExpr(expr, pos)
	if err != nil {
		return nil, err
	}
	return rv[0], nil
}

// parse 解析器主入口
func (r *Expression) parseB(expr []*global.Structure, pos string, foundAndOr bool) (*global.Structure, error) {
	var err error
	rLen := len(expr)
	if rLen == 0 {
		return nil, nil
	}

	e0 := expr[0]
	if rLen == 1 && e0.Tok != "IDENT" {
		return e0, nil
	}

	// 只找括号不换函数
	i := 0
	foundK := -1
	kList := make([]*global.Structure, 0, 10)

	for k, v := range expr {
		if v.Tok == "(" {
			if foundK == -1 {
				foundK = k
			}
			i++
		}
		if v.Tok == ")" {
			i--
		}
		if foundK >= 0 {
			kList = append(kList, v)
		}

		// 括号 非函数
		if foundK >= 0 && i == 0 {
			if foundK == 0 || expr[foundK-1].Tok != "IDENT" {
				// Fixed
				// VarDump((1)+(2)) 此时会进入死循环 因为每次都取到 (1）
				var rv *global.Structure
				if kLen := len(kList); kLen > 1 && kList[0].Tok == "(" && kList[kLen-1].Tok == ")" {
					rv, err = r.parseB(kList[1:kLen-1], pos, foundAndOr)
					if err != nil {
						return nil, err
					}
				} else if rv, err = r.parseB(kList, pos, foundAndOr); err != nil {
					return nil, err
				}

				k1 := expr[k+1:]
				expr = append(expr[:foundK], rv)
				expr = append(expr, k1...)
				foundK = -1
				return r.parseB(expr, pos, foundAndOr)
			}

			// 在括号的结尾发现函数
			if kLen := len(kList); kLen > 1 && kList[0].Tok == "(" && kList[kLen-1].Tok == ")" {
				if rv, _ := r.doFunc(expr, pos, foundAndOr); rv != nil {
					return rv, nil
				}
			}
		}
	}

	// FIXME
	// 临时有符号的整型和浮点型的处理逻辑
	if rLen == 2 && (e0.Tok == "-" || e0.Tok == "+") {
		// TODO
		// 考虑弱类型支持 如 "-100" 和 -100 是否等义
		if e1 := expr[1]; e1.Tok == "INT" || e1.Tok == "FLOAT" {
			if e0.Tok == "+" {
				return &global.Structure{Tok: e1.Tok, Lit: e1.Lit}, nil
			}
			return &global.Structure{Tok: e1.Tok, Lit: e0.Tok + e1.Lit}, nil
		}
	}

	// FIXME
	// 针对有 && 和 || 符号的表达式才进入这个逻辑
	// 后期可以进一步优化
	if foundAndOr {
		rvAfter, err := r.parseAndOr(expr, pos, foundAndOr)
		if err != nil {
			return nil, err
		}
		if rvAfter != nil {
			return rvAfter, nil
		}
	}

	// 函数判断
	if rLen >= 3 {
		rv, err := r.doFunc(expr, pos, foundAndOr)
		if err != nil {
			return nil, err
		}
		if rv != nil {
			return rv, nil
		}
	}

	// 只找函数不找括号
	count := 0
	found := false
	foundFNKey := -1
	fns := []*global.Structure{}
	for k, v := range expr {
		if v.Tok == "(" {
			if foundFNKey == -1 && k > 0 && expr[k-1].Tok == "IDENT" {
				foundFNKey = k
				fns = append(fns, expr[k-1])
			}
			found = true
			count++
		} else if v.Tok == ")" {
			count--
		}
		if count > 0 {
			fns = append(fns, v)
		}
		if count == 0 && found {
			if foundFNKey != -1 {
				// 补全最后一个括号
				fns = append(fns, v)
				rv, err := r.parseB(fns, pos, foundAndOr)
				if err != nil {
					return nil, err
				}
				right := []*global.Structure{}
				if k+1 < len(expr) {
					right = expr[k+1:]
				}
				if foundFNKey > 1 {
					expr = append(expr[:foundFNKey-1], rv)
				} else {
					expr = []*global.Structure{rv}
				}
				expr = append(expr, right...)
				return r.parseB(expr, pos, foundAndOr)
			}
		}
	}

	// 找出剩余的未完成的函数
	// 如 true+isInt(1)+isFloat(1.1)
	// 如果不在此处进行检查 那么当前函数只会解析前面的 isInt
	// 而后面的 isFloat 会丢失
	// 这与 isInt(1)+isFloat(1.1)+true 的处理逻辑不一样
	// 所以这里还需要一次递归处理
	// foundLastFuncExpr := false
	for k, v := range expr {
		if v.Tok == "IDENT" && k+1 < len(expr) && expr[k+1].Tok == "(" {
			rv, err := r.parseB(expr, pos, foundAndOr)
			if err != nil {
				return nil, err
			}
			expr = append(expr[:k], rv)
		}
	}

	// FIXME
	if len(expr) == 0 {
		println("表达式可能有误！！！")
		return nil, nil
	}

	rv, err := r.parseExpr(expr, pos)
	if err != nil {
		return nil, err
	}
	return rv[0], nil
}

// parseAndOr
func (r *Expression) parseAndOr(expr []*global.Structure, pos string, foundAndOr bool) (*global.Structure, error) {
	// FIXME 针对 && 符号的解析
	// 优先处理括号
	// 1.针对已经声明的布尔值没有处理正确
	// example false && 12345;
	// 2.使用函数的时候 在带有 && 符号语句中没有解析出正确结果
	foundK := false
	for k, v := range expr {
		if v.Tok == "(" {
			foundK = true
		}
		if v.Tok == "&&" && len(expr) >= 3 && k > 0 {
			if foundK {
				return nil, nil
			}
			rvLeft, err := r.parseB(expr[:k], pos, foundAndOr)
			if err != nil {
				return nil, err
			}
			if fn.ChangeBool(rvLeft).IsBoolFalse() {
				return rvLeft, nil
			}
			rvRight, err := r.parseB(expr[k+1:], pos, foundAndOr)
			if err != nil {
				return nil, err
			}
			return fn.ChangeBool(rvRight), nil
		}

		if v.Tok == "||" && len(expr) >= 3 && k > 0 {
			rvLeft, err := r.parseB(expr[:k], pos, foundAndOr)
			if err != nil {
				return nil, err
			}
			if fn.ChangeBool(rvLeft).IsBoolTrue() {
				return rvLeft, nil
			}
			rvRight, err := r.parseB(expr[k+1:], pos, foundAndOr)
			if err != nil {
				return nil, err
			}
			return fn.ChangeBool(rvRight), nil
		}
	}
	return nil, nil
}

// parseExpr 解析算术表达式入口
// 1+11|23-12;
// 1+2+(3%10+(11*22+1.1-10)+41+(5+9)+10)+1+(2-(10*2%2));
// 2+1/(20-1*(3+10)-1)%2^1;
func (r *Expression) parseExpr(expr []*global.Structure, pos string) ([]*global.Structure, error) {
	v, err := r.findExprK(expr, pos)
	if err != nil {
		return nil, err
	}
	return parsePlusReduceMulDivB(v, pos)
}

func (r *Expression) findExprK(expr []*global.Structure, pos string) ([]*global.Structure, error) {
	exprLen := len(expr)

	// 如果返回的是最终值 则不再需要进一步解析了
	// 加快处理速度
	if exprLen == 1 {
		if e0 := expr[0]; global.InArrayString(e0.Tok, []string{"INT", "STRING", "FLOAT", "BOOL"}) {
			return expr, nil
		}
	}

	// 错误表达式处理
	if exprLen > 1 {
		if e0Tok := expr[0].Tok; (e0Tok == "FLOAT" || e0Tok == "INT") && expr[1].Tok == "IDENT" {
			return nil, types.ErrorWrongSentence
		}
	}

	for k, v := range expr {
		if global.IsVariableOrFunction(v) {
			ret, err := r.Get(v.Lit)
			if err != nil {
				return nil, err
			}
			expr[k] = ret
		}
	}

	var (
		startI   int
		endI     int
		startIdx = -1
		endIdx   = -1
	)
	for k, v := range expr {
		if v.Tok == "(" {
			startI++
			if startIdx == -1 {
				startIdx = k
			}
		}
		if v.Tok == ")" {
			endI++
			endIdx = k
		}
		if endI == startI && startIdx != -1 && endIdx != -1 {
			temp, err := r.findExprK(expr[startIdx+1:endIdx], pos)
			if err != nil {
				return nil, err
			}
			result := expr[:startIdx]
			result = append(result, temp...)
			return r.findExprK(append(result, expr[endIdx+1:]...), pos)
		}
	}

	return parsePlusReduceMulDivB(expr, pos)
}

// parsePlusReduceMulDivB 处理加减乘除
func parsePlusReduceMulDivB(arr []*global.Structure, pos string) ([]*global.Structure, error) {
	result, err := parsePlusReduceMulDiv([]string{"*", "/", "%"}, arr)
	if err != nil {
		return nil, err
	}
	if len(result) == 1 {
		return result, nil
	}

	if result, err = parsePlusReduceMulDiv([]string{"&"}, result); err != nil {
		return nil, err
	}
	if len(result) == 1 {
		return result, nil
	}

	if len(result) > 0 {
		if result, err = parsePlusReduceMulDiv([]string{"+", "-", "|", "^"}, result); err != nil {
			return nil, err
		}
	}

	if len(result) != 1 {
		return nil, types.ErrorWrongSentence
	}
	return result, nil
}

// parsePlusReduceMulDiv 递归解析表达式 最内层
func parsePlusReduceMulDiv(arr []string, expr []*global.Structure) ([]*global.Structure, error) {
	if len(expr) == 0 {
		return expr, nil
	}

	var result []*global.Structure
	for k, v := range expr {
		if inArray(v.Tok, arr) && k-1 > -1 && k+2 <= len(expr) {
			r, err := findExprBetweenSymbool(expr[k-1], expr[k], expr[k+1])
			if err != nil {
				return nil, err
			}
			if r.Type == "STRING" {
				result = append([]*global.Structure{{Tok: "STRING", Lit: r.Value.(string)}}, expr[k+2:]...)
				return parsePlusReduceMulDiv(arr, result)
			}

			middle := &global.Structure{}
			if r.Type == "FLOAT" {
				// FIXME
				// 待观察是否对其他表达式有影响
				// example:
				// a = 2
				// b = 3
				// c = a + 100/10*2.0
				// output FLOAT 22.
				// 在使用 strings.TrimRight(lit, ".") 之后
				// 输出 FLOAT(22)

				middle.Tok = "FLOAT"
				middle.Lit = strings.TrimRight(fmt.Sprintf("%f", r.Value), "0")
				middle.Lit = strings.TrimRight(middle.Lit, ".")
			} else if r.Type == "INT" {
				middle.Tok = "INT"
				middle.Lit = fmt.Sprintf("%d", r.Value)
			}

			result = append(expr[:k-1], middle)
			result = append(result, expr[k+2:]...)
			return parsePlusReduceMulDiv(arr, result)
		}
	}
	return expr, nil
}

// findExprBetweenSymbool 查找某运算符数据
// 如: 1+2-3 输出查找 + 号 则返回 1+2
func findExprBetweenSymbool(l, m, r *global.Structure) (*exprResult, error) {
	var (
		exprTyp string
		left    interface{}
		right   interface{}
		lTok    = l.Tok
		mTok    = m.Tok
		rTok    = r.Tok
	)

	// 弱类型处理
	// 布尔值转整型
	if l.Lit == "true" {
		lTok = "INT"
		l.Tok = "INT"
		l.Lit = "1"
	} else if l.Lit == "false" {
		lTok = "INT"
		l.Tok = "INT"
		l.Lit = "0"
	}
	if r.Lit == "true" {
		rTok = "INT"
		r.Tok = "INT"
		r.Lit = "1"
	} else if r.Lit == "false" {
		rTok = "INT"
		r.Tok = "INT"
		r.Lit = "0"
	}

	// 字符串拼接及弱类型处理的算术计算
	if lTok == "STRING" && rTok == "STRING" {
		// 弱类型处理 如果左右两边都是字符串数字则允许进行算术计算
		isLeftNumeric, err := global.IsNumber(l.Lit)
		if err != nil {
			log.Print(err)
		}
		isRightNumeric, err := global.IsNumber(r.Lit)
		if err != nil {
			log.Print(err)
		}

		// 不是完全由数字构成的字符串则直接拼接
		// 否则转换类型进行算术计算
		// example A
		// a = "1";
		// b = "2";
		// print(a+b) 输出 (int)3

		// example B
		// a = "1";
		// b = "2a";
		// print(a+b) 输出 (string)12a

		if !isLeftNumeric || !isRightNumeric {
			if mTok == "+" {
				return &exprResult{Type: "STRING", Value: l.Lit + r.Lit}, nil
			}
			return nil, types.ErrorNonNumberic
		}
	}

	// 弱类型处理
	// 数字字符串转为整型或浮点型
	if lTok == "STRING" || lTok == "CHAR" {
		lit := l.Lit
		isFloat, err := global.IsFloat(lit)
		if err != nil {
			return nil, types.ErrorWrongSentence
		}
		if isFloat {
			if _, err = strconv.ParseFloat(lit, 64); err != nil {
				return nil, types.ErrorWrongSentence
			}
			lTok = "FLOAT"
			l.Lit = lit
		}

		isInt, err := global.IsInt(lit)
		if err != nil {
			return nil, types.ErrorWrongSentence
		}
		if isInt {
			if _, err = strconv.ParseInt(lit, 10, 64); err != nil {
				return nil, types.ErrorWrongSentence
			}
			lTok = "INT"
			l.Lit = lit
		}
		if !isFloat && !isInt {
			return nil, types.ErrorWrongSentence
		}
	}
	if rTok == "STRING" || rTok == "CHAR" {
		_, err := strconv.ParseInt(r.Lit, 10, 64)
		if err != nil {
			return nil, types.ErrorWrongSentence
		}
		rTok = "INT"
	}

	if lTok == "INT" && rTok == "INT" {
		exprTyp = "INT"
		left, _ = strconv.ParseInt(l.Lit, 10, 64)
		right, _ = strconv.ParseInt(r.Lit, 10, 64)
	}
	if lTok == "FLOAT" || rTok == "FLOAT" {
		exprTyp = "FLOAT"
		left, _ = strconv.ParseFloat(l.Lit, 64)
		right, _ = strconv.ParseFloat(r.Lit, 64)
	}

	if mTok == "*" {
		if exprTyp == "INT" {
			return &exprResult{Type: exprTyp, Tok: mTok, Value: left.(int64) * right.(int64)}, nil
		}
		if exprTyp == "FLOAT" {
			return &exprResult{Type: exprTyp, Tok: mTok, Value: left.(float64) * right.(float64)}, nil
		}
	}
	if mTok == "/" {
		if exprTyp == "INT" {
			return &exprResult{Type: exprTyp, Tok: mTok, Value: left.(int64) / right.(int64)}, nil
		}
		if exprTyp == "FLOAT" {
			return &exprResult{Type: exprTyp, Tok: mTok, Value: left.(float64) / right.(float64)}, nil
		}
	}
	if mTok == "+" {
		if exprTyp == "INT" {
			return &exprResult{Type: exprTyp, Tok: mTok, Value: left.(int64) + right.(int64)}, nil
		}
		if exprTyp == "FLOAT" {
			return &exprResult{Type: exprTyp, Tok: mTok, Value: left.(float64) + right.(float64)}, nil
		}
	}
	if mTok == "-" {
		if exprTyp == "INT" {
			return &exprResult{Type: exprTyp, Tok: mTok, Value: left.(int64) - right.(int64)}, nil
		}
		if exprTyp == "FLOAT" {
			return &exprResult{Type: exprTyp, Tok: mTok, Value: left.(float64) - right.(float64)}, nil
		}
	}
	if mTok == "%" && exprTyp == "INT" {
		return &exprResult{Type: exprTyp, Tok: mTok, Value: left.(int64) % right.(int64)}, nil
	}
	if mTok == "^" && exprTyp == "INT" {
		return &exprResult{Type: exprTyp, Tok: mTok, Value: left.(int64) ^ right.(int64)}, nil
	}
	if mTok == "|" && exprTyp == "INT" {
		return &exprResult{Type: exprTyp, Tok: mTok, Value: left.(int64) | right.(int64)}, nil
	}
	if mTok == "&" && exprTyp == "INT" {
		return &exprResult{Type: exprTyp, Tok: mTok, Value: left.(int64) & right.(int64)}, nil
	}
	return nil, types.ErrorWrongSentence
}

// findStrInfrontSymbool 查找指定符号之前的数据
func findStrInfrontSymbool(sym string, src []*global.Structure) ([]*global.Structure, int) {
	for k, v := range src {
		if sym == v.Tok {
			return src[:k], k
		}
	}
	return nil, -1
}

func inArray(sep string, arr []string) bool {
	for _, v := range arr {
		if sep == v {
			return true
		}
	}
	return false
}

func (r *Expression) FindFunction(expr []*global.Structure, pos string, foundAndOr bool) ([]*global.Structure, error) {
	count := 0
	found := false
	foundFNKey := -1
	fns := []*global.Structure{}
	for k, v := range expr {
		if v.Tok == "(" {
			if foundFNKey == -1 && k > 0 && expr[k-1].Tok == "IDENT" {
				foundFNKey = k
				fns = append(fns, expr[k-1])
			}
			found = true
			count++
		} else if v.Tok == ")" {
			fns = append(fns, v)
			count--
		}
		if count > 0 {
			fns = append(fns, v)
		}
		if count == 0 && found {
			if foundFNKey != -1 {
				rv, err := r.parseB(fns, pos, foundAndOr)
				if err != nil {
					return nil, err
				}
				if foundFNKey > 1 {
					expr = append(expr[:foundFNKey-1], rv)
				}
				if k+1 < len(expr) {
					expr = append(expr, expr[k+1:]...)
				}
			}
			break
		}
	}
	return expr, nil
}
