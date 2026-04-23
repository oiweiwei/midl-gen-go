package idcominterface

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf16"

	dcerpc "github.com/oiweiwei/go-msrpc/dcerpc"
	uuid "github.com/oiweiwei/go-msrpc/midl/uuid"
	dcom "github.com/oiweiwei/go-msrpc/msrpc/dcom"
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
	_ = dcom.GoPackage
	_ = iwbemobjectsink.GoPackage
)

var (
	// import guard
	GoPackage = "myidl"
)

var (
	// IDcomInterface interface identifier 6bffd098-a112-3610-9833-012892020162
	IDcomInterfaceIID = &dcom.IID{Data1: 0x6bffd098, Data2: 0xa112, Data3: 0x3610, Data4: []byte{0x98, 0x33, 0x01, 0x28, 0x92, 0x02, 0x01, 0x62}}
	// Syntax UUID
	IDcomInterfaceSyntaxUUID = &uuid.UUID{TimeLow: 0x6bffd098, TimeMid: 0xa112, TimeHiAndVersion: 0x3610, ClockSeqHiAndReserved: 0x98, ClockSeqLow: 0x33, Node: [6]uint8{0x1, 0x28, 0x92, 0x2, 0x1, 0x62}}
	// Syntax ID
	IDcomInterfaceSyntaxV0_0 = &dcerpc.SyntaxID{IfUUID: IDcomInterfaceSyntaxUUID, IfVersionMajor: 0, IfVersionMinor: 0}
)

// IDcomInterface interface.
type IDcomInterfaceClient interface {

	// IWbemObjectSink retrieval method.
	ObjectSink() iwbemobjectsink.ObjectSinkClient

	// TestDCOMCallWithBase operation.
	TestDcomCallWithBase(context.Context, *TestDcomCallWithBaseRequest, ...dcerpc.CallOption) (*TestDcomCallWithBaseResponse, error)

	// AlterContext alters the client context.
	AlterContext(context.Context, ...dcerpc.Option) error

	// Conn returns the client connection (unsafe)
	Conn() dcerpc.Conn

	// IPID sets the object interface identifier.
	IPID(context.Context, *dcom.IPID) IDcomInterfaceClient
}

type xxx_DefaultIDcomInterfaceClient struct {
	iwbemobjectsink.ObjectSinkClient
	cc   dcerpc.Conn
	ipid *dcom.IPID
}

func (o *xxx_DefaultIDcomInterfaceClient) ObjectSink() iwbemobjectsink.ObjectSinkClient {
	return o.ObjectSinkClient
}

func (o *xxx_DefaultIDcomInterfaceClient) TestDcomCallWithBase(ctx context.Context, in *TestDcomCallWithBaseRequest, opts ...dcerpc.CallOption) (*TestDcomCallWithBaseResponse, error) {
	op := in.xxx_ToOp(ctx, nil)
	if _, ok := dcom.HasIPID(opts); !ok {
		if o.ipid != nil {
			opts = append(opts, dcom.WithIPID(o.ipid))
		} else {
			return nil, fmt.Errorf("%s: ipid is missing", op.OpName())
		}
	}
	if err := o.cc.Invoke(ctx, op, opts...); err != nil {
		return nil, err
	}
	out := &TestDcomCallWithBaseResponse{}
	out.xxx_FromOp(ctx, op)
	if op.Return != int32(0) {
		return out, fmt.Errorf("%s: %w", op.OpName(), o.cc.Error(ctx, op.Return))
	}
	return out, nil
}

func (o *xxx_DefaultIDcomInterfaceClient) AlterContext(ctx context.Context, opts ...dcerpc.Option) error {
	return o.cc.AlterContext(ctx, opts...)
}

func (o *xxx_DefaultIDcomInterfaceClient) Conn() dcerpc.Conn {
	return o.cc
}

func (o *xxx_DefaultIDcomInterfaceClient) IPID(ctx context.Context, ipid *dcom.IPID) IDcomInterfaceClient {
	if ipid == nil {
		ipid = &dcom.IPID{}
	}
	return &xxx_DefaultIDcomInterfaceClient{
		ObjectSinkClient: o.ObjectSinkClient.IPID(ctx, ipid),
		cc:               o.cc,
		ipid:             ipid,
	}
}

