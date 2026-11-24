package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wuonhai/webook/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

func TestEncrypt(t *testing.T) {
	password := "hello#world123"
	encrypt, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	err = bcrypt.CompareHashAndPassword(encrypt, []byte(password))

	assert.NoError(t, err)
}

func ptr[T any](v T) *T {
	return &v
}

func TestValidateEditReq(t *testing.T) {
	now := time.Date(2026, 11, 24, 15, 0, 0, 0, time.FixedZone("CST", 8*3600))

	testCases := []struct {
		name      string
		req       editReq
		wantErrs  map[string]string
		checkFunc func(t *testing.T, update domain.UserProfileUpdate)
	}{
		{
			name: "Chinese rune limits accepted (32 nickname, 500 bio)",
			req: editReq{
				Nickname: ptr("  " + strings.Repeat("好", 32) + "  "),
				Bio:      ptr(" " + strings.Repeat("文", 500) + " "),
			},
			wantErrs: map[string]string{},
			checkFunc: func(t *testing.T, update domain.UserProfileUpdate) {
				require.NotNil(t, update.Nickname)
				assert.Equal(t, strings.Repeat("好", 32), *update.Nickname)
				require.NotNil(t, update.Bio)
				assert.Equal(t, strings.Repeat("文", 500), *update.Bio)
			},
		},
		{
			name: "Chinese rune limits exceeded (33 nickname, 501 bio)",
			req: editReq{
				Nickname: ptr(strings.Repeat("好", 33)),
				Bio:      ptr(strings.Repeat("文", 501)),
			},
			wantErrs: map[string]string{
				"nickname": "昵称长度不能超过 32 个字符",
				"bio":      "个人简介长度不能超过 500 个字符",
			},
		},
		{
			name: "whitespace nickname rejected and whitespace bio clears",
			req: editReq{
				Nickname: ptr("   \t\n "),
				Bio:      ptr("   "),
			},
			wantErrs: map[string]string{
				"nickname": "昵称不能为空",
			},
			checkFunc: func(t *testing.T, update domain.UserProfileUpdate) {
				require.NotNil(t, update.Bio)
				assert.Equal(t, "", *update.Bio)
			},
		},
		{
			name: "empty bio clears",
			req: editReq{
				Bio: ptr(""),
			},
			wantErrs: map[string]string{},
			checkFunc: func(t *testing.T, update domain.UserProfileUpdate) {
				require.NotNil(t, update.Bio)
				assert.Equal(t, "", *update.Bio)
			},
		},
		{
			name: "valid birthdays including 1970-01-01 and pre-1970",
			req: editReq{
				Birthday: ptr("1970-01-01"),
			},
			wantErrs: map[string]string{},
			checkFunc: func(t *testing.T, update domain.UserProfileUpdate) {
				require.False(t, update.ClearBirthday)
				require.NotNil(t, update.Birthday)
				assert.Equal(t, time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), *update.Birthday)
				assert.Equal(t, int64(0), update.Birthday.UnixMilli())
			},
		},
		{
			name: "valid pre-1970 and 1992 birthdays",
			req: editReq{
				Birthday: ptr("1965-05-20"),
			},
			wantErrs: map[string]string{},
			checkFunc: func(t *testing.T, update domain.UserProfileUpdate) {
				require.NotNil(t, update.Birthday)
				assert.Equal(t, time.Date(1965, 5, 20, 0, 0, 0, 0, time.UTC), *update.Birthday)
			},
		},
		{
			name: "valid birthday 1992-01-01",
			req: editReq{
				Birthday: ptr("1992-01-01"),
			},
			wantErrs: map[string]string{},
			checkFunc: func(t *testing.T, update domain.UserProfileUpdate) {
				require.NotNil(t, update.Birthday)
				assert.Equal(t, time.Date(1992, 1, 1, 0, 0, 0, 0, time.UTC), *update.Birthday)
			},
		},
		{
			name: "clear birthday with empty string",
			req: editReq{
				Birthday: ptr(""),
			},
			wantErrs: map[string]string{},
			checkFunc: func(t *testing.T, update domain.UserProfileUpdate) {
				assert.True(t, update.ClearBirthday)
				assert.Nil(t, update.Birthday)
			},
		},
		{
			name: "invalid calendar date 2023-02-30",
			req: editReq{
				Birthday: ptr("2023-02-30"),
			},
			wantErrs: map[string]string{
				"birthday": "生日格式必须为 YYYY-MM-DD 且为有效日期",
			},
		},
		{
			name: "birthday before 1900-01-01",
			req: editReq{
				Birthday: ptr("1899-12-31"),
			},
			wantErrs: map[string]string{
				"birthday": "生日不能早于 1900-01-01",
			},
		},
		{
			name: "birthday in the future",
			req: editReq{
				Birthday: ptr("2026-11-25"),
			},
			wantErrs: map[string]string{
				"birthday": "生日不能晚于今天",
			},
		},
		{
			name: "aggregated multi-field validation errors",
			req: editReq{
				Nickname: ptr(""),
				Birthday: ptr("bad-date"),
				Bio:      ptr(strings.Repeat("a", 501)),
			},
			wantErrs: map[string]string{
				"nickname": "昵称不能为空",
				"birthday": "生日格式必须为 YYYY-MM-DD 且为有效日期",
				"bio":      "个人简介长度不能超过 500 个字符",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			update, errs := validateEditReq(42, tc.req, now)
			assert.Equal(t, int64(42), update.UserId)
			assert.Equal(t, tc.wantErrs, errs)
			if tc.checkFunc != nil {
				tc.checkFunc(t, update)
			}
		})
	}
}

