package lit

import (
	fn "github.com/pywee/lit/function"
	"github.com/pywee/lit/global"
	"github.com/pywee/lit/types"
)

// parseExecFUNC 解析自定义函数调用
func parseExecFUNC(blocks []*global.Block, expr []*global.Structure, i int, rlen int) ([]*global.Block, int) {
	code := make([]*global.Structure, 0, 5)
	for j := i; j < rlen; j++ {
		exprJ := expr[j]
		if exprJ.Tok == ";" {
			blocks = append(blocks, &global.Block{
				Name: expr[0].Lit,
				Type: types.CodeTypeFunctionExec,
				Code: code,
			})
			i = j
			break
		}
		code = append(code, exprJ)
	}
	return blocks, i
}

// parseIdentFUNC 解析函数定义
func parseIdentFUNC(funcBlocks []*fn.FunctionInfo, expr []*global.Structure, i int, rlen int) ([]*fn.FunctionInfo, int, error) {
	var (
		bracket  uint8
		funcCode = make([]*global.Structure, 0, 10)
	)

	for j := i; j < rlen; j++ {
		exprJ := expr[j]
		funcCode = append(funcCode, exprJ)
		if exprJ.Tok == "{" {
			bracket++
		} else if exprJ.Tok == "}" {
			bracket--
			if bracket == 0 {
				if len(funcCode) < 6 {
					return nil, 0, types.ErrorFunctionIlligle
				}
				funcsParsed, err := cfn.ParseCutFunc(funcCode, exprJ.Position)
				if err != nil {
					return nil, 0, err
				}
				funcBlocks = append(funcBlocks, funcsParsed)
				i = j
				break
			}
		}
	}
	return funcBlocks, i, nil
}

// execCustomFunc 执行自定义函数
// 当自定义的函数被调用时才会调用此方法
// realArgValues 为函数被调用时得到的实参
func (r *expression) execCustomFunc(fni *fn.FunctionInfo, realArgValues []*global.Structure, pos string, innerVarInFuncParams map[string]*global.Structure) (*global.Structure, error) {
	// innerVar 函数体内的变量声明
	var innerVar = make(map[string]*global.Structure)

	// 以下场景需要在维护上下文 innerVarInFuncParams 局部变量
	for k, v := range innerVarInFuncParams {
		innerVar[k] = v
	}

	// 为形参赋值
	// 即: 将调用函数时传入的实参赋值给形参
	if err := r.setInnerVal(fni, realArgValues, pos, innerVar); err != nil {
		return nil, err
	}

	// 函数体代码解析
	// 递归回去 从头开始操作
	// FIXME.需要进一步检查参数上下文传递问题
	return r.initExpr(fni.CustFN, innerVar, nil)
}

// execInnerFunc 执行内置函数
func (r *expression) execInnerFunc(funcName string, expr []*global.Structure, pos string, innerVar map[string]*global.Structure) (*global.Structure, error) {
	fArgs := fn.GetInnerIdentedFunc(funcName)
	if fArgs == nil {
		return nil, types.ErrorNotFoundFunction
	}

	// 函数内参数检查
	// 获取传入执行函数的具体参数
	// 并将它们的结果值递归解析出来
	args := fn.GetFunctionArgList(expr)
	argsLen := len(args)
	if fArgs.MustAmount > argsLen {
		return nil, types.ErrorArgsNotEnough
	}
	if fArgs.MaxAmount != -1 && fArgs.MaxAmount < argsLen {
		return nil, types.ErrorTooManyArgs
	}

	// global.Output(args)

	// FIXME
	// get params after parsing
	// 汇总解析成功之后的实参数据
	// 传入回调函数进行实际执行
	// 当前仅支持内置函数
	var paramsList []*global.Structure
	for k, varg := range args {
		// FIXME
		// 函数中的实参表达式 实参可以是函数、变量、算术表达式等等
		// global.Output(varg)
		// return nil, nil

		rv, err := r.parse(varg, innerVar)
		if err != nil {
			return nil, err
		}

		if fArgs.MaxAmount != -1 {
			// 检查最终解析出来的参数值类型是否与函数要求的形参类型一致
			if fa := fArgs.Args[k]; fa.Type != types.INTERFACE && fa.Type != rv.Tok {
				// TODO
				// 参数[弱类型]支持
				// 参数[提前在形参中设置默认值]支持
				// fmt.Println(fa.Type, rv.Tok)
				return nil, types.ErrorFunctionArgsNotSuitable
			}
		}
		paramsList = append(paramsList, rv)
	}

	// global.Output(paramsList)

	fRet, err := fArgs.FN(pos, paramsList...)
	if err != nil {
		return nil, err
	}
	return fRet, err
}

// setInnerVal 解析函数体内的变量声明句子
// for example:
// a = 1;
// b = abc();
// c = a + b;
// return a+b+c
func (r *expression) setInnerVal(fni *fn.FunctionInfo, realArgValues []*global.Structure, pos string, innerVar map[string]*global.Structure) error {
	// 对比函数形参和实参
	realArgList := fn.GetFunctionArgList(realArgValues)
	rLen := len(realArgList)
	for k, nArg := range fni.Args {
		var thisArg = make([]*global.Structure, 0, 3)
		if k+1 > rLen {
			if nArg.Must {
				return types.ErrorArgsNotEnough
			}
			thisArg = []*global.Structure{{Tok: nArg.Type, Lit: nArg.Value}}
		} else {
			thisArg = realArgList[k]
		}

		// 解析传入的实参 因为实参可能也是函数
		realArgValueParsed, err := r.parse(thisArg, innerVar)
		if err != nil {
			return err
		}
		nArg.Value = realArgValueParsed.Lit
		innerVar[nArg.Name] = realArgValueParsed
	}
	return nil
}

// execFUNC 执行函数
func (r *expression) execFUNC(expr []*global.Structure, xArgs []*global.Structure, innerVar map[string]*global.Structure) (*global.Structure, error) {
	var (
		innerFunc  *fn.FunctionInfo
		customFunc *fn.FunctionInfo
		funcName   = expr[0].Lit
	)
	if customFunc = r.getIdentedCustomFunc(funcName); customFunc != nil {
		rv, err := r.execCustomFunc(customFunc, xArgs, "", innerVar)
		if err != nil {
			return nil, err
		}
		return rv, nil
	}
	if innerFunc = fn.GetInnerIdentedFunc(funcName); innerFunc != nil {
		// 查找是否有内置函数
		// global.Output(xArgs)
		rv, err := r.execInnerFunc(funcName, xArgs, "", innerVar)
		if err != nil {
			return nil, err
		}
		return rv, nil
	}
	return nil, types.ErrorNotFoundFunction
}
