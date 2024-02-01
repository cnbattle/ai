package token

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Client struct {
	SecretKey string // 密钥
	Exp       int    // 分钟
}

// NewClient 初始化一个 Token Client
func NewClient(secretKey string, exp int) *Client {
	return &Client{SecretKey: secretKey, Exp: exp}
}

// GenerateToken 生成一个token
func (c *Client) GenerateToken(uid, role string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid":  uid,
		"role": role,
		"exp":  time.Now().Add(time.Minute * time.Duration(c.Exp)).Unix(),
		"iat":  time.Now().Unix(),
	})
	return token.SignedString([]byte(c.SecretKey))
}

// VerifyToken 验证token
func (c *Client) VerifyToken(tokenStr string) (claims Claims, err error) {
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
	jwtClaims := Claims{}
	err = json.Unmarshal(b, &jwtClaims)
	if err != nil {
		return claims, err
	}
	return jwtClaims, nil
}

// VerifyToken 验证token
func (c *Client) VerifyTokenWithCtx(ctx *gin.Context) (claims Claims, err error) {
	tokenStr, _ := ctx.Cookie("Authorization")
	if len(tokenStr) == 0 {
		tokenStr = ctx.GetHeader("Authorization")
	}
	return c.VerifyToken(tokenStr)
}

// VerifyTokenForRole 验证token
func (c *Client) VerifyTokenForRole(tokenStr, role string) (claims Claims, err error) {
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
	jwtClaims := Claims{}
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
func (c *Client) RenewToken(tokenStr string) (string, error) {
	claims, err := c.VerifyToken(tokenStr)
	if err != nil {
		return "", err
	}
	return c.GenerateToken(claims.UID, claims.Role)
}
