package dao

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T {
	return &v
}

func TestBuildProfileUpdates(t *testing.T) {
	const fixedNow int64 = 1732447200000

	testCases := []struct {
		name     string
		update   UserProfileUpdate
		expected map[string]any
	}{
		{
			name:   "bio nil omits bio key",
			update: UserProfileUpdate{Id: 1},
			expected: map[string]any{
				"utime": fixedNow,
			},
		},
		{
			name:   "bio empty string sets untyped nil",
			update: UserProfileUpdate{Id: 1, Bio: ptr("")},
			expected: map[string]any{
				"utime": fixedNow,
				"bio":   nil,
			},
		},
		{
			name:   "bio non-empty sets bio value",
			update: UserProfileUpdate{Id: 1, Bio: ptr("hello")},
			expected: map[string]any{
				"utime": fixedNow,
				"bio":   "hello",
			},
		},
		{
			name:   "clear birthday sets untyped nil",
			update: UserProfileUpdate{Id: 1, ClearBirthday: true, Birthday: ptr(int64(123))},
			expected: map[string]any{
				"utime":    fixedNow,
				"birthday": nil,
			},
		},
		{
			name:   "birthday 1970-01-01 sets zero timestamp",
			update: UserProfileUpdate{Id: 1, Birthday: ptr(int64(0))},
			expected: map[string]any{
				"utime":    fixedNow,
				"birthday": int64(0),
			},
		},
		{
			name:   "birthday pre-1970 sets negative timestamp",
			update: UserProfileUpdate{Id: 1, Birthday: ptr(int64(-145756800000))},
			expected: map[string]any{
				"utime":    fixedNow,
				"birthday": int64(-145756800000),
			},
		},
		{
			name: "nickname set updates nickname",
			update: UserProfileUpdate{
				Id:       1,
				Nickname: ptr("小明"),
			},
			expected: map[string]any{
				"utime":    fixedNow,
				"nickname": "小明",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildProfileUpdates(tc.update, fixedNow)
			assert.Equal(t, tc.expected, got)

			if v, ok := got["bio"]; ok && tc.update.Bio != nil && *tc.update.Bio == "" {
				require.True(t, v == nil, "bio clear value must be untyped nil")
			}
			if v, ok := got["birthday"]; ok && tc.update.ClearBirthday {
				require.True(t, v == nil, "birthday clear value must be untyped nil")
			}
		})
	}
}
