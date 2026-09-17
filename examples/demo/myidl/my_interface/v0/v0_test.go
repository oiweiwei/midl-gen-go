package my_interface

import (
	"encoding/hex"
	"testing"

	"github.com/oiweiwei/go-msrpc/ndr"
	"github.com/stretchr/testify/assert"
)

func TestNullMaskMarshal(t *testing.T) {

	for _, testCase := range []struct {
		name     string
		mrs      ndr.Marshaler
		expected string
	}{
		{
			name:     "default_not_null",
			mrs:      &TestCall2Request{},
			expected: "00000000" + "05000000" /* ptr */ + "00000000",
		},
		{
			name:     "set_null",
			mrs:      &TestCall2Request{NullMask: TestCall2NullMaskDwordPointer},
			expected: "00000000" + "00000000", /* ptr */
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// marshal data.
			b, err := ndr.Marshal(testCase.mrs)
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, hex.EncodeToString(b))

			// unmarshal data.
			um := &TestCall2Request{}
			assert.NoError(t, ndr.Unmarshal(b, um))
			assert.Equal(t, testCase.mrs, um)
		})
	}

}
