package logger

import (
	"fmt"
	"log"
	"strings"

	"gopkg.in/BurntSushi/toml.v0"
	"gopkg.in/natefinch/lumberjack.v2"
)

type logLevel int

const (
	l_ERROR logLevel = iota
	l_INFO
	l_DEBUG
)

func init() {
	type configFile struct {
		LogPath string `toml:"log_path"`
	}
	cfg := &configFile{}
	if _, err := toml.DecodeFile("./conf.toml", &cfg); err != nil {
		fmt.Printf("errors is %v\n", err)
	}
	log.SetOutput(&lumberjack.Logger{
		Filename:   cfg.LogPath,
		MaxSize:    20, // megabytes
		MaxBackups: 50,
		MaxAge:     30, //days
		LocalTime:  true,
	})
}

var currentLogLevel = l_INFO

func UpdateLoggingLevel(levelString string) {
	currentLogLevel = strToLevel(levelString)
}

func Info(msg string) {
	doLogging(msg, l_INFO, "INFO")
}

func Infof(msg string, params ...interface{}) {
	doLogging(fmt.Sprintf(msg, params), l_INFO, "INFO")
}

func Debug(msg string) {
	doLogging(msg, l_DEBUG, "DEBUG")
}

func Debugf(msg string, params ...interface{}) {
	doLogging(fmt.Sprintf(msg, params...), l_DEBUG, "DEBUG")
}

func ErrorMsg(msg string) {
	doLogging(msg, l_ERROR, "ERROR")
}

func ErrorMsgf(msg string, params ...interface{}) {
	doLogging(fmt.Sprintf(msg, params...), l_ERROR, "ERROR")
}

func ErrorWithMsg(msg string, err error) {
	var errMsg = fmt.Sprintf("%s : %s", msg, err.Error())
	doLogging(errMsg, l_ERROR, "ERROR")
}

func Error(err error) {
	doLogging(err, l_ERROR, "ERROR")
}

func doLogging(v interface{}, level logLevel, levelName string) {
	if currentLogLevel >= level {
		fmt.Printf("%v\n", v)
		//cant use println here since it doesn't return err msg
		err := log.Output(1, fmt.Sprintln(levelName, v))
		if err != nil {
			fmt.Println(err)
		}

	}
}

func strToLevel(levelString string) logLevel {
	switch strings.ToUpper(levelString) {
	case "INFO":
		return l_INFO
	case "DEBUG":
		return l_DEBUG
	case "ERROR":
		return l_ERROR
	default:
		ErrorMsg("Unknown logging level. Stopping DripStat Infrastructure Agent")
		panic("Unknown logging level")
	}
}
