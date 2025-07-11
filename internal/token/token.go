package token

import (
	"core-regulus-backend/internal/config"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserTokenData struct {
	Name  string `json:"name"`
	Id    string `json:"id"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func GenerateJWT(data UserTokenData) (string, error) {		
	cfg := config.Get()
	data.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    "core-regulus",
		Subject:   "user-token",
		ExpiresAt: nil,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}	
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, data)
	return token.SignedString(cfg.JWT.PrivateKey)
}

func ValidateJWT(tokenString string) (*UserTokenData, error) {	
	cfg := config.Get()
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return cfg.JWT.PublicKey, nil
	})
	if err != nil {
		return nil, err
	}
	
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &UserTokenData{
			Id: fmt.Sprintf("%v", claims["id"]),
			Email:   fmt.Sprintf("%v", claims["email"]),
			Name:   fmt.Sprintf("%v", claims["name"]),
		}, nil
	} else {
		return nil, fmt.Errorf("invalid token")
	}
}