package service

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var jwtKey = []byte("magic_girl_secret_key")

type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.StandardClaims
}

// GenerateToken 生成 JWT Token (首字母必须大写!)
func GenerateToken(userID int64) (string, error) {
	expireTime := time.Now().Add(7 * 24 * time.Hour)
	claims := &Claims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
			Issuer:    "vedio-server",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}
