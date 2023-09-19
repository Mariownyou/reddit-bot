package logger

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
)

var ResetColor  = "\033[0m"
var RedColor    = "\033[31m"
var GreenColor  = "\033[32m"
var YellowColor = "\033[33m"
var BlueColor   = "\033[34m"
var PurpleColor = "\033[35m"
var CyanColor   = "\033[36m"
var GrayColor   = "\033[37m"
var WhiteColor  = "\033[97m"

type MyLogger struct {
	logger *log.Logger
}

var (
	once      sync.Once
	singleton *MyLogger
)

func New() *MyLogger {
	once.Do(func() {
		singleton = createLogger()
	})
	return singleton
}

func createLogger() *MyLogger {
	logger := log.New(log.Writer(), "", log.Ldate|log.Ltime)
	return &MyLogger{logger}
}

// Logf logs a message along with the calling function's name.
func (ml *MyLogger) Logf(color string, format string, args ...interface{}) {
	pc, _, _, _ := runtime.Caller(2)  // @TODO remove all logs without purpose, or make them info logs
	callerFunc := runtime.FuncForPC(pc).Name()

	parts := strings.Split(callerFunc, "/")
	funcName := parts[len(parts)-1]

	message := fmt.Sprintf("%s[%s]%s %s", color, funcName, ResetColor, format)
	ml.logger.Printf(message, args...)
}

func Green(format string, args ...interface{}) {
	log := New()
	log.Logf(GreenColor, format, args)
}

func Yellow(format string, args ...interface{}) {
	log := New()
	log.Logf(YellowColor, format, args)
}

func Red(format string, args ...interface{}) {
	log := New()
	log.Logf(RedColor, format, args)
}
