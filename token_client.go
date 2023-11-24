package ai

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenClaims JWT TokenClaims
type TokenClaims struct {
	Exp  int    `json:"exp"`
	Iat  int    `json:"iat"`
	UID  string `json:"uid"`
	Role string `json:"role"`
}

type TokenClient struct {
	SecretKey string // 密钥
	Exp       int    // 分钟
}

// NewTokenClient 初始化一个 TokenClient
func NewTokenClient(secretKey string, exp int) *TokenClient {
	return &TokenClient{SecretKey: secretKey, Exp: exp}
}

// GenerateToken 生成一个token
func (c *TokenClient) GenerateToken(uid, role string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid":  uid,
		"role": role,
		"exp":  time.Now().Add(time.Minute * time.Duration(c.Exp)).Unix(),
		"iat":  time.Now().Unix(),
	})
	return token.SignedString([]byte(c.SecretKey))
}

// VerifyToken 验证token
func (c *TokenClient) VerifyToken(tokenStr string) (claims TokenClaims, err error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("not authorization")
		}
		return []byte(c.SecretKey), nil
	})
	if err != nil {
		return claims, err
	}
	if !token.Valid {
		return claims, fmt.Errorf("not authorization")
	}
	b, err := json.Marshal(token.Claims)
	if err != nil {
		return claims, err
	}
	jwtClaims := TokenClaims{}
	err = json.Unmarshal(b, &jwtClaims)
	if err != nil {
		return claims, err
	}
	return jwtClaims, nil
}

// VerifyTokenForRole 验证token
func (c *TokenClient) VerifyTokenForRole(tokenStr, role string) (claims TokenClaims, err error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("not authorization")
		}
		return []byte(c.SecretKey), nil
	})
	if err != nil {
		return claims, err
	}
	if !token.Valid {
		return claims, fmt.Errorf("not authorization")
	}
	b, err := json.Marshal(token.Claims)
	if err != nil {
		return claims, err
	}
	jwtClaims := TokenClaims{}
	err = json.Unmarshal(b, &jwtClaims)
	if err != nil {
		return jwtClaims, err
	}
	if jwtClaims.Role != role {
		return jwtClaims, fmt.Errorf("token role error")
	}
	return jwtClaims, nil
}

// RenewToken token 续期
func (c *TokenClient) RenewToken(tokenStr string) (string, error) {
	claims, err := c.VerifyToken(tokenStr)
	if err != nil {
		return "", err
	}
	return c.GenerateToken(claims.UID, claims.Role)
}
