package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"

	api "github.com/grapery/common-protoc/gen"
)

func HashPassword(password string) string {
	bytes, _ := bcrypt.GenerateFromPassword([]byte(password), 5)

	return string(bytes)
}

func CheckPasswordHash(password string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	return err == nil
}

type JwtWrapper struct {
	SecretKey       string
	Issuer          string
	ExpirationHours int64
}

func NewJwtWrapper(key string, expiration int) *JwtWrapper {
	return &JwtWrapper{
		SecretKey:       key,
		ExpirationHours: int64(expiration),
	}
}

type jwtClaims struct {
	jwt.StandardClaims
	UID   int64
	GID   int64
	Email string
}

func (w *JwtWrapper) GenerateToken(user *api.UserInfo) (signedToken string, err error) {
	if user.Email == "" || user.UserId == 0 {
		return "", fmt.Errorf("invalid user info: %v", user.Name)
	}
	claims := &jwtClaims{
		UID:   int64(user.UserId),
		Email: user.Email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Hour * time.Duration(w.ExpirationHours)).Unix(),
			Issuer:    w.Issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err = token.SignedString([]byte(w.SecretKey))
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

func (w *JwtWrapper) ValidateToken(signedToken string) (claims *jwtClaims, err error) {
	token, err := jwt.ParseWithClaims(
		signedToken,
		&jwtClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(w.SecretKey), nil
		},
	)
	if err != nil {
		return
	}
	claims, ok := token.Claims.(*jwtClaims)
	if !ok {
		return nil, errors.New("Couldn't parse claims")
	}
	if claims.ExpiresAt < time.Now().Local().Unix() {
		return nil, errors.New("JWT is expired")
	}

	return claims, nil

}
