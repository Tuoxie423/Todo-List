// Package handler 实现各 API 的请求处理。
package handler

import (
	"errors"
	"net/http"
	"strconv"

	"todo-list/backend/internal/model"
	"todo-list/backend/internal/wechat"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler 持有处理请求所需的依赖。
type Handler struct {
	db     *gorm.DB
	wechat wechat.Exchanger
}

func New(db *gorm.DB, wx wechat.Exchanger) *Handler {
	return &Handler{db: db, wechat: wx}
}

// parseParamID 解析路径中的正整数 ID，失败时直接返回 400。
func parseParamID(c *gin.Context, name string, message string) (int, bool) {
	id, err := strconv.Atoi(c.Param(name))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": message})
		return 0, false
	}

	return id, true
}

// roomForUser 校验清单归属，非本人清单按不存在处理。
func (h *Handler) roomForUser(c *gin.Context, roomID int, userID int) (model.Room, bool) {
	var room model.Room
	if err := h.db.Where("id = ? AND user_id = ?", roomID, userID).First(&room).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "list not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to query list"})
		}
		return room, false
	}

	return room, true
}
