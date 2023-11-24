package ai

import (
	"os"
	"strconv"
	"strings"

	// 自动加载配置文件
	_ "github.com/joho/godotenv/autoload"
)

// GetEnv GetEnv
func GetEnv(key string) (value string) {
	if len(key) == 0 {
		return ""
	}
	return os.Getenv(key)
}

// GetDefaultEnv GetDefaultEnv
func GetDefaultEnv(key, defaultValue string) (value string) {
	value = GetEnv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetEnvToInt GetEnvToInt
func GetEnvToInt(key string) (value int) {
	tmp := GetEnv(key)
	if tmp == "" {
		return 0
	}
	value, _ = strconv.Atoi(tmp)
	return value
}

// GetDefaultEnvToInt GetDefaultEnvToInt
func GetDefaultEnvToInt(key string, defaultValue int) (value int) {
	tmp := GetEnv(key)
	if tmp == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(tmp)
	if err != nil {
		return defaultValue
	}
	return value
}

// GetEnvToBool GetEnvToBool
func GetEnvToBool(key string) (value bool) {
	switch strings.ToUpper(GetEnv(key)) {
	case "TRUE":
		return true
	default:
		return false
	}
}

// GetDefaultEnvToBool GetDefaultEnvToBool
func GetDefaultEnvToBool(key string, defaultValue bool) (value bool) {
	tmp := GetEnv(key)
	if tmp == "" {
		return defaultValue
	}
	switch strings.ToUpper(tmp) {
	case "TRUE":
		return true
	case "FALSE":
		return false
	default:
		return defaultValue
	}
}
