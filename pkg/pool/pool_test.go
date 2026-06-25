package pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testStruct struct {
	field string
}

func (t *testStruct) Reset() {
	t.field = ""
}

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		factory   func() *testStruct
		expectErr bool
	}{
		{
			name: "success",
			factory: func() *testStruct {
				return &testStruct{field: "test"}
			},
			expectErr: false,
		},
		{
			name:      "failure - nil factory function returns error",
			factory:   nil,
			expectErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := New(tt.factory)
			if tt.expectErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrFuncNil)

			} else {
				assert.NoError(t, err)
				pool.Put(tt.factory())
				gottenObj := pool.Get()
				expectedObj := testStruct{field: ""}
				assert.Equal(t, *gottenObj, expectedObj)
			}
		})
	}
}
