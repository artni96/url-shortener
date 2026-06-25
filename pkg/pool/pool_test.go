package pool

//func TestNew(t *testing.T) {
//	tests := []struct {
//		name        string
//		factory     func() testBuffer
//		expectError bool
//	}{
//		{
//			name: "valid factory function",
//			factory: func() testBuffer {
//				return testBuffer{data: make([]byte, 0, 1024)}
//			},
//			expectError: false,
//		},
//		{
//			name:        "nil factory function",
//			factory:     nil,
//			expectError: true,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			pool, err := New(tt.factory)
//
//			if tt.expectError {
//				if err == nil {
//					t.Error("Expected error but got nil")
//				}
//				if !errors.Is(err, ErrFuncNil) {
//					t.Errorf("Expected ErrFuncNil, got %v", err)
//				}
//				if pool != nil {
//					t.Error("Expected nil pool when error occurs, got non-nil")
//				}
//			} else {
//				if err != nil {
//					t.Errorf("Unexpected error: %v", err)
//				}
//				if pool == nil {
//					t.Error("Expected non-nil pool, got nil")
//				}
//				if pool.pool == nil {
//					t.Error("Expected non-nil sync.Pool, got nil")
//				}
//				if pool.pool.New == nil {
//					t.Error("Expected New function to be set, got nil")
//				}
//			}
//		})
//	}
//}
