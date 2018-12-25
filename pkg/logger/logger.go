package logger

import (
	config "github.com/justinbarrick/hone/pkg/job"
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/apex/log/handlers/json"
	"github.com/apex/log/handlers/multi"
	"github.com/fatih/color"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

type LogIOWriter struct {
	Logger func(string)
	buf    []byte
}

func (w *LogIOWriter) Write(b []byte) (int, error) {
	w.buf = append(w.buf, b...)

	splitLines := bytes.Split(w.buf, []byte{'\n',})

	numLines := len(splitLines)

	if numLines == 0 {
		return 0, nil
	} else if numLines > 1 {
		for _, line := range splitLines[:numLines - 1] {
			w.Logger(string(line))
		}
	}

	w.buf = splitLines[numLines - 1]
	return len(b), nil
}

type LogHandler struct {
	mu sync.Mutex
	LongestJob int
}

// HandleLog implements log.Handler.
func (h *LogHandler) HandleLog(e *log.Entry) error {
	names := e.Fields.Names()
	clr := cli.Colors[e.Level]

	h.mu.Lock()
	defer h.mu.Unlock()

	jobInt := e.Fields.Get("job")
	job := ""
	if jobInt != nil {
		job = jobInt.(string)
		if h.LongestJob < len(job) {
			h.LongestJob = len(job)
		}
	}

	success := e.Fields.Get("success")
	pipeChar := "|"
	if e.Level == log.ErrorLevel {
		pipeChar = "✗"
		clr = color.New(color.FgHiRed)
	} else if success != nil && success.(bool) {
		pipeChar = "✓"
		clr = color.New(color.FgHiGreen)
	}

	msg := clr.Sprintf("%-*s %s ", h.LongestJob, job, pipeChar)
	msg += fmt.Sprintf("%-60s", e.Message)

	for _, name := range names {
		if name == "source" || name == "job" || name == "success" || name == "stdout" || name == "stderr" {
			continue
		}

		msg += fmt.Sprintf(" %s=%v", clr.Sprint(name), e.Fields.Get(name))
	}

	fmt.Fprintln(os.Stderr, msg)

	return nil
}

var logger = &log.Logger{}

func InitLogger(longestJob int, remoteLog io.WriteCloser) {
	handler := multi.New(&LogHandler{
		LongestJob: longestJob,
	})

	if remoteLog != nil {
		handler.Handlers = append(handler.Handlers, json.New(remoteLog))
	}

	logger = &log.Logger{
		Handler: handler,
		Level: log.DebugLevel,
	}
}
func LogWriter(job *config.Job) io.Writer {
	return &LogIOWriter{
		Logger: logger.WithFields(log.Fields{
			"job": job.GetName(),
			"stdout": true,
		}).Info,
	}
}

func LogWriterError(job *config.Job) io.Writer {
	return &LogIOWriter{
		Logger: logger.WithFields(log.Fields{
			"job": job.GetName(),
			"stderr": true,
		}).Warn,
	}
}

func Printf(message string, args ...interface{}) {
	logger.Infof(message, args...)
}

func Errorf(message string, args ...interface{}) {
	logger.Errorf(message, args...)
}

func Successf(message string, args ...interface{}) {
	logger.WithFields(log.Fields{
		"success": true,
	}).Infof(message, args...)
}

func LoggerForJob(job *config.Job) *log.Entry {
	return logger.WithFields(log.Fields{
		"job": job.GetName(),
	})
}

func Log(job *config.Job, message string) {
	LoggerForJob(job).Info(strings.TrimSpace(message))
}

func LogError(job *config.Job, message string) {
	LoggerForJob(job).Error(strings.TrimSpace(message))
}

func LogDebug(job *config.Job, message string) {
	LoggerForJob(job).Debug(strings.TrimSpace(message))
}

func LogSuccess(job *config.Job, message string) {
	LoggerForJob(job).WithFields(log.Fields{
		"success": true,
	}).Info(strings.TrimSpace(message))
}

func LogJob(callback func(*config.Job) error) func(*config.Job) error {
	return func(job *config.Job) error {
		Log(job, fmt.Sprintf("Running job \"%s\".", job.GetName()))
		err := callback(job)
		if err != nil {
			LogError(job, fmt.Sprintf("Job \"%s\" errored: %s.", job.GetName(), err))
		} else {
			LogSuccess(job, fmt.Sprintf("Job \"%s\" completed!", job.GetName()))
		}
		return err
	}
}
