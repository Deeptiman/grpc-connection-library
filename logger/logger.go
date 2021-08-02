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

// Logger SetLogLevel
func (l *Logger) SetLogLevel(level LogLevel) {
	if level == Debug {
		l.log.Level = logrus.DebugLevel
	}
}

// Logger Debug
func (l *Logger) Debug(args ...interface{}) {
	l.log.Debug(args...)
}

// Logger Debugf
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log.Debugf(format, args...)
}

// Logger Debugln
func (l *Logger) Debugln(args ...interface{}) {
	l.log.Debugln(args...)
}

// Logger Info
func (l *Logger) Info(args ...interface{}) {
	l.log.Info(args...)
}

// Logger Infof
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log.Infof(format, args...)
}

// Logger Infoln
func (l *Logger) Infoln(args ...interface{}) {
	l.log.Infoln(args...)
}

// Logger Warn
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log.Warn(args...)
}

// Logger Warnf
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log.Warnf(format, args...)
}

// Logger Warnln
func (l *Logger) Warnln(format string, args ...interface{}) {
	l.log.Warnln(args...)
}

// Logger Fatal
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log.Fatal(args...)
}

// Logger Fatalf
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log.Fatalf(format, args...)
}

// Logger Fatalln
func (l *Logger) Fatalln(format string, args ...interface{}) {
	l.log.Fatalln(args...)
}

// Logger Error
func (l *Logger) Error(format string, args ...interface{}) {
	l.log.Error(args...)
}

// Logger Errorf
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log.Errorf(format, args...)
}

// Logger Errorln
func (l *Logger) Errorln(format string, args ...interface{}) {
	l.log.Errorln(args...)
}

// Logger WithField
func (l *Logger) WithField(key string, value interface{}) {
	l.log.WithField(key, value)
}

// Logger WithFields
func (l *Logger) WithFields(fields logrus.Fields) {
	l.log.WithFields(fields)
}
