package logger

import (
	"log"
	"os"
)

// Logger é a interface para logging
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
}

// SimpleLogger é uma implementação simples de Logger
type SimpleLogger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	warnLogger  *log.Logger
}

// NewLogger cria uma nova instância de Logger
func NewLogger() Logger {
	return &SimpleLogger{
		infoLogger:  log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile),
		errorLogger: log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile),
		debugLogger: log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile),
		warnLogger:  log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

// Info registra uma mensagem de informação
func (l *SimpleLogger) Info(msg string, keysAndValues ...interface{}) {
	l.infoLogger.Printf(msg+" %v", keysAndValues...)
}

// Error registra uma mensagem de erro
func (l *SimpleLogger) Error(msg string, keysAndValues ...interface{}) {
	l.errorLogger.Printf(msg+" %v", keysAndValues...)
}

// Debug registra uma mensagem de debug
func (l *SimpleLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.debugLogger.Printf(msg+" %v", keysAndValues...)
}

// Warn registra uma mensagem de aviso
func (l *SimpleLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.warnLogger.Printf(msg+" %v", keysAndValues...)
}
