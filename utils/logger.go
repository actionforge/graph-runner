package utils

import (
	"io"
	"log"
	"os"
	"strings"
)

var (
	LoggerBase   *log.Logger
	LoggerDebug  *log.Logger
	LoggerString *StringLogger
	cw           *conditionalWriter
)

const (
	LogGhStartGroup = "##[group]"
	LogGhEndGroup   = "##[endgroup]"
)

type StringLogger struct {
	builder strings.Builder
}

func (l *StringLogger) Write(p []byte) (n int, err error) {
	return l.builder.Write(p)
}

func (l *StringLogger) Clear() {
	l.builder.Reset()
}

func (l *StringLogger) String() string {
	return l.builder.String()
}

type conditionalWriter struct {
	stdWriter io.Writer
	strLogger *StringLogger

	logToString bool
	logToStdout bool
}

func (cw *conditionalWriter) Write(p []byte) (n int, err error) {
	if cw.strLogger != nil && cw.logToString {
		_, _ = cw.strLogger.Write(p)
	}

	if cw.stdWriter != nil && cw.logToStdout {
		_, _ = cw.stdWriter.Write(p)
	}
	return len(p), nil
}

func init() {
	LoggerString = &StringLogger{}
	cw = &conditionalWriter{
		stdWriter:   os.Stdout,
		strLogger:   LoggerString,
		logToString: false,
		logToStdout: true,
	}

	LoggerBase = log.New(cw, "", 0)
	LoggerDebug = log.New(cw, "DEBUG: ", log.Llongfile|log.LstdFlags)
}

func EnableStringLogging(flag bool) {
	cw.logToString = flag
}
