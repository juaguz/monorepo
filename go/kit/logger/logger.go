package logger

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
)

var myLog *logrus.Logger

func init() {
	myLog = logrus.New()
	myLog.SetOutput(os.Stdout)
	myLog.SetLevel(logrus.DebugLevel)

	//Shows file and line where the log is called
	myLog.SetReportCaller(true)
}

func GetLogger() *logrus.Logger {
	return myLog
}

func GetLoggerFromContext(ctx context.Context) *logrus.Entry {
	logger := ctx.Value("logger")

	//if we dont have a logger we create a new one to cascade down
	if logger == nil {
		ctx = context.WithValue(ctx, "logger", &logrus.Entry{})
		logger = ctx.Value("logger")
	}

	return logger.(*logrus.Entry)
}

func SetFuncName(log *logrus.Entry, funcName string) *logrus.Entry {

	return log.WithField("funcName", funcName)

}

func GetLoggerWithFuncName(ctx context.Context, funcName string) *logrus.Entry {
	logger := GetLoggerFromContext(ctx)
	return SetFuncName(logger, funcName)
}
