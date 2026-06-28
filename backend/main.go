package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"todo-list/backend/config"
	"todo-list/backend/global"
	"todo-list/backend/model"

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

type createRoomRequest struct {
	Name string `json:"name"`
}

type createTaskRequest struct {
	Title string `json:"title"`
	Level string `json:"level"`
	Kind  string `json:"kind"`
}

type wechatSession struct {
	OpenID string
}

type authUserResponse struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

var exchangeWeChatCode = exchangeWeChatCodeWithAPI

func migrate(db *gorm.DB) {
	if err := db.AutoMigrate(&model.User{}, &model.Room{}, &model.Task{}); err != nil {
		log.Fatalf("auto migrate failed: %v", err)
	}

	if db.Migrator().HasIndex(&model.Room{}, "idx_rooms_name") {
		if err := db.Migrator().DropIndex(&model.Room{}, "idx_rooms_name"); err != nil {
			log.Printf("drop old list name index failed: %v", err)
		}
	}
}

func main() {
	config.LoadConfig()

	if err := global.Connect(); err != nil {
		log.Fatalf("database connection failed: %v", err)
	}

	if config.Global.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	log.Println("database:", config.Global.DB.DBName)

	migrate(global.DB)
	router := setupRouter(global.DB)

	port := os.Getenv("PORT")
	if port == "" {
		port = config.Global.Server.Port
	}

	if err := router.Run(":" + port); err != nil {
		panic(err)
	}
}

func setupRouter(db *gorm.DB) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), corsMiddleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	api := router.Group("/api")
	{
		api.POST("/auth/login", func(c *gin.Context) {
			loginWithPassword(c, db)
		})

		api.POST("/auth/wechat-login", func(c *gin.Context) {
			loginWithWeChat(c, db)
		})

		private := api.Group("")
		private.Use(requireAuth(db))
		{
			private.POST("/rooms", func(c *gin.Context) {
				user := currentUser(c)
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
				result := db.Where("user_id = ? AND name = ?", user.ID, name).FirstOrCreate(&room)
				if result.Error != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create list"})
					return
				}

				status := http.StatusCreated
				if result.RowsAffected == 0 {
					status = http.StatusOK
				}
				c.JSON(status, room)
			})

			roomTasks := private.Group("/rooms/:roomID/tasks")
			{
				roomTasks.GET("", func(c *gin.Context) {
					roomID, ok := parseParamID(c, "roomID", "room id must be a positive integer")
					if !ok {
						return
					}
					if _, ok := roomForUser(c, db, roomID, currentUser(c).ID); !ok {
						return
					}

					var tasks []model.Task
					if err := db.Where("room_id = ?", roomID).
						Order("created_at DESC, id DESC").
						Find(&tasks).Error; err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to query tasks"})
						return
					}

					c.JSON(http.StatusOK, gin.H{"items": tasks})
				})

				roomTasks.POST("", func(c *gin.Context) {
					roomID, ok := parseParamID(c, "roomID", "room id must be a positive integer")
					if !ok {
						return
					}
					if _, ok := roomForUser(c, db, roomID, currentUser(c).ID); !ok {
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
						Level:  normalizeLevel(payload.Level),
						Kind:   normalizeKind(payload.Kind),
					}

					if err := db.Create(&task).Error; err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create task"})
						return
					}
					c.JSON(http.StatusCreated, task)
				})

				roomTasks.PATCH("/:taskID/toggle", func(c *gin.Context) {
					roomID, ok := parseParamID(c, "roomID", "room id must be a positive integer")
					if !ok {
						return
					}
					if _, ok := roomForUser(c, db, roomID, currentUser(c).ID); !ok {
						return
					}
					taskID, ok := parseParamID(c, "taskID", "task id must be a positive integer")
					if !ok {
						return
					}

					var task model.Task
					if err := db.Where("id = ? AND room_id = ?", taskID, roomID).First(&task).Error; err != nil {
						if errors.Is(err, gorm.ErrRecordNotFound) {
							c.JSON(http.StatusNotFound, gin.H{"message": "task not found"})
						} else {
							c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to query task"})
						}
						return
					}

					task.Done = !task.Done
					if err := db.Save(&task).Error; err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update task"})
						return
					}

					c.JSON(http.StatusOK, task)
				})

				roomTasks.DELETE("/:taskID", func(c *gin.Context) {
					roomID, ok := parseParamID(c, "roomID", "room id must be a positive integer")
					if !ok {
						return
					}
					if _, ok := roomForUser(c, db, roomID, currentUser(c).ID); !ok {
						return
					}
					taskID, ok := parseParamID(c, "taskID", "task id must be a positive integer")
					if !ok {
						return
					}

					result := db.Where("id = ? AND room_id = ?", taskID, roomID).Delete(&model.Task{})
					if result.Error != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to delete task"})
						return
					}
					if result.RowsAffected == 0 {
						c.JSON(http.StatusNotFound, gin.H{"message": "task not found"})
						return
					}
					c.Status(http.StatusNoContent)
				})
			}
		}
	}

	return router
}

