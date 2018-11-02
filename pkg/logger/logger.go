package logger

import (
	"io"
	"fmt"
	"log"
	"os"
	"strings"
	"github.com/justinbarrick/farm/pkg/config"
	"github.com/kvz/logstreamer"
)

func LogWriter(job config.Job) io.Writer {
	logger := log.New(os.Stderr, "", log.Ldate|log.Ltime)
	return logstreamer.NewLogstreamer(logger, fmt.Sprintf(" == %s => ", job.Name), false)
}

func LogWriterError(job config.Job) io.Writer {
	logger := log.New(os.Stderr, "", log.Ldate|log.Ltime)
	return logstreamer.NewLogstreamer(logger, fmt.Sprintf(" !! %s => ", job.Name), false)
}

func Log(job config.Job, message string) {
	log.Printf(" == %s => %s\n", job.Name, strings.TrimSpace(message))
}

func LogJob(callback func (config.Job) error) func (config.Job) error {
	return func (job config.Job) error {
		log.Printf("======> Running job \"%s\".\n", job.Name)
		err := callback(job)
		if err != nil {
			log.Printf("======> Job \"%s\" errored: %s.\n", job.Name, err)
		} else {
			log.Printf("======> Job \"%s\" completed!\n", job.Name)
		}
		return err
	}
}
