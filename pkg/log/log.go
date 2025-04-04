package log

import (
	"github.com/sirupsen/logrus"
	"os"
)

func Init(configs Configs) {
	logrus.SetFormatter(&configs)
	logrus.SetOutput(os.Stdout)
}

// InfoWithFields creates an entry from the standard logger and adds multiple
// fields to it. This is simply a helper for `WithField`, invoking it
// once for each field.
//
// Note that it doesn't log until you call Debug, Print, Info, Warn, Fatal
// or Panic on the Entry it returns.
func InfoWithFields(fields map[string]interface{}, message string, args ...interface{}) {
	logrus.WithFields(fields).Infof(message, args...)
}

// ErrorWithFields creates an entry from the standard logger and adds multiple and return as error
// fields to it. This is simply a helper for `WithField`, invoking it
// once for each field.
//
// Note that it doesn't log until you call Debug, Print, Info, Warn, Fatal
// or Panic on the Entry it returns.
func ErrorWithFields(fields map[string]interface{}, message string, args ...interface{}) {
	logrus.WithFields(fields).Errorf(message, args...)
}

// Infof logs a message at level Info on the standard logger.
func Infof(format string, args ...interface{}) {
	logrus.Infof(format, args...)
}

// Fatalf logs a message at level Info on the standard logger.
func Fatalf(format string, args ...interface{}) {
	logrus.Fatalf(format, args...)
}

// Errorf logs a message at level Info on the standard logger.
func Errorf(format string, args ...interface{}) {
	logrus.Errorf(format, args...)
}

// Warnf logs a message at level Info on the standard logger.
func Warnf(format string, args ...interface{}) {
	logrus.Warnf(format, args...)
}
