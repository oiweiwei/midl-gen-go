package my_interface

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf16"

	dcerpc "github.com/oiweiwei/go-msrpc/dcerpc"
	uuid "github.com/oiweiwei/go-msrpc/midl/uuid"
	dtyp "github.com/oiweiwei/go-msrpc/msrpc/dtyp"
	ndr "github.com/oiweiwei/go-msrpc/ndr"
	myidl "github.com/oiweiwei/midl-gen-go/examples/demo/myidl"
)

var (
	_ = context.Background
	_ = fmt.Errorf
	_ = utf16.Encode
	_ = strings.TrimPrefix
	_ = ndr.ZeroString
	_ = (*uuid.UUID)(nil)
	_ = (*dcerpc.SyntaxID)(nil)
	_ = dtyp.GoPackage
	_ = myidl.GoPackage
)

var (
	// import guard
	GoPackage = "myidl"
)

var (
	// Syntax UUID
	MyInterfaceSyntaxUUID = &uuid.UUID{TimeLow: 0x6bffd098, TimeMid: 0xa112, TimeHiAndVersion: 0x3610, ClockSeqHiAndReserved: 0x98, ClockSeqLow: 0x33, Node: [6]uint8{0x1, 0x28, 0x92, 0x2, 0x1, 0x62}}
	// Syntax ID
	MyInterfaceSyntaxV0_0 = &dcerpc.SyntaxID{IfUUID: MyInterfaceSyntaxUUID, IfVersionMajor: 0, IfVersionMinor: 0}
)

// my_interface interface.
type MyInterfaceClient interface {

	// TestCall operation.
	TestCall(context.Context, *TestCallRequest, ...dcerpc.CallOption) (*TestCallResponse, error)

	// AlterContext alters the client context.
	AlterContext(context.Context, ...dcerpc.Option) error

	// Conn returns the client connection (unsafe)
	Conn() dcerpc.Conn
}

type xxx_DefaultMyInterfaceClient struct {
	cc dcerpc.Conn
}

