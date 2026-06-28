package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"todo-list/backend/model"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Room{}, &model.Task{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return db
}

func createTestUser(t *testing.T, db *gorm.DB, username string, token string) model.User {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := model.User{Username: username, PasswordHash: string(hash), AuthTokenHash: hashToken(token)}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	return user
}

func createTestRoom(t *testing.T, db *gorm.DB, userID int, name string) model.Room {
	t.Helper()

	room := model.Room{UserID: userID, Name: name}
	if err := db.Create(&room).Error; err != nil {
		t.Fatalf("create room: %v", err)
	}

	return room
}

func authRequest(method string, target string, token string, body *bytes.Buffer) *http.Request {
	if body == nil {
		body = bytes.NewBuffer(nil)
	}
	request := httptest.NewRequest(method, target, body)
	request.Header.Set("Authorization", "Bearer "+token)
	if body.Len() > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

func TestWeChatLoginAutoRegistersAndReusesUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	oldExchange := exchangeWeChatCode
	exchangeWeChatCode = func(code string) (wechatSession, error) {
		if code != "wx-code" {
			t.Fatalf("unexpected code: %s", code)
		}
		return wechatSession{OpenID: "openid-alice"}, nil
	}
	defer func() { exchangeWeChatCode = oldExchange }()
	router := setupRouter(db)

	first := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodPost, "/api/auth/wechat-login", bytes.NewBufferString(`{"code":"wx-code"}`))
	firstRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(first, firstRequest)

	if first.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", first.Code, first.Body.String())
	}

	var firstBody struct {
		Registered bool `json:"registered"`
		User       struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
			Token    string `json:"token"`
		} `json:"user"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &firstBody); err != nil {
		t.Fatalf("decode wechat login body: %v", err)
	}
	if !firstBody.Registered || firstBody.User.Token == "" || firstBody.User.Username != "wx_openid-alice" {
		t.Fatalf("unexpected first login body: %+v", firstBody)
	}

	second := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodPost, "/api/auth/wechat-login", bytes.NewBufferString(`{"code":"wx-code"}`))
	secondRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(second, secondRequest)

	if second.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", second.Code, second.Body.String())
	}

	var count int64
	db.Model(&model.User{}).Where("open_id = ?", "openid-alice").Count(&count)
	if count != 1 {
		t.Fatalf("expected one wechat user, got %d", count)
	}
}

func TestWeChatLoginValidatesCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	router := setupRouter(db)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/wechat-login", bytes.NewBufferString(`{"code":"   "}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
}
func TestLoginAutoRegistersAndStoresPasswordHashAndTokenHash(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	router := setupRouter(db)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":" alice ","password":"secret123"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}

	var body struct {
		Registered bool `json:"registered"`
		User       struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
			Token    string `json:"token"`
		} `json:"user"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode login body: %v", err)
	}
	if !body.Registered || body.User.Username != "alice" || body.User.Token == "" {
		t.Fatalf("expected alice to be auto registered with a token, got %+v", body)
	}

	var user model.User
	if err := db.Where("username = ?", "alice").First(&user).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if user.PasswordHash == "secret123" {
		t.Fatal("expected password to be hashed, got plaintext")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("secret123")); err != nil {
		t.Fatalf("stored password hash does not match: %v", err)
	}
	if user.AuthTokenHash == "" || user.AuthTokenHash == body.User.Token {
		t.Fatalf("expected stored auth token to be hashed, got %q", user.AuthTokenHash)
	}
	if user.AuthTokenHash != hashToken(body.User.Token) {
		t.Fatal("stored auth token hash does not match issued token")
	}
}

func TestLoginExistingUserDoesNotCreateDuplicate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	router := setupRouter(db)

	for i := 0; i < 2; i++ {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"alice","password":"secret123"}`))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, request)

		want := http.StatusCreated
		if i == 1 {
			want = http.StatusOK
		}
		if response.Code != want {
			t.Fatalf("attempt %d expected %d, got %d: %s", i+1, want, response.Code, response.Body.String())
		}
	}

	var count int64
	db.Model(&model.User{}).Where("username = ?", "alice").Count(&count)
	if count != 1 {
		t.Fatalf("expected one alice user, got %d", count)
	}
}

func TestLoginExistingUserRejectsWrongPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	createTestUser(t, db, "alice", "alice-token")
	router := setupRouter(db)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"alice","password":"wrong123"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", response.Code, response.Body.String())
	}

	var count int64
	db.Model(&model.User{}).Where("username = ?", "alice").Count(&count)
	if count != 1 {
		t.Fatalf("expected one alice user, got %d", count)
	}
}

