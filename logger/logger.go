// Package logger provides logging mechanism including replaceable Logger interface and its default implementation.
//
// Developers may replace default implementation and output level with her desired Logger implementation in a thread-safe manner as below:
//
//   type MyLogger struct {}
//
//   // MyLogger implements Logger
//   var _ Logger = (*MyLogger)(nil)
//
// 	 func (*MyLogger) Debug(args ...interface{}) {}
//
// 	 func (*MyLogger) Debugf(format string, args ...interface{}) {}
//
// 	 func (*MyLogger) Info(args ...interface{}) {}
//
// 	 func (*MyLogger) Infof(format string, args ...interface{}) {}
//
// 	 func (*MyLogger) Warn(args ...interface{}) {}
//
//   func (*MyLogger) Warnf(format string, args ...interface{}) {}
//
//   func (*MyLogger) Error(args ...interface{}) {}
//
// 	 func (*MyLogger) Errorf(format string, args ...interface{}) {}
//
// 	 l := &MyLogger{}
//
// 	 // Setter methods are thread-safe
// 	 logger.SetLogger(l)
// 	 logger.SetOutputLevel(logger.InfoLevel)
//
// 	 logger.Info("Output via new Logger impl.")
//
package logger

import (
	"fmt"
	"log"
	"os"
	"sync"
)

// Level indicates what logging level the output is representing.
// This typically indicates the severity of a particular logging event.
type Level uint

var (
	outputLevel = DebugLevel
	lgr         = NewWithStandardLogger(log.New(os.Stdout, "", log.LstdFlags|log.Llongfile))

	// mutex avoids race condition caused by concurrent call to SetLogger(), SetOutputLevel() and logging method.
	mutex sync.RWMutex
)

const (
	// ErrorLevel indicates the error state of an events. This must be noted and be fixed.
	// In a practical situation, fix may include lowering the corresponding event's log level.
	ErrorLevel Level = iota

	// WarnLevel represents those events that are not critical but deserve to note.
	// An event with this level may not necessarily be considered as an error;
	// however, its frequent occurrence deserves the developer's attention and may be subject to bug-fix.
	WarnLevel

	// InfoLevel is used to inform what is happening inside the application.
	InfoLevel

	// DebugLevel indicates the output is logged for debugging purposes.
	// This level is not suitable for production usage.
	DebugLevel
)

// String returns the stringified representation of the logging level.
func (level Level) String() string {
	switch level {
	case DebugLevel:
		return "DEBUG"

	case InfoLevel:
		return "INFO"

	case WarnLevel:
		return "WARN"

	case ErrorLevel:
		return "ERROR"
	}

	return "UNKNOWN"
}

// Logger defines the interface that can be used as a logging tool. A developer may provide a customized logger via
// SetLogger() to modify behavior. By default, the instance of defaultLogger is set and is ready to use.
type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})

	Info(args ...interface{})
	Infof(format string, args ...interface{})

	Warn(args ...interface{})
	Warnf(format string, args ...interface{})

	Error(args ...interface{})
	Errorf(format string, args ...interface{})
}

type defaultLogger struct {
	logger *log.Logger
}

var _ Logger = (*defaultLogger)(nil)

func (d *defaultLogger) Debug(args ...interface{}) {
	d.out(DebugLevel, args...)
}

func (d *defaultLogger) Debugf(format string, args ...interface{}) {
	d.outf(DebugLevel, format, args...)
}

func (d *defaultLogger) Info(args ...interface{}) {
	d.out(InfoLevel, args...)
}

func (d *defaultLogger) Infof(format string, args ...interface{}) {
	d.outf(InfoLevel, format, args...)
}

func (d *defaultLogger) Warn(args ...interface{}) {
	d.out(WarnLevel, args...)
}

func (d *defaultLogger) Warnf(format string, args ...interface{}) {
	d.outf(WarnLevel, format, args...)
}
func (d *defaultLogger) Error(args ...interface{}) {
	d.out(ErrorLevel, args...)
}

func (d *defaultLogger) Errorf(format string, args ...interface{}) {
	d.outf(ErrorLevel, format, args...)
}

func (d *defaultLogger) out(level Level, args ...interface{}) {
	// combine level identifier and given arguments for variadic function call
	leveledArgs := append([]interface{}{"[" + level.String() + "]"}, args...)
	_ = d.logger.Output(4, fmt.Sprintln(leveledArgs...))
}

func (d *defaultLogger) outf(level Level, format string, args ...interface{}) {
	// combine level identifier and given arguments for variadic function call
	leveledArgs := append([]interface{}{level}, args...)
	_ = d.logger.Output(4, fmt.Sprintf("[%s] "+format, leveledArgs...))
}

func newDefaultLogger() Logger {
	return NewWithStandardLogger(log.New(os.Stdout, "", log.LstdFlags|log.Llongfile))
}

