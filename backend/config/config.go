package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Server struct {
	Mode string `mapstructure:"mode"`
	Port string `mapstructure:"port"`
}

type DB struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"name"`
	Charset  string `mapstructure:"charset"`
}

type Config struct {
	Server Server `mapstructure:"backend"`
	DB     DB     `mapstructure:"database"`
}

var Global Config

func (d *DB) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.Charset)
}

func LoadConfig() {
	cfg, err := Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	Global = cfg
	log.Println("配置加载完成：mode =", Global.Server.Mode, "port =", Global.Server.Port)
}

func Load() (Config, error) {
	return LoadFromPath(configPath())
}

func LoadFromPath(path string) (Config, error) {
	v := viper.New()
	if isConfigFile(path) {
		v.SetConfigFile(path)
	} else {
		v.AddConfigPath(path)
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read config file from %s: %w", path, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}

func configPath() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return "../config"
	}

	candidates := []string{
		filepath.Join(workingDir, "../config"),
		filepath.Join(workingDir, "../../config"),
		filepath.Join(workingDir, "config"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "config.yaml")); err == nil {
			return candidate
		}
	}

	return "../config"
}

func isConfigFile(path string) bool {
	info, err := os.Stat(path)
	if err == nil {
		return !info.IsDir()
	}

	ext := filepath.Ext(path)
	return ext == ".yaml" || ext == ".yml"
}
