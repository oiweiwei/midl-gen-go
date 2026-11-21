package emit

import (
	"regexp"
	"strings"

	"github.com/oiweiwei/midl-gen-go/midl"
)

var (
	notUsedOpRe = regexp.MustCompile("(NotUsedOnWire)|(NotImplemented)|(LSA_DP)|(LSA_TM)|(Unused)|(Opnum\\d+Reserved)|(Reserved_Opnum\\d+)|(Reserved\\d+)|(Opnum\\d+NotUsedByProtocol)")
)

func (p *Emitter) IsUnusedOp(n string) bool {
	return notUsedOpRe.MatchString(n)
}

var (
	AnyParam = 0
	InParam  = 1
	OutParam = 2
)

func ParamName(dir int) string {
	switch dir {
	case InParam:
		return "Request"
	case OutParam:
		return "Response"
	}

	return ""
}

func (p *Emitter) MethodName(ctx *Env, op *midl.Operation) string {
	n := GoName(ctx, op.Name)
	if strings.HasPrefix(op.Name, "_") {
		n = "_" + n
	}
	if op.Attrs.PropGet {
		n = "Get" + n
	}
	if op.Attrs.PropPut {
		n = "Set" + n
	}
	if op.Attrs.PropPutRef {
		n = "SetByRef" + n
	}
	return n
}

func (p *Emitter) DirName(ctx *Env, dir int) string {
	switch dir {
	case InParam:
		return "Request"
	case OutParam:
		return "Response"
	}
	return ""
}

func (p *Emitter) OpName(ctx *Env, op *midl.Operation, dir int) string {
	n := p.MethodName(ctx, op)
	if dir == InParam {
		n += "Request"
		return n
	} else if dir == OutParam {
		n += "Response"
		return n
	}
	return p.XXX() + n + "Operation"
}

func (p *Emitter) IsDir(ctx *Env, pdir midl.Direction, dir int) bool {
	return dir == AnyParam || (dir == InParam && pdir.In) || (dir == OutParam && pdir.Out)
}

func (p *Emitter) ReturnValue() string {
	return "Return"
}

func (p *Emitter) OperationReturnValue(ctx *Env, op *midl.Operation) *midl.Param {
	return &midl.Param{
		Name:  p.ReturnValue(),
		Type:  op.Type,
		Attrs: &midl.ParamAttr{FieldAttr: &midl.FieldAttr{}, Direction: midl.Direction{Out: true}},
	}
}

func (p *Emitter) OperationParams(ctx *Env, op *midl.Operation) []*midl.Param {

	params := make([]*midl.Param, len(op.Params))
	copy(params, op.Params)

	if !op.Type.Is(midl.TypeVoid) {
		params = append(params, p.OperationReturnValue(ctx, op))
	}

	if !ctx.Interface.IsObject() {
		return params
	}

	return append([]*midl.Param{
		{
			Name:  "This",
			Type:  p.Session.LookupType("ORPCTHIS"),
			Attrs: &midl.ParamAttr{FieldAttr: &midl.FieldAttr{}, Direction: midl.Direction{In: true}},
		},
		{
			Name:  "That",
			Type:  p.Session.LookupType("ORPCTHAT"),
			Attrs: &midl.ParamAttr{FieldAttr: &midl.FieldAttr{}, Direction: midl.Direction{Out: true}},
		},
	}, params...)
}

func (p *Emitter) GetExprIdents(ctx *Env, expr ...midl.Expr) []string {
	ret := make([]string, 0, len(expr))
	for _, expr := range expr {
		val, ok := expr.FirstIdent()
		if ok {
			ret = append(ret, val)
		}
	}
	return ret
}
