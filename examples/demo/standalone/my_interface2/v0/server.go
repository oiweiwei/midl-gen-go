package my_interface2

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf16"

	dcerpc "github.com/oiweiwei/go-msrpc/dcerpc"
	uuid "github.com/oiweiwei/go-msrpc/midl/uuid"
	ndr "github.com/oiweiwei/go-msrpc/ndr"
)

var (
	_ = context.Background
	_ = fmt.Errorf
	_ = utf16.Encode
	_ = strings.TrimPrefix
	_ = ndr.ZeroString
	_ = (*uuid.UUID)(nil)
	_ = (*dcerpc.SyntaxID)(nil)
)

// my_interface2 server interface.
type MyInterface2Server interface {

	// TestCall operation.
	TestCall(context.Context, *TestCallRequest) (*TestCallResponse, error)
}

func RegisterMyInterface2Server(conn dcerpc.Conn, o MyInterface2Server, opts ...dcerpc.Option) {
	conn.RegisterServer(NewMyInterface2ServerHandle(o), append(opts, dcerpc.WithAbstractSyntax(MyInterface2SyntaxV0_0))...)
}

func NewMyInterface2ServerHandle(o MyInterface2Server) dcerpc.ServerHandle {
	return func(ctx context.Context, opNum int, r ndr.Reader) (dcerpc.Operation, error) {
		return MyInterface2ServerHandle(ctx, o, opNum, r)
	}
}

func MyInterface2ServerHandle(ctx context.Context, o MyInterface2Server, opNum int, r ndr.Reader) (dcerpc.Operation, error) {
	switch opNum {
	case 0: // TestCall
		op := &xxx_TestCallOperation{}
		if err := op.UnmarshalNDRRequest(ctx, r); err != nil {
			return nil, err
		}
		req := &TestCallRequest{}
		req.xxx_FromOp(ctx, op)
		resp, err := o.TestCall(ctx, req)
		return resp.xxx_ToOp(ctx, op), err
	}
	return nil, nil
}

// Unimplemented my_interface2
type UnimplementedMyInterface2Server struct {
}

func (UnimplementedMyInterface2Server) TestCall(context.Context, *TestCallRequest) (*TestCallResponse, error) {
	return nil, dcerpc.ErrNotImplemented
}

var _ MyInterface2Server = (*UnimplementedMyInterface2Server)(nil)
