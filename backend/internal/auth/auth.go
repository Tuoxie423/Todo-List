// Package auth 负责登录令牌的生成、摘要与请求上下文中的用户读取。
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"todo-list/backend/internal/model"

	"github.com/gin-gonic/gin"
)

const userContextKey = "user"

// NewToken 生成新的随机登录令牌。
func NewToken() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// HashToken 计算令牌摘要，数据库只保存摘要而不是明文令牌。
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// BearerToken 从 Authorization 头中取出令牌。
func BearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

// SetUser 把已认证用户写入请求上下文。
func SetUser(c *gin.Context, user model.User) {
	c.Set(userContextKey, user)
}

// CurrentUser 读取 RequireAuth 写入的当前用户。
func CurrentUser(c *gin.Context) model.User {
	value, ok := c.Get(userContextKey)
	if !ok {
		return model.User{}
	}
	user, _ := value.(model.User)
	return user
}
