package logger

import (
	"github.com/sirupsen/logrus"
)

// LogLevel type
type LogLevel int

const (
	// LogLevel [Info:Debug]
	Info LogLevel = iota
	Debug
)

// Logger struct defines the loggrus object instance
type Logger struct {
	log *logrus.Logger
}

// NewLogger function creates a new logger instance that defines the loggrus TextFormatter with [ForceColors:true]
func NewLogger() *Logger {
	log := logrus.New()

	log.SetFormatter(&logrus.TextFormatter{
		DisableColors:    false,
		ForceColors:      true,
		DisableTimestamp: true,
	})

	return &Logger{
		log: log,
	}
}

// SetLogLevel Logger
func (l *Logger) SetLogLevel(level LogLevel) {
	if level == Debug {
		l.log.Level = logrus.DebugLevel
	}
}

// Debug Logger
func (l *Logger) Debug(args ...interface{}) {
	l.log.Debug(args...)
}

// Debugf Logger
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log.Debugf(format, args...)
}

// Debugln Logger
func (l *Logger) Debugln(args ...interface{}) {
	l.log.Debugln(args...)
}

// Info Logger
func (l *Logger) Info(args ...interface{}) {
	l.log.Info(args...)
}

// Infof Logger
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log.Infof(format, args...)
}

// Infoln Logger
func (l *Logger) Infoln(args ...interface{}) {
	l.log.Infoln(args...)
}

// Warn Logger
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log.Warn(args...)
}

// Warnf Logger
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log.Warnf(format, args...)
}

// Warnln Logger
func (l *Logger) Warnln(format string, args ...interface{}) {
	l.log.Warnln(args...)
}

// Fatal Logger
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log.Fatal(args...)
}

// Fatalf Logger
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log.Fatalf(format, args...)
}

// Fatalln Logger
func (l *Logger) Fatalln(format string, args ...interface{}) {
	l.log.Fatalln(args...)
}

// Error Logger
func (l *Logger) Error(format string, args ...interface{}) {
	l.log.Error(args...)
}

// Errorf Logger
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log.Errorf(format, args...)
}

// Errorln Logger
func (l *Logger) Errorln(format string, args ...interface{}) {
	l.log.Errorln(args...)
}

// WithField Logger
func (l *Logger) WithField(key string, value interface{}) {
	l.log.WithField(key, value)
}

// WithFields Logger
func (l *Logger) WithFields(fields logrus.Fields) {
	l.log.WithFields(fields)
}
