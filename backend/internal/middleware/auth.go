package middleware

import (
	"net/http"

	"todo-list/backend/internal/auth"
	"todo-list/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RequireAuth 校验 Bearer 令牌，并把对应用户写入请求上下文。
func RequireAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := auth.BearerToken(c.GetHeader("Authorization"))
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "login required"})
			return
		}

		var user model.User
		if err := db.Where("auth_token_hash = ?", auth.HashToken(token)).First(&user).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "login expired"})
			return
		}

		auth.SetUser(c, user)
		c.Next()
	}
}
