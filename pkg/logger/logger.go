package logger

import (
	"log"
	"os"
	"strings"

	"github.com/cnpg-broker/pkg/config"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

var (
	logger       *log.Logger
	currentLevel Level
)

func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
}

func Init() {
	cfg := config.Get()
	currentLevel = parseLevel(cfg.LogLevel)
}

func parseLevel(level string) Level {
	switch strings.ToLower(level) {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn", "warning":
		return WARN
	case "error":
		return ERROR
	case "fatal":
		return FATAL
	default:
		return INFO
	}
}

func Debug(format string, v ...interface{}) {
	if currentLevel <= DEBUG {
		logger.Printf("[DEBUG] "+format, v...)
	}
}

func Info(format string, v ...interface{}) {
	if currentLevel <= INFO {
		logger.Printf("[INFO] "+format, v...)
	}
}

func Warn(format string, v ...interface{}) {
	if currentLevel <= WARN {
		logger.Printf("[WARN] "+format, v...)
	}
}

func Error(format string, v ...interface{}) {
	if currentLevel <= ERROR {
		logger.Printf("[ERROR] "+format, v...)
	}
}

func Fatal(format string, v ...interface{}) {
	logger.Fatalf("[FATAL] "+format, v...)
}
