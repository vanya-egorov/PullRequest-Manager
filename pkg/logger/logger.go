package logger

import (
	"log"
	"os"
)

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

type SimpleLogger struct {
	infoLog  *log.Logger
	errorLog *log.Logger
	debugLog *log.Logger
}

func New() Logger {
	return &SimpleLogger{
		infoLog:  log.New(os.Stdout, "[INFO] ", log.LstdFlags|log.Lshortfile),
		errorLog: log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lshortfile),
		debugLog: log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lshortfile),
	}
}

func (l *SimpleLogger) Info(msg string, args ...any) {
	if len(args) > 0 {
		l.infoLog.Printf(msg+" %v", args...)
	} else {
		l.infoLog.Println(msg)
	}
}

func (l *SimpleLogger) Error(msg string, args ...any) {
	if len(args) > 0 {
		l.errorLog.Printf(msg+" %v", args...)
	} else {
		l.errorLog.Println(msg)
	}
}

func (l *SimpleLogger) Debug(msg string, args ...any) {
	if len(args) > 0 {
		l.debugLog.Printf(msg+" %v", args...)
	} else {
		l.debugLog.Println(msg)
	}
}
