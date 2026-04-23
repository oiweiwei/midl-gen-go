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

var (
	// import guard
	GoPackage = "sub/subidl"
)

var (
	// Syntax UUID
	MyInterface3SyntaxUUID = &uuid.UUID{TimeLow: 0x6bffd098, TimeMid: 0xa112, TimeHiAndVersion: 0x3610, ClockSeqHiAndReserved: 0x98, ClockSeqLow: 0x33, Node: [6]uint8{0x1, 0x28, 0x92, 0x2, 0x1, 0x63}}
	// Syntax ID
	MyInterface3SyntaxV0_0 = &dcerpc.SyntaxID{IfUUID: MyInterface3SyntaxUUID, IfVersionMajor: 0, IfVersionMinor: 0}
)

// my_interface3 interface.
type MyInterface3Client interface {

	// TestCall operation.
	TestCall(context.Context, *TestCallRequest, ...dcerpc.CallOption) (*TestCallResponse, error)

	// AlterContext alters the client context.
	AlterContext(context.Context, ...dcerpc.Option) error

	// Conn returns the client connection (unsafe)
	Conn() dcerpc.Conn
}

type xxx_DefaultMyInterface3Client struct {
	cc dcerpc.Conn
}

func (o *xxx_DefaultMyInterface3Client) TestCall(ctx context.Context, in *TestCallRequest, opts ...dcerpc.CallOption) (*TestCallResponse, error) {
	op := in.xxx_ToOp(ctx, nil)
	if err := o.cc.Invoke(ctx, op, opts...); err != nil {
		return nil, err
	}
	out := &TestCallResponse{}
	out.xxx_FromOp(ctx, op)
	if op.Return != uint32(0) {
		return out, fmt.Errorf("%s: %w", op.OpName(), o.cc.Error(ctx, op.Return))
	}
	return out, nil
}

func (o *xxx_DefaultMyInterface3Client) AlterContext(ctx context.Context, opts ...dcerpc.Option) error {
	return o.cc.AlterContext(ctx, opts...)
}

func (o *xxx_DefaultMyInterface3Client) Conn() dcerpc.Conn {
	return o.cc
}

func NewMyInterface3Client(ctx context.Context, cc dcerpc.Conn, opts ...dcerpc.Option) (MyInterface3Client, error) {
	cc, err := cc.Bind(ctx, append(opts, dcerpc.WithAbstractSyntax(MyInterface3SyntaxV0_0))...)
	if err != nil {
		return nil, err
	}
	return &xxx_DefaultMyInterface3Client{cc: cc}, nil
}

// xxx_TestCallOperation structure represents the TestCall operation
type xxx_TestCallOperation struct {
	Input  string `idl:"name:input;string;pointer:unique" json:"input"`
	Output string `idl:"name:output;string;pointer:unique" json:"output"`
	Return uint32 `idl:"name:Return" json:"return"`
}

// OpNum returns the operation number of TestCall operation.
func (o *xxx_TestCallOperation) OpNum() int { return 0 }

// OpName returns the operation name of TestCall operation.
func (o *xxx_TestCallOperation) OpName() string { return "/my_interface3/v0/TestCall" }

