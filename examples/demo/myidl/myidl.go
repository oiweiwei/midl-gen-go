// The myidl package implements the MYIDL client protocol.
package myidl

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf16"

	dcerpc "github.com/oiweiwei/go-msrpc/dcerpc"
	uuid "github.com/oiweiwei/go-msrpc/midl/uuid"
	dcetypes "github.com/oiweiwei/go-msrpc/msrpc/dcetypes"
	dcom "github.com/oiweiwei/go-msrpc/msrpc/dcom"
	dtyp "github.com/oiweiwei/go-msrpc/msrpc/dtyp"
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
	_ = dtyp.GoPackage
	_ = dcetypes.GoPackage
	_ = dcom.GoPackage
)

var (
	// import guard
	GoPackage = "myidl"
)

// MyUnicodeString structure represents MY_UNICODE_STRING RPC structure.
type MyUnicodeString dtyp.UnicodeString

func (o *MyUnicodeString) UnicodeString() *dtyp.UnicodeString { return (*dtyp.UnicodeString)(o) }

func (o *MyUnicodeString) xxx_PreparePayload(ctx context.Context) error {
	if err := ndr.BeforePreparePayload(ctx, o); err != nil {
		return err
	}
	if o.Buffer != "" && o.MaximumLength == 0 {
		o.MaximumLength = uint16((ndr.UTF16Len(o.Buffer) * 2))
	}
	if o.Buffer != "" && o.Length == 0 {
		o.Length = uint16((ndr.UTF16Len(o.Buffer) * 2))
	}
	if err := ndr.AfterPreparePayload(ctx, o); err != nil {
		return err
	}
	return nil
}
func (o *MyUnicodeString) MarshalNDR(ctx context.Context, w ndr.Writer) error {
	if err := o.xxx_PreparePayload(ctx); err != nil {
		return err
	}
	if err := w.WriteAlign(7); err != nil {
		return err
	}
	if err := w.WriteData(o.Length); err != nil {
		return err
	}
	if err := w.WriteData(o.MaximumLength); err != nil {
		return err
	}
	if o.Buffer != "" || (o.MaximumLength/2) > 0 {
		_ptr_Buffer := ndr.MarshalNDRFunc(func(ctx context.Context, w ndr.Writer) error {
			dimSize1 := uint64((o.MaximumLength / 2))
			if err := w.WriteSize(dimSize1); err != nil {
				return err
			}
			sizeInfo := []uint64{
				dimSize1,
			}
			dimLength1 := uint64((o.Length / 2))
			if dimLength1 > sizeInfo[0] {
				dimLength1 = sizeInfo[0]
			} else {
				sizeInfo[0] = dimLength1
			}
			if err := w.WriteSize(0); err != nil {
				return err
			}
			if err := w.WriteSize(dimLength1); err != nil {
				return err
			}
			_Buffer_buf := utf16.Encode([]rune(o.Buffer))
			if uint64(len(_Buffer_buf)) > sizeInfo[0] {
				_Buffer_buf = _Buffer_buf[:sizeInfo[0]]
			}
			for i1 := range _Buffer_buf {
				i1 := i1
				if uint64(i1) >= sizeInfo[0] {
					break
				}
				if err := w.WriteData(_Buffer_buf[i1]); err != nil {
					return err
				}
			}
			for i1 := len(_Buffer_buf); uint64(i1) < sizeInfo[0]; i1++ {
				if err := w.WriteData(uint16(0)); err != nil {
					return err
				}
			}
			return nil
		})
		if err := w.WritePointer(&o.Buffer, _ptr_Buffer); err != nil {
			return err
		}
	} else {
		if err := w.WritePointer(nil); err != nil {
			return err
		}
	}
	return nil
}
func (o *MyUnicodeString) UnmarshalNDR(ctx context.Context, w ndr.Reader) error {
	if err := w.ReadAlign(7); err != nil {
		return err
	}
	if err := w.ReadData(&o.Length); err != nil {
		return err
	}
	if err := w.ReadData(&o.MaximumLength); err != nil {
		return err
	}
	_ptr_Buffer := ndr.UnmarshalNDRFunc(func(ctx context.Context, w ndr.Reader) error {
		sizeInfo := []uint64{
			0,
		}
		for sz1 := range sizeInfo {
			if err := w.ReadSize(&sizeInfo[sz1]); err != nil {
				return err
			}
		}
		for sz1 := range sizeInfo {
			if err := w.ReadSize(&sizeInfo[sz1]); err != nil {
				return err
			}
			if err := w.ReadSize(&sizeInfo[sz1]); err != nil {
				return err
			}
		}
		var _Buffer_buf []uint16
		if sizeInfo[0] > uint64(w.Len()) /* sanity-check */ {
			return fmt.Errorf("buffer overflow for size %d of array _Buffer_buf", sizeInfo[0])
		}
		_Buffer_buf = make([]uint16, sizeInfo[0])
		for i1 := range _Buffer_buf {
			i1 := i1
			if err := w.ReadData(&_Buffer_buf[i1]); err != nil {
				return err
			}
		}
		o.Buffer = strings.TrimRight(string(utf16.Decode(_Buffer_buf)), ndr.ZeroString)
		return nil
	})
	_s_Buffer := func(ptr interface{}) { o.Buffer = *ptr.(*string) }
	if err := w.ReadPointer(&o.Buffer, _s_Buffer, _ptr_Buffer); err != nil {
		return err
	}
	return nil
}

