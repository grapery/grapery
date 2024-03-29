// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: github.com/grapery/grapery/common-protoc/disscuss.proto

package api

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang/protobuf/ptypes"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = ptypes.DynamicAny{}
)

// Validate checks the field values on CreateDisscussReq with the rules defined
// in the proto definition for this message. If any rules are violated, an
// error is returned.
func (m *CreateDisscussReq) Validate() error {
	if m == nil {
		return nil
	}

	return nil
}

// CreateDisscussReqValidationError is the validation error returned by
// CreateDisscussReq.Validate if the designated constraints aren't met.
type CreateDisscussReqValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e CreateDisscussReqValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e CreateDisscussReqValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e CreateDisscussReqValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e CreateDisscussReqValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e CreateDisscussReqValidationError) ErrorName() string {
	return "CreateDisscussReqValidationError"
}

// Error satisfies the builtin error interface
func (e CreateDisscussReqValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sCreateDisscussReq.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = CreateDisscussReqValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = CreateDisscussReqValidationError{}

// Validate checks the field values on CreateDisscusResp with the rules defined
// in the proto definition for this message. If any rules are violated, an
// error is returned.
func (m *CreateDisscusResp) Validate() error {
	if m == nil {
		return nil
	}

	return nil
}

// CreateDisscusRespValidationError is the validation error returned by
// CreateDisscusResp.Validate if the designated constraints aren't met.
type CreateDisscusRespValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e CreateDisscusRespValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e CreateDisscusRespValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e CreateDisscusRespValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e CreateDisscusRespValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e CreateDisscusRespValidationError) ErrorName() string {
	return "CreateDisscusRespValidationError"
}

// Error satisfies the builtin error interface
func (e CreateDisscusRespValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sCreateDisscusResp.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = CreateDisscusRespValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = CreateDisscusRespValidationError{}

// Validate checks the field values on GetDisscusReq with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned.
func (m *GetDisscusReq) Validate() error {
	if m == nil {
		return nil
	}

	return nil
}

// GetDisscusReqValidationError is the validation error returned by
// GetDisscusReq.Validate if the designated constraints aren't met.
type GetDisscusReqValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e GetDisscusReqValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e GetDisscusReqValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e GetDisscusReqValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e GetDisscusReqValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e GetDisscusReqValidationError) ErrorName() string { return "GetDisscusReqValidationError" }

// Error satisfies the builtin error interface
func (e GetDisscusReqValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sGetDisscusReq.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = GetDisscusReqValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = GetDisscusReqValidationError{}

// Validate checks the field values on GetDisscusResp with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned.
func (m *GetDisscusResp) Validate() error {
	if m == nil {
		return nil
	}

	return nil
}

// GetDisscusRespValidationError is the validation error returned by
// GetDisscusResp.Validate if the designated constraints aren't met.
type GetDisscusRespValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e GetDisscusRespValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e GetDisscusRespValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e GetDisscusRespValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e GetDisscusRespValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e GetDisscusRespValidationError) ErrorName() string { return "GetDisscusRespValidationError" }

// Error satisfies the builtin error interface
func (e GetDisscusRespValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sGetDisscusResp.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = GetDisscusRespValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = GetDisscusRespValidationError{}
