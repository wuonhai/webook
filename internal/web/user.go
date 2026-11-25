package web

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	regexp "github.com/dlclark/regexp2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/wuonhai/webook/internal/domain"
	"github.com/wuonhai/webook/internal/service"
)

// define the user related router
type UserHandler struct {
	svc         *service.UserService
	emailExp    *regexp.Regexp
	passwordExp *regexp.Regexp
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	const (
		emailRegexPattern    = `^\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*$`
		passwordRegexPattern = `^(?=.*[A-Za-z])(?=.*\d)(?=.*[$@$!%*#?&])[A-Za-z\d$@$!%*#?&]{8,72}$`
	)
	emailExp := regexp.MustCompile(emailRegexPattern, regexp.None)
	passwordExp := regexp.MustCompile(passwordRegexPattern, regexp.None)
	return &UserHandler{
		svc:         svc,
		emailExp:    emailExp,
		passwordExp: passwordExp,
	}
}

func (u *UserHandler) RegisterRoutes(server *gin.Engine) {
	ug := server.Group("/users")
	ug.GET("/profile", u.Profile)
	ug.POST("/signup", u.Signup)
	ug.POST("/login", u.Login)
	ug.POST("/edit", u.Edit)
}

func (u *UserHandler) Signup(ctx *gin.Context) {
	type SignUpReq struct {
		Email           string `json:"email"`
		ConfirmPassword string `json:"confirmPassword"`
		Password        string `json:"password"`
	}

	var req SignUpReq
	// Bind function will parse the Content-Type in the request body and fill the signupReq struct with the corresponding data.
	// If the Content-Type is not supported, it will return an error.

	if err := ctx.Bind(&req); err != nil {
		return
	}

	ok, err := u.emailExp.MatchString(req.Email)
	if err != nil {
		ctx.String(http.StatusOK, "system error")
		return
	}
	if !ok {
		ctx.String(http.StatusOK, "🚫email format is not correct")
		return
	}
	if req.ConfirmPassword != req.Password {
		ctx.String(http.StatusOK, "🚫password and confirm password are not the same")
		return
	}

	ok, err = u.passwordExp.MatchString(req.Password)
	if err != nil {
		// 记录日志
		ctx.String(http.StatusOK, "system error")
		return
	}
	if !ok {
		ctx.String(http.StatusOK, "🚫password must be more than 8 characters, and contain numbers and special characters")
		return
	}

	err = u.svc.Signup(ctx, domain.User{
		Email:    req.Email,
		Password: req.Password,
	})

	if errors.Is(err, service.ErrUserDuplicateEmail) {
		ctx.String(http.StatusOK, "🚫email conflict")
		return
	}
	if err != nil {
		ctx.String(http.StatusOK, "system error")
		return
	}
	ctx.String(http.StatusOK, "🎉signup success")

}

func (u *UserHandler) Login(ctx *gin.Context) {
	type LoginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var req LoginReq
	if err := ctx.Bind(&req); err != nil {
		return
	}

	userDo, err := u.svc.Login(ctx, domain.User{
		Email:    req.Email,
		Password: req.Password,
	})

	if err != nil {
		switch err {
		case service.ErrInvalidUserOrPassword:
			ctx.String(http.StatusOK, "🚫invalid user or password")
		case service.ErrUserNotFound:
			ctx.String(http.StatusOK, "🚫user not found")
		default:
			ctx.String(http.StatusOK, "system error")
		}
		return
	}

	sess := sessions.Default(ctx)
	sess.Set("userId", userDo.Id)
	sess.Save()

	ctx.String(http.StatusOK, "🎉login success")
}

const (
	codeSuccess      = 0
	codeBadRequest   = 40001
	codeUnauthorized = 40100
	codeNotFound     = 40400
	codeSystemError  = 50000

	maxNicknameRunes = 32
	maxBioRunes      = 500
)

// Result represents the unified JSON response envelope for profile endpoints.
type Result struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func getSessionUserId(ctx *gin.Context) (int64, bool) {
	val := sessions.Default(ctx).Get("userId")
	userId, ok := val.(int64)
	if !ok || userId <= 0 {
		return 0, false
	}
	return userId, true
}