func TestCreateRoomCreatesAndReusesByNamePerUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	alice := createTestUser(t, db, "alice", "alice-token")
	bob := createTestUser(t, db, "bob", "bob-token")
	router := setupRouter(db)

	first := httptest.NewRecorder()
	firstRequest := authRequest(http.MethodPost, "/api/rooms", "alice-token", bytes.NewBufferString(`{"name":" Project "}`))
	firstRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(first, firstRequest)

	if first.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", first.Code, first.Body.String())
	}

	var created model.Room
	if err := json.Unmarshal(first.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created room: %v", err)
	}
	if created.Name != "Project" || created.UserID != alice.ID {
		t.Fatalf("expected alice private room, got %+v", created)
	}

	second := httptest.NewRecorder()
	secondRequest := authRequest(http.MethodPost, "/api/rooms", "alice-token", bytes.NewBufferString(`{"name":"Project"}`))
	secondRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(second, secondRequest)

	if second.Code != http.StatusOK {
		t.Fatalf("expected 200 for existing alice room, got %d: %s", second.Code, second.Body.String())
	}

	third := httptest.NewRecorder()
	thirdRequest := authRequest(http.MethodPost, "/api/rooms", "bob-token", bytes.NewBufferString(`{"name":"Project"}`))
	thirdRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(third, thirdRequest)

	if third.Code != http.StatusCreated {
		t.Fatalf("expected 201 for bob room with same name, got %d: %s", third.Code, third.Body.String())
	}

	var count int64
	db.Model(&model.Room{}).Where("name = ?", "Project").Count(&count)
	if count != 2 {
		t.Fatalf("expected two private Project rooms, got %d", count)
	}
	if bob.ID == alice.ID {
		t.Fatal("test users should be distinct")
	}
}

func TestCreateRoomRequiresLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	router := setupRouter(db)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewBufferString(`{"name":"Project"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestCreateRoomValidatesName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	createTestUser(t, db, "alice", "alice-token")
	router := setupRouter(db)

	response := httptest.NewRecorder()
	request := authRequest(http.MethodPost, "/api/rooms", "alice-token", bytes.NewBufferString(`{"name":"   "}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
}

