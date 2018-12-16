package logger

import (
	"fmt"
	config "github.com/justinbarrick/hone/pkg/job"
	"github.com/kvz/logstreamer"
	"io"
	"log"
	"os"
	"strings"
)

var logger = log.New(os.Stderr, "", log.Ltime)

func LogWriter(job *config.Job) io.Writer {
	return logstreamer.NewLogstreamer(logger, fmt.Sprintf(" == %s => ", job.GetName()), false)
}

func LogWriterError(job *config.Job) io.Writer {
	return logstreamer.NewLogstreamer(logger, fmt.Sprintf(" !! %s => ", job.GetName()), false)
}

func Printf(message string, args ...interface{}) {
	logger.Printf(message, args...)
}

func Log(job *config.Job, message string) {
	logger.Printf(" == %s => %s\n", job.GetName(), strings.TrimSpace(message))
}

func LogJob(callback func(*config.Job) error) func(*config.Job) error {
	return func(job *config.Job) error {
		logger.Printf("======> Running job \"%s\".\n", job.GetName())
		err := callback(job)
		if err != nil {
			logger.Printf("======> Job \"%s\" errored: %s.\n", job.GetName(), err)
		} else {
			logger.Printf("======> Job \"%s\" completed!\n", job.GetName())
		}
		return err
	}
}