func NewIDcomInterfaceClient(ctx context.Context, cc dcerpc.Conn, opts ...dcerpc.Option) (IDcomInterfaceClient, error) {
	var err error
	if !dcom.IsSuperclass(opts) {
		cc, err = cc.Bind(ctx, append(opts, dcerpc.WithAbstractSyntax(IDcomInterfaceSyntaxV0_0))...)
		if err != nil {
			return nil, err
		}
	}
	base, err := iwbemobjectsink.NewObjectSinkClient(ctx, cc, append(opts, dcom.Superclass(cc))...)
	if err != nil {
		return nil, err
	}
	ipid, ok := dcom.HasIPID(opts)
	if ok {
		base = base.IPID(ctx, ipid)
	}
	return &xxx_DefaultIDcomInterfaceClient{
		ObjectSinkClient: base,
		cc:               cc,
		ipid:             ipid,
	}, nil
}

// xxx_TestDcomCallWithBaseOperation structure represents the TestDCOMCallWithBase operation
type xxx_TestDcomCallWithBaseOperation struct {
	This   *dcom.ORPCThis `idl:"name:This" json:"this"`
	That   *dcom.ORPCThat `idl:"name:That" json:"that"`
	Return int32          `idl:"name:Return" json:"return"`
}

// OpNum returns the operation number of TestDCOMCallWithBase operation.
func (o *xxx_TestDcomCallWithBaseOperation) OpNum() int { return 5 }

// OpName returns the operation name of TestDCOMCallWithBase operation.
func (o *xxx_TestDcomCallWithBaseOperation) OpName() string {
	return "/IDcomInterface/v0/TestDCOMCallWithBase"
}

