package handler

import (
	"net/http"
	"strings"

	"todo-list/backend/internal/auth"
	"todo-list/backend/internal/model"

	"github.com/gin-gonic/gin"
)

type createRoomRequest struct {
	Name string `json:"name"`
}

// CreateRoom 创建清单；同一用户同名清单直接返回已有记录。
func (h *Handler) CreateRoom(c *gin.Context) {
	user := auth.CurrentUser(c)

	var payload createRoomRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "request body must be JSON"})
		return
	}

	name := strings.TrimSpace(payload.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "list name is required"})
		return
	}

	room := model.Room{UserID: user.ID, Name: name}
	result := h.db.Where("user_id = ? AND name = ?", user.ID, name).FirstOrCreate(&room)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create list"})
		return
	}

	status := http.StatusCreated
	if result.RowsAffected == 0 {
		status = http.StatusOK
	}
	c.JSON(status, room)
}
