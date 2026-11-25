package model

import "strings"

// 任务时间紧迫度。客户端传入未知值时统一归一为 LevelSomeday。
const (
	LevelToday   = "今天"
	LevelWeek    = "本周"
	LevelSoon    = "近期"
	LevelSomeday = "有空"
)

// NormalizeLevel 把客户端传入的紧迫度归一为合法值。
func NormalizeLevel(level string) string {
	switch strings.TrimSpace(level) {
	case LevelToday:
		return LevelToday
	case LevelWeek:
		return LevelWeek
	case LevelSoon:
		return LevelSoon
	default:
		return LevelSomeday
	}
}
