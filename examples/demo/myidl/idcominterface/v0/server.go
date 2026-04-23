package idcominterface

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf16"

	dcerpc "github.com/oiweiwei/go-msrpc/dcerpc"
	uuid "github.com/oiweiwei/go-msrpc/midl/uuid"
	iwbemobjectsink "github.com/oiweiwei/go-msrpc/msrpc/dcom/wmi/iwbemobjectsink/v0"
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
	_ = iwbemobjectsink.GoPackage
)

// IDcomInterface server interface.
type IDcomInterfaceServer interface {

	// IWbemObjectSink base class.
	iwbemobjectsink.ObjectSinkServer

	// TestDCOMCallWithBase operation.
	TestDcomCallWithBase(context.Context, *TestDcomCallWithBaseRequest) (*TestDcomCallWithBaseResponse, error)
}

func RegisterIDcomInterfaceServer(conn dcerpc.Conn, o IDcomInterfaceServer, opts ...dcerpc.Option) {
	conn.RegisterServer(NewIDcomInterfaceServerHandle(o), append(opts, dcerpc.WithAbstractSyntax(IDcomInterfaceSyntaxV0_0))...)
}

func NewIDcomInterfaceServerHandle(o IDcomInterfaceServer) dcerpc.ServerHandle {
	return func(ctx context.Context, opNum int, r ndr.Reader) (dcerpc.Operation, error) {
		return IDcomInterfaceServerHandle(ctx, o, opNum, r)
	}
}

func IDcomInterfaceServerHandle(ctx context.Context, o IDcomInterfaceServer, opNum int, r ndr.Reader) (dcerpc.Operation, error) {
	if opNum < 5 {
		// IWbemObjectSink base method.
		return iwbemobjectsink.ObjectSinkServerHandle(ctx, o, opNum, r)
	}
	switch opNum {
	case 5: // TestDCOMCallWithBase
		op := &xxx_TestDcomCallWithBaseOperation{}
		if err := op.UnmarshalNDRRequest(ctx, r); err != nil {
			return nil, err
		}
		req := &TestDcomCallWithBaseRequest{}
		req.xxx_FromOp(ctx, op)
		resp, err := o.TestDcomCallWithBase(ctx, req)
		return resp.xxx_ToOp(ctx, op), err
	}
	return nil, nil
}

// Unimplemented IDcomInterface
type UnimplementedIDcomInterfaceServer struct {
	iwbemobjectsink.UnimplementedObjectSinkServer
}

func (UnimplementedIDcomInterfaceServer) TestDcomCallWithBase(context.Context, *TestDcomCallWithBaseRequest) (*TestDcomCallWithBaseResponse, error) {
	return nil, dcerpc.ErrNotImplemented
}

var _ IDcomInterfaceServer = (*UnimplementedIDcomInterfaceServer)(nil)
