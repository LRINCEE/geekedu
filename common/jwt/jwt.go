package jwt

import (
	"errors"
	"time"

	"geekedu/common/config"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID int64 `json:"user_id"`
	Role   int32 `json:"role"`
	jwtv5.RegisteredClaims
}

func GenerateToken(userID int64, role int32) (string, error) {
	cfg := config.GetConfig()
	expireHours := cfg.JWT.ExpireHours
	if expireHours <= 0 {
		expireHours = 24
	}

	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwtv5.RegisteredClaims{
			ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
			IssuedAt:  jwtv5.NewNumericDate(time.Now()),
			Issuer:    "geekedu",
		},
	}

	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWT.Secret))
}

func ParseToken(tokenStr string) (*Claims, error) {
	cfg := config.GetConfig()
	token, err := jwtv5.ParseWithClaims(tokenStr, &Claims{}, func(token *jwtv5.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(cfg.JWT.Secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