// PcontextHandleMy structure represents PCONTEXT_HANDLE_MY RPC structure.
type PcontextHandleMy dcetypes.ContextHandle

func (o *PcontextHandleMy) ContextHandle() *dcetypes.ContextHandle {
	return (*dcetypes.ContextHandle)(o)
}

func (o *PcontextHandleMy) xxx_PreparePayload(ctx context.Context) error {
	if err := ndr.BeforePreparePayload(ctx, o); err != nil {
		return err
	}
	if err := ndr.AfterPreparePayload(ctx, o); err != nil {
		return err
	}
	return nil
}
func (o *PcontextHandleMy) MarshalNDR(ctx context.Context, w ndr.Writer) error {
	if err := o.xxx_PreparePayload(ctx); err != nil {
		return err
	}
	if err := w.WriteAlign(4); err != nil {
		return err
	}
	if err := w.WriteData(o.Attributes); err != nil {
		return err
	}
	if o.UUID != nil {
		if err := o.UUID.MarshalNDR(ctx, w); err != nil {
			return err
		}
	} else {
		if err := (&dtyp.GUID{}).MarshalNDR(ctx, w); err != nil {
			return err
		}
	}
	return nil
}
func (o *PcontextHandleMy) UnmarshalNDR(ctx context.Context, w ndr.Reader) error {
	if err := w.ReadAlign(4); err != nil {
		return err
	}
	if err := w.ReadData(&o.Attributes); err != nil {
		return err
	}
	if o.UUID == nil {
		o.UUID = &dtyp.GUID{}
	}
	if err := o.UUID.UnmarshalNDR(ctx, w); err != nil {
		return err
	}
	return nil
}

// MyInterfaceStruct structure represents MY_INTERFACE_STRUCT RPC structure.
type MyInterfaceStruct struct {
	Str1 string              `idl:"name:str1" json:"str1"`
	Str2 *dtyp.UnicodeString `idl:"name:str2" json:"str2"`
}

func (o *MyInterfaceStruct) xxx_PreparePayload(ctx context.Context) error {
	if err := ndr.BeforePreparePayload(ctx, o); err != nil {
		return err
	}
	if err := ndr.AfterPreparePayload(ctx, o); err != nil {
		return err
	}
	return nil
}
func (o *MyInterfaceStruct) MarshalNDR(ctx context.Context, w ndr.Writer) error {
	if err := o.xxx_PreparePayload(ctx); err != nil {
		return err
	}
	if err := w.WriteAlign(6); err != nil {
		return err
	}
	if o.Str1 != "" {
		_ptr_str1 := ndr.MarshalNDRFunc(func(ctx context.Context, w ndr.Writer) error {
			if err := ndr.WriteUTF16String(ctx, w, o.Str1); err != nil {
				return err
			}
			return nil
		})
		if err := w.WritePointer(&o.Str1, _ptr_str1); err != nil {
			return err
		}
	} else {
		if err := w.WritePointer(nil); err != nil {
			return err
		}
	}
	if o.Str2 != nil {
		_ptr_str2 := ndr.MarshalNDRFunc(func(ctx context.Context, w ndr.Writer) error {
			if o.Str2 != nil {
				if err := o.Str2.MarshalNDR(ctx, w); err != nil {
					return err
				}
			} else {
				if err := (&dtyp.UnicodeString{}).MarshalNDR(ctx, w); err != nil {
					return err
				}
			}
			return nil
		})
		if err := w.WritePointer(&o.Str2, _ptr_str2); err != nil {
			return err
		}
	} else {
		if err := w.WritePointer(nil); err != nil {
			return err
		}
	}
	return nil
}
func (o *MyInterfaceStruct) UnmarshalNDR(ctx context.Context, w ndr.Reader) error {
	if err := w.ReadAlign(6); err != nil {
		return err
	}
	_ptr_str1 := ndr.UnmarshalNDRFunc(func(ctx context.Context, w ndr.Reader) error {
		if err := ndr.ReadUTF16String(ctx, w, &o.Str1); err != nil {
			return err
		}
		return nil
	})
	_s_str1 := func(ptr interface{}) { o.Str1 = *ptr.(*string) }
	if err := w.ReadPointer(&o.Str1, _s_str1, _ptr_str1); err != nil {
		return err
	}
	_ptr_str2 := ndr.UnmarshalNDRFunc(func(ctx context.Context, w ndr.Reader) error {
		if o.Str2 == nil {
			o.Str2 = &dtyp.UnicodeString{}
		}
		if err := o.Str2.UnmarshalNDR(ctx, w); err != nil {
			return err
		}
		return nil
	})
	_s_str2 := func(ptr interface{}) { o.Str2 = *ptr.(**dtyp.UnicodeString) }
	if err := w.ReadPointer(&o.Str2, _s_str2, _ptr_str2); err != nil {
		return err
	}
	return nil
}