func (o *xxx_TestCallOperation) xxx_PrepareRequestPayload(ctx context.Context) error {
	if hook, ok := (interface{})(o).(interface{ AfterPrepareRequestPayload(context.Context) error }); ok {
		if err := hook.AfterPrepareRequestPayload(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (o *xxx_TestCallOperation) MarshalNDRRequest(ctx context.Context, w ndr.Writer) error {
	if err := o.xxx_PrepareRequestPayload(ctx); err != nil {
		return err
	}
	// input {in} (1:{string, pointer=unique}*(1)[dim:0,string,null](wchar))
	{
		if o.Input != "" {
			_ptr_input := ndr.MarshalNDRFunc(func(ctx context.Context, w ndr.Writer) error {
				if err := ndr.WriteUTF16NString(ctx, w, o.Input); err != nil {
					return err
				}
				return nil
			})
			if err := w.WritePointer(&o.Input, _ptr_input); err != nil {
				return err
			}
		} else {
			if err := w.WritePointer(nil); err != nil {
				return err
			}
		}
		if err := w.WriteDeferred(); err != nil {
			return err
		}
	}
	return nil
}

func (o *xxx_TestCallOperation) UnmarshalNDRRequest(ctx context.Context, w ndr.Reader) error {
	// input {in} (1:{string, pointer=unique}*(1)[dim:0,string,null](wchar))
	{
		_ptr_input := ndr.UnmarshalNDRFunc(func(ctx context.Context, w ndr.Reader) error {
			if err := ndr.ReadUTF16NString(ctx, w, &o.Input); err != nil {
				return err
			}
			return nil
		})
		_s_input := func(ptr interface{}) { o.Input = *ptr.(*string) }
		if err := w.ReadPointer(&o.Input, _s_input, _ptr_input); err != nil {
			return err
		}
		if err := w.ReadDeferred(); err != nil {
			return err
		}
	}
	return nil
}

func (o *xxx_TestCallOperation) xxx_PrepareResponsePayload(ctx context.Context) error {
	if hook, ok := (interface{})(o).(interface{ AfterPrepareResponsePayload(context.Context) error }); ok {
		if err := hook.AfterPrepareResponsePayload(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (o *xxx_TestCallOperation) MarshalNDRResponse(ctx context.Context, w ndr.Writer) error {
	if err := o.xxx_PrepareResponsePayload(ctx); err != nil {
		return err
	}
	// output {out} (1:{string, pointer=unique}*(1)[dim:0,string,null](wchar))
	{
		if o.Output != "" {
			_ptr_output := ndr.MarshalNDRFunc(func(ctx context.Context, w ndr.Writer) error {
				if err := ndr.WriteUTF16NString(ctx, w, o.Output); err != nil {
					return err
				}
				return nil
			})
			if err := w.WritePointer(&o.Output, _ptr_output); err != nil {
				return err
			}
		} else {
			if err := w.WritePointer(nil); err != nil {
				return err
			}
		}
		if err := w.WriteDeferred(); err != nil {
			return err
		}
	}
	// Return {out} (1:(error_status_t))
	{
		if err := w.WriteData(o.Return); err != nil {
			return err
		}
	}
	return nil
}

func (o *xxx_TestCallOperation) UnmarshalNDRResponse(ctx context.Context, w ndr.Reader) error {
	// output {out} (1:{string, pointer=unique}*(1)[dim:0,string,null](wchar))
	{
		_ptr_output := ndr.UnmarshalNDRFunc(func(ctx context.Context, w ndr.Reader) error {
			if err := ndr.ReadUTF16NString(ctx, w, &o.Output); err != nil {
				return err
			}
			return nil
		})
		_s_output := func(ptr interface{}) { o.Output = *ptr.(*string) }
		if err := w.ReadPointer(&o.Output, _s_output, _ptr_output); err != nil {
			return err
		}
		if err := w.ReadDeferred(); err != nil {
			return err
		}
	}
	// Return {out} (1:(error_status_t))
	{
		if err := w.ReadData(&o.Return); err != nil {
			return err
		}
	}
	return nil
}

// TestCallRequest structure represents the TestCall operation request
type TestCallRequest struct {
	Input string `idl:"name:input;string;pointer:unique" json:"input"`
}

func (o *TestCallRequest) xxx_ToOp(ctx context.Context, op *xxx_TestCallOperation) *xxx_TestCallOperation {
	if op == nil {
		op = &xxx_TestCallOperation{}
	}
	if o == nil {
		return op
	}
	op.Input = o.Input
	return op
}

func (o *TestCallRequest) xxx_FromOp(ctx context.Context, op *xxx_TestCallOperation) {
	if o == nil {
		return
	}
	o.Input = op.Input
}
func (o *TestCallRequest) MarshalNDR(ctx context.Context, w ndr.Writer) error {
	return o.xxx_ToOp(ctx, nil).MarshalNDRRequest(ctx, w)
}
func (o *TestCallRequest) UnmarshalNDR(ctx context.Context, r ndr.Reader) error {
	_o := &xxx_TestCallOperation{}
	if err := _o.UnmarshalNDRRequest(ctx, r); err != nil {
		return err
	}
	o.xxx_FromOp(ctx, _o)
	return nil
}

// MakeTestCallRequest build a response structure from the given request structure.
func (o *TestCallRequest) MakeResponse() *TestCallResponse {
	return &TestCallResponse{}
}

// OpNum returns the operation number of TestCall operation.
func (o *TestCallRequest) OpNum() int { return 0 }

// OpName returns the operation name of TestCall operation.
func (o *TestCallRequest) OpName() string { return "/my_interface3/v0/TestCall" }

// TestCallResponse structure represents the TestCall operation response
type TestCallResponse struct {
	Output string `idl:"name:output;string;pointer:unique" json:"output"`
	// Return: The TestCall return value.
	Return uint32 `idl:"name:Return" json:"return"`
}

func (o *TestCallResponse) xxx_ToOp(ctx context.Context, op *xxx_TestCallOperation) *xxx_TestCallOperation {
	if op == nil {
		op = &xxx_TestCallOperation{}
	}
	if o == nil {
		return op
	}
	op.Output = o.Output
	op.Return = o.Return
	return op
}

func (o *TestCallResponse) xxx_FromOp(ctx context.Context, op *xxx_TestCallOperation) {
	if o == nil {
		return
	}
	o.Output = op.Output
	o.Return = op.Return
}
func (o *TestCallResponse) MarshalNDR(ctx context.Context, w ndr.Writer) error {
	return o.xxx_ToOp(ctx, nil).MarshalNDRResponse(ctx, w)
}
func (o *TestCallResponse) UnmarshalNDR(ctx context.Context, r ndr.Reader) error {
	_o := &xxx_TestCallOperation{}
	if err := _o.UnmarshalNDRResponse(ctx, r); err != nil {
		return err
	}
	o.xxx_FromOp(ctx, _o)
	return nil
}