// NewWithStandardLogger creates an instance of defaultLogger with Go's standard log.Logger. This can be used when
// implementing Logger interface is too much of a task but still, a bit of modification to defaultLogger is required.
//
// Returning Logger can be fed to SetLogger() to replace the old defaultLogger.
func NewWithStandardLogger(l *log.Logger) Logger {
	return &defaultLogger{
		logger: l,
	}
}

// SetLogger receives a Logger implementation and sets this as a logger. From this call forward, any call to logging
// method proxies arguments to the corresponding logging method of the given Logger.
// e.g. call to logger.Info() points to Logger.Info().
//
// This method is "thread-safe."
func SetLogger(l Logger) {
	mutex.Lock()
	defer mutex.Unlock()
	lgr = l
}

// GetLogger returns currently set Logger. Once a preferred Logger implementation is set via SetLogger(),
// the developer may use its method by calling this package's public function: Debug(), Debugf(), and others.
//
// However, when a developer wishes to retrieve the underlying Logger instance, this function helps. This is particularly
// useful in such a situation where Logger implementation must be temporarily switched but must be switched back when a
// task is done. Example follows.
//
//	import (
//		"github.com/oklahomer/go-kasumi/logger"
//		"io/ioutil"
//		"log"
//		"os"
//		"testing"
//	)
//
//	func TestMain(m *testing.M) {
//		oldLogger := logger.GetLogger()
//		defer logger.SetLogger(oldLogger)
//
//		l := log.New(ioutil.Discard, "dummyLog", 0)
// 		newLogger := logger.NewWithStandardLogger(l)
//		logger.SetLogger(newLogger)
//
//		code := m.Run()
//
//		os.Exit(code)
//	}
func GetLogger() Logger {
	mutex.RLock()
	defer mutex.RUnlock()
	return lgr
}

// SetOutputLevel sets what log level to output. The application may call the logging method any time, but Logger only
// outputs when the corresponding log level is equal to or higher than the level set here.
// e.g. When InfoLevel is set, output with Debug() and Debugf() are ignored.
//
// This method is "thread-safe."
//
// DebugLevel is set by default, so this should be explicitly overridden with a higher logging level in the production
// environment to avoid printing undesired sensitive data.
func SetOutputLevel(level Level) {
	mutex.Lock()
	defer mutex.Unlock()
	outputLevel = level
}

// Debug outputs given arguments via pre-set Logger implementation. The logging level must be left with the default
// setting or be set to DebugLevel with SetOutputLevel().
func Debug(args ...interface{}) {
	mutex.RLock()
	defer mutex.RUnlock()
	if outputLevel >= DebugLevel {
		lgr.Debug(args...)
	}
}

// Debugf outputs given arguments with format via pre-set Logger implementation. The logging level must be left with the
// default setting or be set to DebugLevel with SetOutputLevel().
func Debugf(format string, args ...interface{}) {
	mutex.RLock()
	defer mutex.RUnlock()
	if outputLevel >= DebugLevel {
		lgr.Debugf(format, args...)
	}
}

// Info outputs given arguments via pre-set Logger implementation. Logging level must be set to DebugLevel or InfoLevel
// with SetOutputLevel().
func Info(args ...interface{}) {
	mutex.RLock()
	defer mutex.RUnlock()
	if outputLevel >= InfoLevel {
		lgr.Info(args...)
	}
}

// Infof outputs given arguments with format via pre-set Logger implementation. Logging level must be set to DebugLevel
// or InfoLevel with SetOutputLevel().
func Infof(format string, args ...interface{}) {
	mutex.RLock()
	defer mutex.RUnlock()
	if outputLevel >= InfoLevel {
		lgr.Infof(format, args...)
	}
}

// Warn outputs given arguments via pre-set Logger implementation. Logging level must be set to DebugLevel, InfoLevel or
// WarnLevel with SetOutputLevel().
func Warn(args ...interface{}) {
	mutex.RLock()
	defer mutex.RUnlock()
	if outputLevel >= WarnLevel {
		lgr.Warn(args...)
	}
}

// Warnf outputs given arguments with format via pre-set Logger implementation. Logging level must be set to DebugLevel,
// InfoLevel or WarnLevel with SetOutputLevel().
func Warnf(format string, args ...interface{}) {
	mutex.RLock()
	defer mutex.RUnlock()
	if outputLevel >= WarnLevel {
		lgr.Warnf(format, args...)
	}
}

// Error outputs given arguments via pre-set Logger implementation.
func Error(args ...interface{}) {
	mutex.RLock()
	defer mutex.RUnlock()
	if outputLevel >= ErrorLevel {
		lgr.Error(args...)
	}
}

// Errorf outputs given arguments with format via pre-set Logger implementation.
func Errorf(format string, args ...interface{}) {
	mutex.RLock()
	defer mutex.RUnlock()
	if outputLevel >= ErrorLevel {
		lgr.Errorf(format, args...)
	}
}
