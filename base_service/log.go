package base_service

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"strings"
	"time"
)

var GlobalLogger *zerolog.Logger
var LogFile *os.File
var LogLevel = zerolog.InfoLevel

func CreateLog() {
	if GlobalLogger != nil {
		return
	}
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(LogLevel)
	file, err := os.OpenFile(fmt.Sprintf("log-%s.log", time.Now().Format(time.DateOnly)), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.DateTime, NoColor: false}
	fileWriter := zerolog.ConsoleWriter{Out: file, TimeFormat: time.DateTime, NoColor: true}
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	}
	fileWriter.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	}
	log.Logger = zerolog.New(zerolog.MultiLevelWriter(output, fileWriter)).With().Timestamp().Logger()
	GlobalLogger = &log.Logger
	LogFile = file
}

func GetLogger(module string) zerolog.Logger {
	if GlobalLogger == nil {
		CreateLog()
	}
	return GlobalLogger.With().Str("module", module).Logger()
}
