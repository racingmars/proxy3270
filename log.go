/*
 * Copyright 2020-2021 by Matthew R. Wilson <mwilson@mattwilson.org>
 *
 * This file is part of proxy3270.
 *
 * proxy3270 is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * proxy3270 is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with proxy3270. If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"io"
	"log"
)

type LogLevel int

const (
	TraceLvl LogLevel = iota
	DebugLvl
	InfoLvl
	WarnLvl
	ErrorLvl
	FatalLvl
)

type Logger struct {
	Level                                         LogLevel
	trace, debug, info, warn, err, fatal, unknown *log.Logger
}

func InitLogger(level LogLevel, writer io.Writer) *Logger {
	loggerFlags := log.Ldate | log.Ltime
	l := new(Logger)
	l.Level = level
	l.trace = log.New(writer, "TRC ", loggerFlags)
	l.debug = log.New(writer, "DGB ", loggerFlags)
	l.info = log.New(writer, "INF ", loggerFlags)
	l.warn = log.New(writer, "WRN ", loggerFlags)
	l.err = log.New(writer, "ERR ", loggerFlags)
	l.fatal = log.New(writer, "FTL ", loggerFlags)
	l.unknown = log.New(writer, "UNK ", loggerFlags)
	return l
}

func (l *Logger) Log(level LogLevel, format string, v ...interface{}) {
	logger := l.getLoggerForLevel(level)
	if logger != nil {
		logger.Printf(format, v...)
	}
}

func (l *Logger) LogWithErr(level LogLevel, err error, format string,
	v ...interface{}) {
	logger := l.getLoggerForLevel(level)
	if logger != nil {
		params := []interface{}{err}
		params = append(params, v...)
		logger.Printf("[%v] "+format, params...)
	}
}

func (l *Logger) getLoggerForLevel(level LogLevel) *log.Logger {
	if level < l.Level {
		return nil
	}

	switch level {
	case TraceLvl:
		return l.trace
	case DebugLvl:
		return l.debug
	case InfoLvl:
		return l.info
	case WarnLvl:
		return l.warn
	case ErrorLvl:
		return l.err
	case FatalLvl:
		return l.fatal
	default:
		// umm... how'd this happen?
		return l.unknown
	}
}