func TestListRoomTasksOnlyAllowsRoomOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	alice := createTestUser(t, db, "alice", "alice-token")
	bob := createTestUser(t, db, "bob", "bob-token")
	roomA := createTestRoom(t, db, alice.ID, "A")
	roomB := createTestRoom(t, db, bob.ID, "B")
	db.Create(&model.Task{RoomID: roomA.ID, Title: "A1", Kind: "learning"})
	db.Create(&model.Task{RoomID: roomB.ID, Title: "B1", Kind: "optimization"})
	router := setupRouter(db)

	response := httptest.NewRecorder()
	request := authRequest(http.MethodGet, "/api/rooms/"+strconv.Itoa(roomA.ID)+"/tasks", "alice-token", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}

	var body struct {
		Items []model.Task `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode tasks: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].Title != "A1" {
		t.Fatalf("expected only alice room task, got %+v", body.Items)
	}

	forbidden := httptest.NewRecorder()
	forbiddenRequest := authRequest(http.MethodGet, "/api/rooms/"+strconv.Itoa(roomA.ID)+"/tasks", "bob-token", nil)
	router.ServeHTTP(forbidden, forbiddenRequest)

	if forbidden.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for bob reading alice room, got %d", forbidden.Code)
	}
}

func TestListRoomTasksRoomNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	createTestUser(t, db, "alice", "alice-token")
	router := setupRouter(db)

	response := httptest.NewRecorder()
	request := authRequest(http.MethodGet, "/api/rooms/999/tasks", "alice-token", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.Code)
	}
}

func TestCreateRoomTaskSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	alice := createTestUser(t, db, "alice", "alice-token")
	room := createTestRoom(t, db, alice.ID, "Go")
	router := setupRouter(db)

	body := bytes.NewBufferString(`{"title":"Learn GORM","level":"进阶","kind":"optimization"}`)
	request := authRequest(http.MethodPost, "/api/rooms/"+strconv.Itoa(room.ID)+"/tasks", "alice-token", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}

	var task model.Task
	if err := json.Unmarshal(response.Body.Bytes(), &task); err != nil {
		t.Fatalf("decode task: %v", err)
	}
	if task.RoomID != room.ID {
		t.Fatalf("expected room id %d, got %d", room.ID, task.RoomID)
	}
	if task.Level != "进阶" || task.Kind != "optimization" {
		t.Fatalf("unexpected normalized fields: %+v", task)
	}
}

func TestCreateRoomTaskRejectsOtherUsersRoom(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	alice := createTestUser(t, db, "alice", "alice-token")
	createTestUser(t, db, "bob", "bob-token")
	room := createTestRoom(t, db, alice.ID, "Go")
	router := setupRouter(db)

	body := bytes.NewBufferString(`{"title":"Steal room","level":"进阶","kind":"optimization"}`)
	request := authRequest(http.MethodPost, "/api/rooms/"+strconv.Itoa(room.ID)+"/tasks", "bob-token", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
}

func TestCreateRoomTaskNormalizesFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	alice := createTestUser(t, db, "alice", "alice-token")
	room := createTestRoom(t, db, alice.ID, "Go")
	router := setupRouter(db)

	body := bytes.NewBufferString(`{"title":"Anything","level":"Other","kind":"Other"}`)
	request := authRequest(http.MethodPost, "/api/rooms/"+strconv.Itoa(room.ID)+"/tasks", "alice-token", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}

	var task model.Task
	json.Unmarshal(response.Body.Bytes(), &task)
	if task.Level != "基础" {
		t.Fatalf("expected default level 基础, got %q", task.Level)
	}
	if task.Kind != "learning" {
		t.Fatalf("expected default kind learning, got %q", task.Kind)
	}
}

func TestCreateRoomTaskValidatesTitle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	alice := createTestUser(t, db, "alice", "alice-token")
	room := createTestRoom(t, db, alice.ID, "Go")
	router := setupRouter(db)

	body := bytes.NewBufferString(`{"title":"   "}`)
	request := authRequest(http.MethodPost, "/api/rooms/"+strconv.Itoa(room.ID)+"/tasks", "alice-token", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
}

func TestToggleRoomTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	alice := createTestUser(t, db, "alice", "alice-token")
	room := createTestRoom(t, db, alice.ID, "Go")
	task := model.Task{RoomID: room.ID, Title: "Toggle", Done: false}
	db.Create(&task)
	router := setupRouter(db)

	response := httptest.NewRecorder()
	request := authRequest(http.MethodPatch, "/api/rooms/"+strconv.Itoa(room.ID)+"/tasks/"+strconv.Itoa(task.ID)+"/toggle", "alice-token", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}

	var toggled model.Task
	json.Unmarshal(response.Body.Bytes(), &toggled)
	if !toggled.Done {
		t.Fatal("expected task to be done")
	}
}

func TestToggleRoomTaskRejectsOtherUsersRoom(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	alice := createTestUser(t, db, "alice", "alice-token")
	createTestUser(t, db, "bob", "bob-token")
	room := createTestRoom(t, db, alice.ID, "Go")
	task := model.Task{RoomID: room.ID, Title: "A task"}
	db.Create(&task)
	router := setupRouter(db)

	response := httptest.NewRecorder()
	request := authRequest(http.MethodPatch, "/api/rooms/"+strconv.Itoa(room.ID)+"/tasks/"+strconv.Itoa(task.ID)+"/toggle", "bob-token", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.Code)
	}
}

func TestDeleteRoomTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	alice := createTestUser(t, db, "alice", "alice-token")
	room := createTestRoom(t, db, alice.ID, "Go")
	task := model.Task{RoomID: room.ID, Title: "Delete"}
	db.Create(&task)
	router := setupRouter(db)

	response := httptest.NewRecorder()
	request := authRequest(http.MethodDelete, "/api/rooms/"+strconv.Itoa(room.ID)+"/tasks/"+strconv.Itoa(task.ID), "alice-token", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", response.Code, response.Body.String())
	}

	var count int64
	db.Model(&model.Task{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected no tasks after delete, got %d", count)
	}
}

func TestDeleteRoomTaskRejectsOtherUsersRoom(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	alice := createTestUser(t, db, "alice", "alice-token")
	createTestUser(t, db, "bob", "bob-token")
	room := createTestRoom(t, db, alice.ID, "Go")
	task := model.Task{RoomID: room.ID, Title: "Delete"}
	db.Create(&task)
	router := setupRouter(db)

	response := httptest.NewRecorder()
	request := authRequest(http.MethodDelete, "/api/rooms/"+strconv.Itoa(room.ID)+"/tasks/"+strconv.Itoa(task.ID), "bob-token", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.Code)
	}

	var count int64
	db.Model(&model.Task{}).Where("id = ?", task.ID).Count(&count)
	if count != 1 {
		t.Fatalf("expected alice task to remain, got count %d", count)
	}
}
