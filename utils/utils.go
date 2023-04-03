package utils

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/mail"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/utils/cache"
)

var (
	CookieName   = "grapery"
	Domain       = ""
	CookieMaxAge = 86400
	CookiePath   = ""
)

// HasPrefixes returns true if the string s has any of the given prefixes.
func HasPrefixes(src string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(src, prefix) {
			return true
		}
	}
	return false
}

// ValidateEmail validates the email.
func ValidateEmail(email string) bool {
	if _, err := mail.ParseAddress(email); err != nil {
		return false
	}
	return true
}

func GenUUID() string {
	return uuid.New().String()
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// RandomString returns a random string with length n.
func RandomString(n int) (string, error) {
	var sb strings.Builder
	sb.Grow(n)
	for i := 0; i < n; i++ {
		// The reason for using crypto/rand instead of math/rand is that
		// the former relies on hardware to generate random numbers and
		// thus has a stronger source of random numbers.
		randNum, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		if _, err := sb.WriteRune(letters[randNum.Uint64()]); err != nil {
			return "", err
		}
	}
	return sb.String(), nil
}

type EmbedType string

const (
	EmbedTypeRich    EmbedType = "rich"
	EmbedTypeImage   EmbedType = "image"
	EmbedTypeVideo   EmbedType = "video"
	EmbedTypeGifv    EmbedType = "gifv"
	EmbedTypeArticle EmbedType = "article"
	EmbedTypeLink    EmbedType = "link"
)

type Context struct {
	GinC   *gin.Context
	Ctx    context.Context
	UserID uint64
	Err    error
	Resp   interface{}
}

type HandlerFunc func(c *Context)

func WrapHandler(h HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ret = new(Result)
		Ctx, _ := context.WithCancel(c.Request.Context())
		cookie, err := c.Cookie(CookieName)
		if err != nil {
			ret.Code = http.StatusOK
			ret.Message = "need login"
			ret.Data = nil
			c.JSON(http.StatusOK, ret)
			return
		}
		log.Info("cookie : ", cookie)
		cookieKey, err := base64.StdEncoding.DecodeString(cookie)
		if err != nil {
			ret.Code = http.StatusOK
			ret.Message = "need login"
			ret.Data = nil
			c.JSON(http.StatusOK, ret)
			return
		}
		// parse cookie
		infoData, err := cache.GetBytes(Ctx, string(cookieKey))
		if err != nil {
			ret.Code = http.StatusOK
			ret.Message = "need login"
			ret.Data = nil
			c.JSON(http.StatusOK, ret)
			return
		}
		log.Info(string(infoData))
		ctx := &Context{
			GinC:   c,
			Ctx:    Ctx,
			UserID: 0,
		}
		var info = new(api.UserInfo)
		err = json.Unmarshal(infoData, info)
		if err != nil {
			ctx.Err = err
			ctx.Resp = nil
		} else {
			ctx.UserID = info.GetUserId()
			h(ctx)
		}
		// err handle
		if ctx.Err != nil {
			ret.Code = http.StatusOK
			ret.Message = ctx.Err.Error()
			ret.Data = nil
			c.JSON(http.StatusOK, ret)
			return
		}
		ret.Message = "ok"
		ret.Code = http.StatusOK
		ret.Data = ctx.Resp
		c.JSON(http.StatusOK, ret)
	}
}
