package handler

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"todo-list/backend/internal/auth"
	"todo-list/backend/internal/model"
	"todo-list/backend/internal/wechat"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type wechatLoginRequest struct {
	Code string `json:"code"`
}

type authUserResponse struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// LoginWithPassword 登录；用户名不存在时自动注册。
func (h *Handler) LoginWithPassword(c *gin.Context) {
	var payload loginRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "request body must be JSON"})
		return
	}

	username := strings.TrimSpace(payload.Username)
	password := payload.Password
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "username is required"})
		return
	}
	if len(username) > 40 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "username cannot exceed 40 characters"})
		return
	}
	if len(password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "password must be at least 6 characters"})
		return
	}

	var user model.User
	registered := false
	if err := h.db.Where("username = ?", username).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to query user"})
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to encrypt password"})
			return
		}

		user = model.User{Username: username, PasswordHash: string(hash)}
		if err := h.db.Create(&user).Error; err != nil {
			c.JSON(http.StatusConflict, gin.H{"message": "username already exists, please log in"})
			return
		}
		registered = true
	} else if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "incorrect password"})
		return
	}

	h.issueLoginToken(c, user, registered)
}

// LoginWithWeChat 用小程序 code 登录；openid 不存在时自动注册。
func (h *Handler) LoginWithWeChat(c *gin.Context) {
	var payload wechatLoginRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "request body must be JSON"})
		return
	}

	code := strings.TrimSpace(payload.Code)
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "wechat code is required"})
		return
	}

	session, err := h.wechat.ExchangeCode(code)
	if err != nil {
		log.Printf("wechat login failed: %v", wechat.RedactSecret(err.Error()))
		c.JSON(http.StatusBadGateway, gin.H{"message": "wechat login failed"})
		return
	}
	if session.OpenID == "" {
		c.JSON(http.StatusBadGateway, gin.H{"message": "wechat did not return openid"})
		return
	}

	var user model.User
	registered := false
	if err := h.db.Where("open_id = ?", session.OpenID).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to query wechat user"})
			return
		}

		openID := session.OpenID
		user = model.User{Username: temporaryWeChatUsername(), OpenID: &openID, PasswordHash: "-"}
		if err := h.db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create wechat user"})
			return
		}
		user.Username = wechatUsername(user.ID)
		registered = true
	} else if shouldResetWeChatUsername(user.Username, session.OpenID) {
		user.Username = wechatUsername(user.ID)
	}

	h.issueLoginToken(c, user, registered)
}

// issueLoginToken 签发令牌并保存令牌摘要。
func (h *Handler) issueLoginToken(c *gin.Context, user model.User, registered bool) {
	token, err := auth.NewToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to issue login token"})
		return
	}
	user.AuthTokenHash = auth.HashToken(token)
	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to save login token"})
		return
	}

	status := http.StatusOK
	if registered {
		status = http.StatusCreated
	}
	c.JSON(status, gin.H{"user": publicUser(user, token), "registered": registered})
}

func publicUser(user model.User, token string) authUserResponse {
	return authUserResponse{ID: user.ID, Username: user.Username, Token: token, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt}
}

func wechatUsername(userID int) string {
	return fmt.Sprintf("微信用户%d", userID)
}

func temporaryWeChatUsername() string {
	data := make([]byte, 4)
	if _, err := rand.Read(data); err != nil {
		return fmt.Sprintf("wx_user_%08d", time.Now().UnixNano()%100000000)
	}
	number := (uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])) % 100000000
	return fmt.Sprintf("wx_user_%08d", number)
}

func shouldResetWeChatUsername(username string, openID string) bool {
	username = strings.TrimSpace(username)
	return username == "" || strings.HasPrefix(username, "wx_") || (openID != "" && strings.Contains(username, openID))
}