func TestBuildProfileRespJSON(t *testing.T) {
	testCases := []struct {
		name     string
		user     domain.User
		expected map[string]any
	}{
		{
			name: "unset nullable fields serialize as null and omit password",
			user: domain.User{
				Id:       1,
				Email:    "user@example.com",
				Password: "hashed-secret-password",
				Ctime:    time.UnixMilli(1732270000000).UTC(),
			},
			expected: map[string]any{
				"id":       float64(1),
				"email":    "user@example.com",
				"nickname": nil,
				"birthday": nil,
				"bio":      nil,
				"ctime":    float64(1732270000000),
			},
		},
		{
			name: "1970-01-01 birthday formats properly",
			user: domain.User{
				Id:       2,
				Email:    "epoch@example.com",
				Password: "hashed-secret-password",
				Nickname: ptr("小明"),
				Birthday: ptr(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)),
				Bio:      ptr("Go developer"),
				Ctime:    time.UnixMilli(1732270000001).UTC(),
			},
			expected: map[string]any{
				"id":       float64(2),
				"email":    "epoch@example.com",
				"nickname": "小明",
				"birthday": "1970-01-01",
				"bio":      "Go developer",
				"ctime":    float64(1732270000001),
			},
		},
		{
			name: "pre-1970 birthday formats properly",
			user: domain.User{
				Id:       3,
				Email:    "pre@example.com",
				Password: "hashed-secret-password",
				Birthday: ptr(time.Date(1965, 5, 20, 0, 0, 0, 0, time.UTC)),
				Ctime:    time.UnixMilli(1732270000002).UTC(),
			},
			expected: map[string]any{
				"id":       float64(3),
				"email":    "pre@example.com",
				"nickname": nil,
				"birthday": "1965-05-20",
				"bio":      nil,
				"ctime":    float64(1732270000002),
			},
		},
		{
			name: "zero ctime serializes as 0",
			user: domain.User{
				Id:       4,
				Email:    "zero@example.com",
				Password: "hashed-secret-password",
				Ctime:    time.Time{},
			},
			expected: map[string]any{
				"id":       float64(4),
				"email":    "zero@example.com",
				"nickname": nil,
				"birthday": nil,
				"bio":      nil,
				"ctime":    float64(0),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := buildProfileResp(tc.user)
			raw, err := json.Marshal(resp)
			require.NoError(t, err)

			assert.NotContains(t, string(raw), "password")
			assert.NotContains(t, string(raw), "hashed-secret-password")

			var got map[string]any
			require.NoError(t, json.Unmarshal(raw, &got))
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestUserHandlerEditHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := gin.New()
	store := cookie.NewStore([]byte("secret"))
	server.Use(sessions.Sessions("webookSSID", store))

	server.GET("/test-login", func(ctx *gin.Context) {
		sess := sessions.Default(ctx)
		sess.Set("userId", int64(7))
		require.NoError(t, sess.Save())
		ctx.Status(http.StatusOK)
	})

	handler := NewUserHandler(nil)
	handler.RegisterRoutes(server)

	loginReq := httptest.NewRequest(http.MethodGet, "/test-login", nil)
	loginRec := httptest.NewRecorder()
	server.ServeHTTP(loginRec, loginReq)
	require.Equal(t, http.StatusOK, loginRec.Code)
	cookies := loginRec.Result().Cookies()
	require.NotEmpty(t, cookies)

	doEdit := func(body string, withCookie bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/users/edit", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		if withCookie {
			for _, c := range cookies {
				req.AddCookie(c)
			}
		}
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)
		return rec
	}

	t.Run("missing session returns HTTP 200 with code 40100", func(t *testing.T) {
		rec := doEdit(`{"nickname":"小明"}`, false)
		assert.Equal(t, http.StatusOK, rec.Code)

		var got Result
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, codeUnauthorized, got.Code)
	})

	t.Run("empty object returns HTTP 200 with code 40001", func(t *testing.T) {
		rec := doEdit(`{}`, true)
		assert.Equal(t, http.StatusOK, rec.Code)

		var got Result
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, codeBadRequest, got.Code)
		assert.Equal(t, "至少需要提供一个要修改的字段", got.Msg)
	})

	t.Run("invalid JSON returns HTTP 200 with code 40001", func(t *testing.T) {
		rec := doEdit(`{invalid`, true)
		assert.Equal(t, http.StatusOK, rec.Code)

		var got Result
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, codeBadRequest, got.Code)
		assert.Equal(t, "请求参数格式错误", got.Msg)
	})

	t.Run("multi-field validation error returns HTTP 200 with code 40001 and field map", func(t *testing.T) {
		rec := doEdit(`{"nickname":"","birthday":"2023-02-30"}`, true)
		assert.Equal(t, http.StatusOK, rec.Code)

		var got Result
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, codeBadRequest, got.Code)
		assert.Equal(t, "参数校验失败", got.Msg)
		assert.Equal(t, map[string]any{
			"nickname": "昵称不能为空",
			"birthday": "生日格式必须为 YYYY-MM-DD 且为有效日期",
		}, got.Data)
	})
}
