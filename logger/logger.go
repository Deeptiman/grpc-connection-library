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

// SetLogLevel
func (l *Logger) SetLogLevel(level LogLevel) {
	if level == Debug {
		l.log.Level = logrus.DebugLevel
	}
}

// Debug
func (l *Logger) Debug(args ...interface{}) {
	l.log.Debug(args...)
}

// Debugf
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log.Debugf(format, args...)
}

// Debugln
func (l *Logger) Debugln(args ...interface{}) {
	l.log.Debugln(args...)
}

// Info
func (l *Logger) Info(args ...interface{}) {
	l.log.Info(args...)
}

// Infof
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log.Infof(format, args...)
}

// Infoln
func (l *Logger) Infoln(args ...interface{}) {
	l.log.Infoln(args...)
}

// Warn
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log.Warn(args...)
}

// Warnf
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log.Warnf(format, args...)
}

// Warnln
func (l *Logger) Warnln(format string, args ...interface{}) {
	l.log.Warnln(args...)
}

// Fatal
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log.Fatal(args...)
}

// Fatalf
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log.Fatalf(format, args...)
}

// Fatalln
func (l *Logger) Fatalln(format string, args ...interface{}) {
	l.log.Fatalln(args...)
}

// Error
func (l *Logger) Error(format string, args ...interface{}) {
	l.log.Error(args...)
}

// Errorf
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log.Errorf(format, args...)
}

// Errorln
func (l *Logger) Errorln(format string, args ...interface{}) {
	l.log.Errorln(args...)
}

// WithField
func (l *Logger) WithField(key string, value interface{}) {
	l.log.WithField(key, value)
}

// WithFields
func (l *Logger) WithFields(fields logrus.Fields) {
	l.log.WithFields(fields)
}
