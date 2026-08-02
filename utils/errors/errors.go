package errors

import (
	"fmt"

	api "github.com/grapery/common-protoc/gen"
)

type SysError struct {
	Code        int
	Description string
}

func NewSysError(code int, desc string) *SysError {
	return &SysError{
		Code:        code,
		Description: desc,
	}
}

func (e SysError) Error() string {
	return fmt.Sprintf("error: code %d description %s", e.Code, e.Description)
}

var (
	ErrAuthNotFound        = NewSysError(int(api.ResponseCode_ACCOUNT_NOT_FOUND), "auth info is not exist")
	ErrAuthExpired         = NewSysError(int(api.ResponseCode_ACCOUNT_EXPIRED), "auth info is expired")
	ErrAuthIsExist         = NewSysError(int(api.ResponseCode_USER_ALREADY_EXISTS), "auth info is exist")
	ErrAuthPasswordIsWrong = NewSysError(int(api.ResponseCode_WRONG_PASSWORD), "auth password is wrong")
)

var (
	ErrUserIsExist           = NewSysError(2001, "user is not exist")
	ErrCreateAuthFailed      = NewSysError(2002, "create auth info failed")
	ErrResetPasswordFailed   = NewSysError(2003, "reset password failed")
	ErrGetUserAuthInfoFailed = NewSysError(2004, "get user auth info failed")
	ErrDeleteUserAuthInfo    = NewSysError(2005, "delete uaer auth info failed")
)

var (
	ErrGroupIsNotExist     = NewSysError(3001, "group is not exist")
	ErrGroupIsAlreadyExist = NewSysError(3001, "group is already exist")
)

var (
	ErrProjectIsNotExist = NewSysError(4001, "project is not exist")
	ErrProjectIsClosed   = NewSysError(4002, "project is closed")
	ErrProjectIsInvalid  = NewSysError(4003, "project is invalid")
	ErrProjectIsExpired  = NewSysError(4004, "project is expired")
	ErrProjectIsPrivate  = NewSysError(4005, "project is private")
)

var (
	ErrItemIsNotExist = NewSysError(4001, "Item is not exist")
)

var (
	ErrLikeItemIsNotExist = NewSysError(5001, "LikeItem is not exist")
	ErrLikeItemIsExist    = NewSysError(5002, "LikeItem is exist")
)

// AuthResponseCodeMap 认证相关响应码映射表
// 客户端可以根据这些响应码显示对应的提示信息
var AuthResponseCodeMap = map[api.ResponseCode]string{
	api.ResponseCode_OK:                  "操作成功",
	api.ResponseCode_ACCOUNT_NOT_FOUND:   "账号不存在",
	api.ResponseCode_WRONG_PASSWORD:      "密码错误",
	api.ResponseCode_USER_ALREADY_EXISTS: "用户已存在",
	api.ResponseCode_ACCOUNT_EXPIRED:     "账号已过期",
	api.ResponseCode_OPERATION_FAILED:    "操作失败",
	api.ResponseCode_INVALID_PARAMETER:   "参数错误",
	api.ResponseCode_DATABASE_ERROR:      "数据库错误",
	api.ResponseCode_TOKEN_EXPIRED:       "令牌已过期",
	api.ResponseCode_TOKEN_INVALID:       "令牌无效",
	api.ResponseCode_PERMISSION_DENIED:   "权限不足",
	api.ResponseCode_RATE_LIMIT_EXCEEDED: "请求频率过高",
	api.ResponseCode_SERVICE_UNAVAILABLE: "服务不可用",
	api.ResponseCode_INTERNAL_ERROR:      "服务器内部错误",
}

// GetAuthResponseMessage 根据响应码获取对应的中文提示信息
func GetAuthResponseMessage(code api.ResponseCode) string {
	if message, exists := AuthResponseCodeMap[code]; exists {
		return message
	}
	return "未知错误"
}

// GetAuthResponseCode 根据错误类型获取对应的响应码
func GetAuthResponseCode(err error) api.ResponseCode {
	switch err {
	case ErrAuthNotFound:
		return api.ResponseCode_ACCOUNT_NOT_FOUND
	case ErrAuthPasswordIsWrong:
		return api.ResponseCode_WRONG_PASSWORD
	case ErrAuthIsExist:
		return api.ResponseCode_USER_ALREADY_EXISTS
	case ErrAuthExpired:
		return api.ResponseCode_ACCOUNT_EXPIRED
	default:
		return api.ResponseCode_OPERATION_FAILED
	}
}

var (
	ErrStoryIsNotExist = NewSysError(6001, "Story is not exist")
	ErrStoryIsClosed   = NewSysError(6002, "Story is closed")
	ErrStoryIsInvalid  = NewSysError(6003, "Story is invalid")
	ErrStoryIsExpired  = NewSysError(6004, "Story is expired")
)

var (
	ErrTokenIsEmpty          = NewSysError(int(api.ResponseCode_MISSING_PARAMETER), "token is empty")
	ErrFeatureNotImplemented = NewSysError(int(api.ResponseCode_OPERATION_NOT_SUPPORTED), "feature not implemented")
)

var (
	ErrUserDefaultGroupMismatch = NewSysError(int(api.ResponseCode_USER_STATUS_ERROR), "user default group info not match")
	ErrCreateDefaultGroupFailed = NewSysError(int(api.ResponseCode_GROUP_OPERATION_DENIED), "create default group failed")
	ErrInvalidUserID            = NewSysError(int(api.ResponseCode_INVALID_PARAMETER), "invalid user id")
	ErrInvalidActiveType        = NewSysError(int(api.ResponseCode_INVALID_PARAMETER), "invalid active type")
)

var (
	ErrMissingParameter = NewSysError(int(api.ResponseCode_MISSING_PARAMETER), "missing parameter")
	ErrInvalidParameter = NewSysError(int(api.ResponseCode_INVALID_PARAMETER), "invalid parameter")
)
