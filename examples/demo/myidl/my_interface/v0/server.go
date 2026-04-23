package my_interface

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

// my_interface server interface.
type MyInterfaceServer interface {

	// TestCall operation.
	TestCall(context.Context, *TestCallRequest) (*TestCallResponse, error)
}

func RegisterMyInterfaceServer(conn dcerpc.Conn, o MyInterfaceServer, opts ...dcerpc.Option) {
	conn.RegisterServer(NewMyInterfaceServerHandle(o), append(opts, dcerpc.WithAbstractSyntax(MyInterfaceSyntaxV0_0))...)
}

func NewMyInterfaceServerHandle(o MyInterfaceServer) dcerpc.ServerHandle {
	return func(ctx context.Context, opNum int, r ndr.Reader) (dcerpc.Operation, error) {
		return MyInterfaceServerHandle(ctx, o, opNum, r)
	}
}

func MyInterfaceServerHandle(ctx context.Context, o MyInterfaceServer, opNum int, r ndr.Reader) (dcerpc.Operation, error) {
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

// Unimplemented my_interface
type UnimplementedMyInterfaceServer struct {
}

func (UnimplementedMyInterfaceServer) TestCall(context.Context, *TestCallRequest) (*TestCallResponse, error) {
	return nil, dcerpc.ErrNotImplemented
}

var _ MyInterfaceServer = (*UnimplementedMyInterfaceServer)(nil)