func loginWithPassword(c *gin.Context, db *gorm.DB) {
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
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
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
		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusConflict, gin.H{"message": "username already exists, please log in"})
			return
		}
		registered = true
	} else if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "incorrect password"})
		return
	}

	issueLoginToken(c, db, user, registered)
}

func loginWithWeChat(c *gin.Context, db *gorm.DB) {
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

	session, err := exchangeWeChatCode(code)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
		return
	}
	if session.OpenID == "" {
		c.JSON(http.StatusBadGateway, gin.H{"message": "wechat did not return openid"})
		return
	}

	var user model.User
	registered := false
	if err := db.Where("open_id = ?", session.OpenID).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to query wechat user"})
			return
		}

		openID := session.OpenID
		user = model.User{Username: wechatUsername(openID), OpenID: &openID, PasswordHash: "-"}
		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create wechat user"})
			return
		}
		registered = true
	}

	issueLoginToken(c, db, user, registered)
}

func issueLoginToken(c *gin.Context, db *gorm.DB, user model.User, registered bool) {
	token, err := newAuthToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to issue login token"})
		return
	}
	user.AuthTokenHash = hashToken(token)
	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to save login token"})
		return
	}

	status := http.StatusOK
	if registered {
		status = http.StatusCreated
	}
	c.JSON(status, gin.H{"user": publicUser(user, token), "registered": registered})
}

func exchangeWeChatCodeWithAPI(code string) (wechatSession, error) {
	appID := strings.TrimSpace(config.Global.WeChat.AppID)
	secret := strings.TrimSpace(config.Global.WeChat.AppSecret)
	if appID == "" || secret == "" {
		return wechatSession{}, errors.New("wechat appid or secret is not configured")
	}

	query := url.Values{}
	query.Set("appid", appID)
	query.Set("secret", secret)
	query.Set("js_code", code)
	query.Set("grant_type", "authorization_code")

	client := http.Client{Timeout: 8 * time.Second}
	response, err := client.Get("https://api.weixin.qq.com/sns/jscode2session?" + query.Encode())
	if err != nil {
		return wechatSession{}, fmt.Errorf("wechat login request failed: %w", err)
	}
	defer response.Body.Close()

	var body struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return wechatSession{}, fmt.Errorf("decode wechat login response failed: %w", err)
	}
	if body.ErrCode != 0 {
		message := body.ErrMsg
		if message == "" {
			message = "wechat login failed"
		}
		return wechatSession{}, fmt.Errorf("wechat login failed: %s", message)
	}

	return wechatSession{OpenID: strings.TrimSpace(body.OpenID)}, nil
}

func parseParamID(c *gin.Context, name string, message string) (int, bool) {
	id, err := strconv.Atoi(c.Param(name))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": message})
		return 0, false
	}

	return id, true
}

func requireAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "login required"})
			return
		}

		var user model.User
		if err := db.Where("auth_token_hash = ?", hashToken(token)).First(&user).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "login expired"})
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func currentUser(c *gin.Context) model.User {
	user, _ := c.Get("user")
	return user.(model.User)
}

func roomForUser(c *gin.Context, db *gorm.DB, roomID int, userID int) (model.Room, bool) {
	var room model.Room
	if err := db.Where("id = ? AND user_id = ?", roomID, userID).First(&room).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "list not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to query list"})
		}
		return room, false
	}

	return room, true
}

func publicUser(user model.User, token string) authUserResponse {
	return authUserResponse{ID: user.ID, Username: user.Username, Token: token, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt}
}

func newAuthToken() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func wechatUsername(openID string) string {
	value := strings.TrimSpace(openID)
	if len(value) > 32 {
		value = value[:32]
	}
	if value == "" {
		value = "user"
	}
	return "wx_" + value
}

func normalizeLevel(level string) string {
	level = strings.TrimSpace(level)
	switch level {
	case "基础", "进阶", "挑战", "鍩虹", "杩涢樁", "鎸戞垬":
		return level
	default:
		return "基础"
	}
}

func normalizeKind(kind string) string {
	kind = strings.TrimSpace(kind)
	if kind == "optimization" {
		return kind
	}

	return "learning"
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