func (o *xxx_TestDcomCallWithBaseOperation) xxx_PrepareRequestPayload(ctx context.Context) error {
	if hook, ok := (interface{})(o).(interface{ AfterPrepareRequestPayload(context.Context) error }); ok {
		if err := hook.AfterPrepareRequestPayload(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (o *xxx_TestDcomCallWithBaseOperation) MarshalNDRRequest(ctx context.Context, w ndr.Writer) error {
	if err := o.xxx_PrepareRequestPayload(ctx); err != nil {
		return err
	}
	// This {in} (1:{alias=ORPCTHIS}(struct))
	{
		if o.This != nil {
			if err := o.This.MarshalNDR(ctx, w); err != nil {
				return err
			}
		} else {
			if err := (&dcom.ORPCThis{}).MarshalNDR(ctx, w); err != nil {
				return err
			}
		}
		if err := w.WriteDeferred(); err != nil {
			return err
		}
	}
	return nil
}

func (o *xxx_TestDcomCallWithBaseOperation) UnmarshalNDRRequest(ctx context.Context, w ndr.Reader) error {
	// This {in} (1:{alias=ORPCTHIS}(struct))
	{
		if o.This == nil {
			o.This = &dcom.ORPCThis{}
		}
		if err := o.This.UnmarshalNDR(ctx, w); err != nil {
			return err
		}
		if err := w.ReadDeferred(); err != nil {
			return err
		}
	}
	return nil
}

func (o *xxx_TestDcomCallWithBaseOperation) xxx_PrepareResponsePayload(ctx context.Context) error {
	if hook, ok := (interface{})(o).(interface{ AfterPrepareResponsePayload(context.Context) error }); ok {
		if err := hook.AfterPrepareResponsePayload(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (o *xxx_TestDcomCallWithBaseOperation) MarshalNDRResponse(ctx context.Context, w ndr.Writer) error {
	if err := o.xxx_PrepareResponsePayload(ctx); err != nil {
		return err
	}
	// That {out} (1:{alias=ORPCTHAT}(struct))
	{
		if o.That != nil {
			if err := o.That.MarshalNDR(ctx, w); err != nil {
				return err
			}
		} else {
			if err := (&dcom.ORPCThat{}).MarshalNDR(ctx, w); err != nil {
				return err
			}
		}
		if err := w.WriteDeferred(); err != nil {
			return err
		}
	}
	// Return {out} (1:{alias=HRESULT}(int32))
	{
		if err := w.WriteData(o.Return); err != nil {
			return err
		}
	}
	return nil
}

func (o *xxx_TestDcomCallWithBaseOperation) UnmarshalNDRResponse(ctx context.Context, w ndr.Reader) error {
	// That {out} (1:{alias=ORPCTHAT}(struct))
	{
		if o.That == nil {
			o.That = &dcom.ORPCThat{}
		}
		if err := o.That.UnmarshalNDR(ctx, w); err != nil {
			return err
		}
		if err := w.ReadDeferred(); err != nil {
			return err
		}
	}
	// Return {out} (1:{alias=HRESULT}(int32))
	{
		if err := w.ReadData(&o.Return); err != nil {
			return err
		}
	}
	return nil
}

// TestDcomCallWithBaseRequest structure represents the TestDCOMCallWithBase operation request
type TestDcomCallWithBaseRequest struct {
	// This: ORPCTHIS structure that is used to send ORPC extension data to the server.
	This *dcom.ORPCThis `idl:"name:This" json:"this"`
}

func (o *TestDcomCallWithBaseRequest) xxx_ToOp(ctx context.Context, op *xxx_TestDcomCallWithBaseOperation) *xxx_TestDcomCallWithBaseOperation {
	if op == nil {
		op = &xxx_TestDcomCallWithBaseOperation{}
	}
	if o == nil {
		return op
	}
	op.This = o.This
	return op
}

func (o *TestDcomCallWithBaseRequest) xxx_FromOp(ctx context.Context, op *xxx_TestDcomCallWithBaseOperation) {
	if o == nil {
		return
	}
	o.This = op.This
}
func (o *TestDcomCallWithBaseRequest) MarshalNDR(ctx context.Context, w ndr.Writer) error {
	return o.xxx_ToOp(ctx, nil).MarshalNDRRequest(ctx, w)
}
func (o *TestDcomCallWithBaseRequest) UnmarshalNDR(ctx context.Context, r ndr.Reader) error {
	_o := &xxx_TestDcomCallWithBaseOperation{}
	if err := _o.UnmarshalNDRRequest(ctx, r); err != nil {
		return err
	}
	o.xxx_FromOp(ctx, _o)
	return nil
}

// MakeTestDcomCallWithBaseRequest build a response structure from the given request structure.
func (o *TestDcomCallWithBaseRequest) MakeResponse() *TestDcomCallWithBaseResponse {
	return &TestDcomCallWithBaseResponse{}
}

// OpNum returns the operation number of TestDCOMCallWithBase operation.
func (o *TestDcomCallWithBaseRequest) OpNum() int { return 5 }

// OpName returns the operation name of TestDCOMCallWithBase operation.
func (o *TestDcomCallWithBaseRequest) OpName() string {
	return "/IDcomInterface/v0/TestDCOMCallWithBase"
}

// TestDcomCallWithBaseResponse structure represents the TestDCOMCallWithBase operation response
type TestDcomCallWithBaseResponse struct {
	// That: ORPCTHAT structure that is used to return ORPC extension data to the client.
	That *dcom.ORPCThat `idl:"name:That" json:"that"`
	// Return: The TestDCOMCallWithBase return value.
	Return int32 `idl:"name:Return" json:"return"`
}

func (o *TestDcomCallWithBaseResponse) xxx_ToOp(ctx context.Context, op *xxx_TestDcomCallWithBaseOperation) *xxx_TestDcomCallWithBaseOperation {
	if op == nil {
		op = &xxx_TestDcomCallWithBaseOperation{}
	}
	if o == nil {
		return op
	}
	op.That = o.That
	op.Return = o.Return
	return op
}

func (o *TestDcomCallWithBaseResponse) xxx_FromOp(ctx context.Context, op *xxx_TestDcomCallWithBaseOperation) {
	if o == nil {
		return
	}
	o.That = op.That
	o.Return = op.Return
}
func (o *TestDcomCallWithBaseResponse) MarshalNDR(ctx context.Context, w ndr.Writer) error {
	return o.xxx_ToOp(ctx, nil).MarshalNDRResponse(ctx, w)
}
func (o *TestDcomCallWithBaseResponse) UnmarshalNDR(ctx context.Context, r ndr.Reader) error {
	_o := &xxx_TestDcomCallWithBaseOperation{}
	if err := _o.UnmarshalNDRResponse(ctx, r); err != nil {
		return err
	}
	o.xxx_FromOp(ctx, _o)
	return nil
}
