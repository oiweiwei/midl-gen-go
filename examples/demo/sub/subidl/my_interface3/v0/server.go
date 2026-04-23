package my_interface3

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

// my_interface3 server interface.
type MyInterface3Server interface {

	// TestCall operation.
	TestCall(context.Context, *TestCallRequest) (*TestCallResponse, error)
}

func RegisterMyInterface3Server(conn dcerpc.Conn, o MyInterface3Server, opts ...dcerpc.Option) {
	conn.RegisterServer(NewMyInterface3ServerHandle(o), append(opts, dcerpc.WithAbstractSyntax(MyInterface3SyntaxV0_0))...)
}

func NewMyInterface3ServerHandle(o MyInterface3Server) dcerpc.ServerHandle {
	return func(ctx context.Context, opNum int, r ndr.Reader) (dcerpc.Operation, error) {
		return MyInterface3ServerHandle(ctx, o, opNum, r)
	}
}

func MyInterface3ServerHandle(ctx context.Context, o MyInterface3Server, opNum int, r ndr.Reader) (dcerpc.Operation, error) {
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

// Unimplemented my_interface3
type UnimplementedMyInterface3Server struct {
}

func (UnimplementedMyInterface3Server) TestCall(context.Context, *TestCallRequest) (*TestCallResponse, error) {
	return nil, dcerpc.ErrNotImplemented
}

var _ MyInterface3Server = (*UnimplementedMyInterface3Server)(nil)
