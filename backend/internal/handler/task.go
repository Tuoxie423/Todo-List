package handler

import (
	"errors"
	"net/http"
	"strings"

	"todo-list/backend/internal/auth"
	"todo-list/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// taskLevelOrderSQL 按紧迫度排序：今天 > 本周 > 近期 > 有空。
const taskLevelOrderSQL = "CASE level WHEN '" + model.LevelToday + "' THEN 0 WHEN '" + model.LevelWeek + "' THEN 1 WHEN '" + model.LevelSoon + "' THEN 2 WHEN '" + model.LevelSomeday + "' THEN 3 ELSE 3 END"

type createTaskRequest struct {
	Title string `json:"title"`
	Level string `json:"level"`
}

// ListTasks 列出指定清单下的任务。
func (h *Handler) ListTasks(c *gin.Context) {
	roomID, ok := parseParamID(c, "roomID", "room id must be a positive integer")
	if !ok {
		return
	}
	if _, ok := h.roomForUser(c, roomID, auth.CurrentUser(c).ID); !ok {
		return
	}

	var tasks []model.Task
	if err := h.db.Where("room_id = ?", roomID).
		Order(taskLevelOrderSQL).
		Order("created_at DESC, id DESC").
		Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to query tasks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": tasks})
}

// CreateTask 在指定清单下新增任务。
func (h *Handler) CreateTask(c *gin.Context) {
	roomID, ok := parseParamID(c, "roomID", "room id must be a positive integer")
	if !ok {
		return
	}
	if _, ok := h.roomForUser(c, roomID, auth.CurrentUser(c).ID); !ok {
		return
	}

	var payload createTaskRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "request body must be JSON"})
		return
	}

	title := strings.TrimSpace(payload.Title)
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "task title is required"})
		return
	}

	task := model.Task{
		RoomID: roomID,
		Title:  title,
		Level:  model.NormalizeLevel(payload.Level),
	}

	if err := h.db.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create task"})
		return
	}
	c.JSON(http.StatusCreated, task)
}

// ToggleTask 切换任务的完成状态。
func (h *Handler) ToggleTask(c *gin.Context) {
	roomID, ok := parseParamID(c, "roomID", "room id must be a positive integer")
	if !ok {
		return
	}
	if _, ok := h.roomForUser(c, roomID, auth.CurrentUser(c).ID); !ok {
		return
	}
	taskID, ok := parseParamID(c, "taskID", "task id must be a positive integer")
	if !ok {
		return
	}

	var task model.Task
	if err := h.db.Where("id = ? AND room_id = ?", taskID, roomID).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to query task"})
		}
		return
	}

	task.Done = !task.Done
	if err := h.db.Save(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update task"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// DeleteTask 删除指定清单下的任务。
func (h *Handler) DeleteTask(c *gin.Context) {
	roomID, ok := parseParamID(c, "roomID", "room id must be a positive integer")
	if !ok {
		return
	}
	if _, ok := h.roomForUser(c, roomID, auth.CurrentUser(c).ID); !ok {
		return
	}
	taskID, ok := parseParamID(c, "taskID", "task id must be a positive integer")
	if !ok {
		return
	}

	result := h.db.Where("id = ? AND room_id = ?", taskID, roomID).Delete(&model.Task{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to delete task"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "task not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
