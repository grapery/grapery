package errors

import (
	"fmt"
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
	ErrAuthNotFound        = NewSysError(1001, "auth info is not exist")
	ErrAuthExpired         = NewSysError(1002, "auth info is expired")
	ErrAuthIsExist         = NewSysError(1003, "auth info is exist")
	ErrAuthPasswordIsWrong = NewSysError(1003, "auth password is wrong")
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

var (
	ErrStoryIsNotExist = NewSysError(6001, "Story is not exist")
	ErrStoryIsClosed   = NewSysError(6002, "Story is closed")
	ErrStoryIsInvalid  = NewSysError(6003, "Story is invalid")
	ErrStoryIsExpired  = NewSysError(6004, "Story is expired")
)