func validateProfileFields(userId int64, nickname, birthday, bio *string, now time.Time) (domain.UserProfileUpdate, map[string]string) {
	update := domain.UserProfileUpdate{UserId: userId}
	fieldErrs := make(map[string]string)

	if nickname != nil {
		trimmed := strings.TrimSpace(*nickname)
		count := utf8.RuneCountInString(trimmed)
		if count == 0 {
			fieldErrs["nickname"] = "昵称不能为空"
		} else if count > maxNicknameRunes {
			fieldErrs["nickname"] = "昵称长度不能超过 32 个字符"
		} else {
			update.Nickname = &trimmed
		}
	}

	if birthday != nil {
		raw := strings.TrimSpace(*birthday)
		if raw == "" {
			update.ClearBirthday = true
		} else if t, err := time.Parse("2006-01-02", raw); err != nil {
			fieldErrs["birthday"] = "生日格式必须为 YYYY-MM-DD 且为有效日期"
		} else {
			parsed := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			minDate := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			if parsed.Before(minDate) {
				fieldErrs["birthday"] = "生日不能早于 1900-01-01"
			} else if parsed.After(today) {
				fieldErrs["birthday"] = "生日不能晚于今天"
			} else {
				update.Birthday = &parsed
			}
		}
	}

	if bio != nil {
		trimmed := strings.TrimSpace(*bio)
		if utf8.RuneCountInString(trimmed) > maxBioRunes {
			fieldErrs["bio"] = "个人简介长度不能超过 500 个字符"
		} else {
			update.Bio = &trimmed
		}
	}

	return update, fieldErrs
}

func formatBirthday(birthday *time.Time) *string {
	if birthday == nil {
		return nil
	}
	formatted := birthday.UTC().Format("2006-01-02")
	return &formatted
}

func formatCtime(ctime time.Time) int64 {
	if ctime.IsZero() {
		return 0
	}
	return ctime.UnixMilli()
}

func (u *UserHandler) Edit(ctx *gin.Context) {
	type EditReq struct {
		Nickname *string `json:"nickname"`
		Birthday *string `json:"birthday"`
		Bio      *string `json:"bio"`
	}

	userId, ok := getSessionUserId(ctx)
	if !ok {
		ctx.JSON(http.StatusOK, Result{Code: codeUnauthorized, Msg: "未登录或会话无效"})
		return
	}

	var req EditReq
	// Use ShouldBindJSON so malformed JSON still returns HTTP 200 with unified Result payload.
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, Result{Code: codeBadRequest, Msg: "请求参数格式错误"})
		return
	}

	if req.Nickname == nil && req.Birthday == nil && req.Bio == nil {
		ctx.JSON(http.StatusOK, Result{Code: codeBadRequest, Msg: "至少需要提供一个要修改的字段"})
		return
	}

	update, fieldErrs := validateProfileFields(userId, req.Nickname, req.Birthday, req.Bio, time.Now())
	if len(fieldErrs) > 0 {
		ctx.JSON(http.StatusOK, Result{Code: codeBadRequest, Msg: "参数校验失败", Data: fieldErrs})
		return
	}

	if err := u.svc.Edit(ctx, update); err != nil {
		ctx.JSON(http.StatusOK, Result{Code: codeSystemError, Msg: "系统错误"})
		return
	}

	ctx.JSON(http.StatusOK, Result{Code: codeSuccess, Msg: "success"})
}

func (u *UserHandler) Profile(ctx *gin.Context) {
	type ProfileResp struct {
		Id       int64   `json:"id"`
		Email    string  `json:"email"`
		Nickname *string `json:"nickname"`
		Birthday *string `json:"birthday"`
		Bio      *string `json:"bio"`
		Ctime    int64   `json:"ctime"`
	}

	userId, ok := getSessionUserId(ctx)
	if !ok {
		ctx.JSON(http.StatusOK, Result{Code: codeUnauthorized, Msg: "未登录或会话无效"})
		return
	}

	userDo, err := u.svc.Profile(ctx, userId)
	if errors.Is(err, service.ErrUserNotFound) {
		ctx.JSON(http.StatusOK, Result{Code: codeNotFound, Msg: "用户不存在"})
		return
	}
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Code: codeSystemError, Msg: "系统错误"})
		return
	}

	ctx.JSON(http.StatusOK, Result{
		Code: codeSuccess,
		Msg:  "success",
		Data: ProfileResp{
			Id:       userDo.Id,
			Email:    userDo.Email,
			Nickname: userDo.Nickname,
			Birthday: formatBirthday(userDo.Birthday),
			Bio:      userDo.Bio,
			Ctime:    formatCtime(userDo.Ctime),
		},
	})
}
