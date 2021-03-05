package errors

import "errors"

var (
	ErrUserIsExist           = errors.New("user is exist")
	ErrCreateAuthFailed      = errors.New("create auth info failed")
	ErrResetPasswordFailed   = errors.New("reset password failed")
	ErrGetUserAuthInfoFailed = errors.New("get user auth info failed")
	ErrDeleteUserAuthInfo    = errors.New("delete uaer auth info failed")
)