// IDcomInterface structure represents IDcomInterface RPC structure.
type IDcomInterface dcom.InterfacePointer

func (o *IDcomInterface) InterfacePointer() *dcom.InterfacePointer {
	return (*dcom.InterfacePointer)(o)
}

func (o *IDcomInterface) xxx_PreparePayload(ctx context.Context) error {
	if err := ndr.BeforePreparePayload(ctx, o); err != nil {
		return err
	}
	if o.Data != nil && o.DataCount == 0 {
		o.DataCount = uint32(len(o.Data))
	}
	if err := ndr.AfterPreparePayload(ctx, o); err != nil {
		return err
	}
	return nil
}

func (o *IDcomInterface) NDRSizeInfo() []uint64 {
	dimSize1 := uint64(o.DataCount)
	return []uint64{
		dimSize1,
	}
}
func (o *IDcomInterface) MarshalNDR(ctx context.Context, w ndr.Writer) error {
	if err := o.xxx_PreparePayload(ctx); err != nil {
		return err
	}
	sizeInfo, ok := ctx.Value(ndr.SizeInfo).([]uint64)
	if !ok {
		sizeInfo = o.NDRSizeInfo()
		for sz1 := range sizeInfo {
			if err := w.WriteSize(sizeInfo[sz1]); err != nil {
				return err
			}
		}
		ctx = context.WithValue(ctx, ndr.SizeInfo, sizeInfo)
	}
	if err := w.WriteAlign(4); err != nil {
		return err
	}
	if err := w.WriteData(o.DataCount); err != nil {
		return err
	}
	for i1 := range o.Data {
		i1 := i1
		if uint64(i1) >= sizeInfo[0] {
			break
		}
		if err := w.WriteData(o.Data[i1]); err != nil {
			return err
		}
	}
	for i1 := len(o.Data); uint64(i1) < sizeInfo[0]; i1++ {
		if err := w.WriteData(uint8(0)); err != nil {
			return err
		}
	}
	return nil
}
func (o *IDcomInterface) UnmarshalNDR(ctx context.Context, w ndr.Reader) error {
	sizeInfo, ok := ctx.Value(ndr.SizeInfo).([]uint64)
	if !ok {
		sizeInfo = o.NDRSizeInfo()
		for i1 := range sizeInfo {
			if err := w.ReadSize(&sizeInfo[i1]); err != nil {
				return err
			}
		}
		ctx = context.WithValue(ctx, ndr.SizeInfo, sizeInfo)
	}
	if err := w.ReadAlign(4); err != nil {
		return err
	}
	if err := w.ReadData(&o.DataCount); err != nil {
		return err
	}
	// XXX: for opaque unmarshaling
	if o.DataCount > 0 && sizeInfo[0] == 0 {
		sizeInfo[0] = uint64(o.DataCount)
	}
	if sizeInfo[0] > uint64(w.Len()) /* sanity-check */ {
		return fmt.Errorf("buffer overflow for size %d of array o.Data", sizeInfo[0])
	}
	o.Data = make([]byte, sizeInfo[0])
	for i1 := range o.Data {
		i1 := i1
		if err := w.ReadData(&o.Data[i1]); err != nil {
			return err
		}
	}
	return nil
}
