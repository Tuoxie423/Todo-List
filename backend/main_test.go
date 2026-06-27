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
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Room{}, &model.Task{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return db
}

func createTestRoom(t *testing.T, db *gorm.DB, name string) model.Room {
	t.Helper()

	room := model.Room{Name: name}
	if err := db.Create(&room).Error; err != nil {
		t.Fatalf("create room: %v", err)
	}

	return room
}

func TestCreateRoomCreatesAndReusesByName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	router := setupRouter(db)

	first := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewBufferString(`{"name":" Go 练习 "}`))
	firstRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(first, firstRequest)

	if first.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", first.Code, first.Body.String())
	}

	var created model.Room
	if err := json.Unmarshal(first.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created room: %v", err)
	}
	if created.Name != "Go 练习" {
		t.Fatalf("expected trimmed room name, got %q", created.Name)
	}

	second := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewBufferString(`{"name":"Go 练习"}`))
	secondRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(second, secondRequest)

	if second.Code != http.StatusOK {
		t.Fatalf("expected 200 for existing room, got %d: %s", second.Code, second.Body.String())
	}

	var count int64
	db.Model(&model.Room{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected one room, got %d", count)
	}
}

func TestCreateRoomValidatesName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	router := setupRouter(db)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewBufferString(`{"name":"   "}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
}

func TestListRoomTasks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	roomA := createTestRoom(t, db, "A")
	roomB := createTestRoom(t, db, "B")
	db.Create(&model.Task{RoomID: roomA.ID, Title: "A1", Kind: "learning"})
	db.Create(&model.Task{RoomID: roomB.ID, Title: "B1", Kind: "optimization"})
	router := setupRouter(db)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/rooms/"+strconv.Itoa(roomA.ID)+"/tasks", nil)
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
		t.Fatalf("expected only room A task, got %+v", body.Items)
	}
}

func TestListRoomTasksRoomNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	router := setupRouter(db)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/rooms/999/tasks", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.Code)
	}
}

func TestCreateRoomTaskSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	room := createTestRoom(t, db, "Go")
	router := setupRouter(db)

	body := bytes.NewBufferString(`{"title":"学习 GORM","level":"进阶","kind":"optimization"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/rooms/"+strconv.Itoa(room.ID)+"/tasks", body)
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

func TestCreateRoomTaskNormalizesFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	room := createTestRoom(t, db, "Go")
	router := setupRouter(db)

	body := bytes.NewBufferString(`{"title":"随便写","level":"其他","kind":"其他"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/rooms/"+strconv.Itoa(room.ID)+"/tasks", body)
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
	room := createTestRoom(t, db, "Go")
	router := setupRouter(db)

	body := bytes.NewBufferString(`{"title":"   "}`)
	request := httptest.NewRequest(http.MethodPost, "/api/rooms/"+strconv.Itoa(room.ID)+"/tasks", body)
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
	room := createTestRoom(t, db, "Go")
	task := model.Task{RoomID: room.ID, Title: "Toggle", Done: false}
	db.Create(&task)
	router := setupRouter(db)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/api/rooms/"+strconv.Itoa(room.ID)+"/tasks/"+strconv.Itoa(task.ID)+"/toggle", nil)
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

func TestToggleRoomTaskDoesNotCrossRooms(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	roomA := createTestRoom(t, db, "A")
	roomB := createTestRoom(t, db, "B")
	task := model.Task{RoomID: roomA.ID, Title: "A task"}
	db.Create(&task)
	router := setupRouter(db)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/api/rooms/"+strconv.Itoa(roomB.ID)+"/tasks/"+strconv.Itoa(task.ID)+"/toggle", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.Code)
	}
}

func TestDeleteRoomTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	room := createTestRoom(t, db, "Go")
	task := model.Task{RoomID: room.ID, Title: "Delete"}
	db.Create(&task)
	router := setupRouter(db)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/api/rooms/"+strconv.Itoa(room.ID)+"/tasks/"+strconv.Itoa(task.ID), nil)
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
