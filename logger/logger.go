package logger

import (
	"io"
	"log"
	"os"

	kitlog "github.com/go-kit/log"
)

var logger kitlog.Logger

func Init() {
	var (
		logout io.Writer
		err    error
	)

	if true {
		logout = os.Stderr
	} else {
		logpath := os.Getenv("SENDIT_LOG")
		if _, err := os.Stat(logpath); err != nil {
			_, err := os.Create(logpath)
			if err != nil {
				log.Fatal(err)
			}
		}
		logout, err = os.OpenFile(logpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal(err)
		}
	}
	logger = kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(logout))
	logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC, "caller", kitlog.Caller(4))
}

func Log(args ...any) error {
	return logger.Log(args...)
}
