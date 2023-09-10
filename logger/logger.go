package logger

import (
	"fmt"
	"log"
	"runtime"
	// "strings"
	"sync"
)

// MyLogger represents the custom logger.
type MyLogger struct {
	logger *log.Logger
}

var (
	once      sync.Once
	singleton *MyLogger
)

// New creates a new instance of MyLogger.
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
func (ml *MyLogger) Logf(format string, args ...interface{}) {
	// Get the function name of the caller
	pc, _, _, _ := runtime.Caller(2)
	callerFunc := runtime.FuncForPC(pc).Name()

	// // Extract only the function name (without package)
	// parts := strings.Split(callerFunc, ".")
	// funcName := parts[len(parts)-1]
	funcName := callerFunc

	// Format and log the message
	message := fmt.Sprintf("[%s] %s", funcName, format)
	ml.logger.Printf(message, args...)
}

func Logf(format string, args ...interface{}) {
	log := New()
	log.Logf(format, args)
}