func (o *xxx_DefaultMyInterfaceClient) TestCall(ctx context.Context, in *TestCallRequest, opts ...dcerpc.CallOption) (*TestCallResponse, error) {
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

func (o *xxx_DefaultMyInterfaceClient) AlterContext(ctx context.Context, opts ...dcerpc.Option) error {
	return o.cc.AlterContext(ctx, opts...)
}

func (o *xxx_DefaultMyInterfaceClient) Conn() dcerpc.Conn {
	return o.cc
}

func NewMyInterfaceClient(ctx context.Context, cc dcerpc.Conn, opts ...dcerpc.Option) (MyInterfaceClient, error) {
	cc, err := cc.Bind(ctx, append(opts, dcerpc.WithAbstractSyntax(MyInterfaceSyntaxV0_0))...)
	if err != nil {
		return nil, err
	}
	return &xxx_DefaultMyInterfaceClient{cc: cc}, nil
}

// xxx_TestCallOperation structure represents the TestCall operation
type xxx_TestCallOperation struct {
	Input                 string                   `idl:"name:input;string;pointer:unique" json:"input"`
	Input2                *dtyp.UnicodeString      `idl:"name:input2;pointer:unique" json:"input2"`
	Output                string                   `idl:"name:output;string;pointer:unique" json:"output"`
	MyStructOutput        *myidl.MyInterfaceStruct `idl:"name:my_struct_output;pointer:unique" json:"my_struct_output"`
	MyUnicodeStringOutput *myidl.MyUnicodeString   `idl:"name:my_unicode_string_output;pointer:unique" json:"my_unicode_string_output"`
	Return                uint32                   `idl:"name:Return" json:"return"`
}

// OpNum returns the operation number of TestCall operation.
func (o *xxx_TestCallOperation) OpNum() int { return 0 }

// OpName returns the operation name of TestCall operation.
func (o *xxx_TestCallOperation) OpName() string { return "/my_interface/v0/TestCall" }

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
	// input2 {in} (1:{pointer=unique, alias=PRPC_UNICODE_STRING}*(1))(2:{alias=RPC_UNICODE_STRING}(struct))
	{
		if o.Input2 != nil {
			_ptr_input2 := ndr.MarshalNDRFunc(func(ctx context.Context, w ndr.Writer) error {
				if o.Input2 != nil {
					if err := o.Input2.MarshalNDR(ctx, w); err != nil {
						return err
					}
				} else {
					if err := (&dtyp.UnicodeString{}).MarshalNDR(ctx, w); err != nil {
						return err
					}
				}
				return nil
			})
			if err := w.WritePointer(&o.Input2, _ptr_input2); err != nil {
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
	// input2 {in} (1:{pointer=unique, alias=PRPC_UNICODE_STRING}*(1))(2:{alias=RPC_UNICODE_STRING}(struct))
	{
		_ptr_input2 := ndr.UnmarshalNDRFunc(func(ctx context.Context, w ndr.Reader) error {
			if o.Input2 == nil {
				o.Input2 = &dtyp.UnicodeString{}
			}
			if err := o.Input2.UnmarshalNDR(ctx, w); err != nil {
				return err
			}
			return nil
		})
		_s_input2 := func(ptr interface{}) { o.Input2 = *ptr.(**dtyp.UnicodeString) }
		if err := w.ReadPointer(&o.Input2, _s_input2, _ptr_input2); err != nil {
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
	// my_struct_output {out} (1:{pointer=unique, alias=PMY_INTERFACE_STRUCT}*(1))(2:{alias=MY_INTERFACE_STRUCT}(struct))
	{
		if o.MyStructOutput != nil {
			_ptr_my_struct_output := ndr.MarshalNDRFunc(func(ctx context.Context, w ndr.Writer) error {
				if o.MyStructOutput != nil {
					if err := o.MyStructOutput.MarshalNDR(ctx, w); err != nil {
						return err
					}
				} else {
					if err := (&myidl.MyInterfaceStruct{}).MarshalNDR(ctx, w); err != nil {
						return err
					}
				}
				return nil
			})
			if err := w.WritePointer(&o.MyStructOutput, _ptr_my_struct_output); err != nil {
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
	// my_unicode_string_output {out} (1:{pointer=unique, alias=PMY_UNICODE_STRING}*(1))(2:{alias=MY_UNICODE_STRING, names=RPC_UNICODE_STRING}(struct))
	{
		if o.MyUnicodeStringOutput != nil {
			_ptr_my_unicode_string_output := ndr.MarshalNDRFunc(func(ctx context.Context, w ndr.Writer) error {
				if o.MyUnicodeStringOutput != nil {
					if err := o.MyUnicodeStringOutput.MarshalNDR(ctx, w); err != nil {
						return err
					}
				} else {
					if err := (&myidl.MyUnicodeString{}).MarshalNDR(ctx, w); err != nil {
						return err
					}
				}
				return nil
			})
			if err := w.WritePointer(&o.MyUnicodeStringOutput, _ptr_my_unicode_string_output); err != nil {
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
	// my_struct_output {out} (1:{pointer=unique, alias=PMY_INTERFACE_STRUCT}*(1))(2:{alias=MY_INTERFACE_STRUCT}(struct))
	{
		_ptr_my_struct_output := ndr.UnmarshalNDRFunc(func(ctx context.Context, w ndr.Reader) error {
			if o.MyStructOutput == nil {
				o.MyStructOutput = &myidl.MyInterfaceStruct{}
			}
			if err := o.MyStructOutput.UnmarshalNDR(ctx, w); err != nil {
				return err
			}
			return nil
		})
		_s_my_struct_output := func(ptr interface{}) { o.MyStructOutput = *ptr.(**myidl.MyInterfaceStruct) }
		if err := w.ReadPointer(&o.MyStructOutput, _s_my_struct_output, _ptr_my_struct_output); err != nil {
			return err
		}
		if err := w.ReadDeferred(); err != nil {
			return err
		}
	}
	// my_unicode_string_output {out} (1:{pointer=unique, alias=PMY_UNICODE_STRING}*(1))(2:{alias=MY_UNICODE_STRING, names=RPC_UNICODE_STRING}(struct))
	{
		_ptr_my_unicode_string_output := ndr.UnmarshalNDRFunc(func(ctx context.Context, w ndr.Reader) error {
			if o.MyUnicodeStringOutput == nil {
				o.MyUnicodeStringOutput = &myidl.MyUnicodeString{}
			}
			if err := o.MyUnicodeStringOutput.UnmarshalNDR(ctx, w); err != nil {
				return err
			}
			return nil
		})
		_s_my_unicode_string_output := func(ptr interface{}) { o.MyUnicodeStringOutput = *ptr.(**myidl.MyUnicodeString) }
		if err := w.ReadPointer(&o.MyUnicodeStringOutput, _s_my_unicode_string_output, _ptr_my_unicode_string_output); err != nil {
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
	Input  string              `idl:"name:input;string;pointer:unique" json:"input"`
	Input2 *dtyp.UnicodeString `idl:"name:input2;pointer:unique" json:"input2"`
}

func (o *TestCallRequest) xxx_ToOp(ctx context.Context, op *xxx_TestCallOperation) *xxx_TestCallOperation {
	if op == nil {
		op = &xxx_TestCallOperation{}
	}
	if o == nil {
		return op
	}
	op.Input = o.Input
	op.Input2 = o.Input2
	return op
}

func (o *TestCallRequest) xxx_FromOp(ctx context.Context, op *xxx_TestCallOperation) {
	if o == nil {
		return
	}
	o.Input = op.Input
	o.Input2 = op.Input2
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
func (o *TestCallRequest) OpName() string { return "/my_interface/v0/TestCall" }

// TestCallResponse structure represents the TestCall operation response
type TestCallResponse struct {
	Output                string                   `idl:"name:output;string;pointer:unique" json:"output"`
	MyStructOutput        *myidl.MyInterfaceStruct `idl:"name:my_struct_output;pointer:unique" json:"my_struct_output"`
	MyUnicodeStringOutput *myidl.MyUnicodeString   `idl:"name:my_unicode_string_output;pointer:unique" json:"my_unicode_string_output"`
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
	op.MyStructOutput = o.MyStructOutput
	op.MyUnicodeStringOutput = o.MyUnicodeStringOutput
	op.Return = o.Return
	return op
}

func (o *TestCallResponse) xxx_FromOp(ctx context.Context, op *xxx_TestCallOperation) {
	if o == nil {
		return
	}
	o.Output = op.Output
	o.MyStructOutput = op.MyStructOutput
	o.MyUnicodeStringOutput = op.MyUnicodeStringOutput
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
