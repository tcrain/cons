/*
github.com/tcrain/cons - Experimental project for testing and scaling consensus algorithms.
Copyright (C) 2020 The project authors - tcrain

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.

*/
/*
Basic logging functionality.
*/
package logging

import (
	"fmt"
	"log"

	// "github.com/golang/glog"

	"github.com/tcrain/cons/config"
)

// setup the logging flags
func init() {
	if config.LoggingType == config.GOLOG {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	}
}

// Printf logs args accoring to format.
func Printf(format string, args ...interface{}) {
	switch config.LoggingType {
	case config.GLOG:
		// glog.ErrorDepth(1, fmt.Sprintf(format, args))
		panic("no longer used")
	case config.GOLOG:
		err := log.Output(2, fmt.Sprintf(format, args...))
		if err != nil {
			panic(err)
		}
	case config.FMT:
		fmt.Printf(format+"\n", args...)
	default:
		panic("Invalid logging type")
	}
}

// Print logs args.
func Print(args ...interface{}) {
	switch config.LoggingType {
	case config.GLOG:
		// glog.ErrorDepth(1, args)
		panic("no longer used")
	case config.GOLOG:
		err := log.Output(2, fmt.Sprint(args...))
		if err != nil {
			panic(err)
		}
	case config.FMT:
		fmt.Println(args...)
	default:
		panic("Invalid logging type")
	}
}

// Errorf logs an error args using format.
func Errorf(format string, args ...interface{}) {
	if config.LoggingFmtLevel >= config.LOGERROR {
		switch config.LoggingType {
		case config.GLOG:
			// glog.ErrorDepth(1, fmt.Sprintf(format, args))
			panic("no longer used")
		case config.GOLOG:
			err := log.Output(2, fmt.Sprintf("ERR: "+format, args...))
			if err != nil {
				panic(err)
			}
		case config.FMT:
			fmt.Printf("ERR: "+format+"\n", args...)
		default:
			panic("Invalid logging type")
		}
	}
}

// Error logs an error args.
func Error(args ...interface{}) {
	if config.LoggingFmtLevel >= config.LOGERROR {
		switch config.LoggingType {
		case config.GLOG:
			// glog.ErrorDepth(1, args)
			panic("no longer used")
		case config.GOLOG:
			err := log.Output(2, fmt.Sprint("ERR: ", args))
			if err != nil {
				panic(err)
			}
		case config.FMT:
			fmt.Println("ERR: ", args)
		default:
			panic("Invalid logging type")
		}
	}
}

// Warningf logs a warning args using format.
func Warningf(format string, args ...interface{}) {
	if config.LoggingFmtLevel >= config.LOGWARNING {
		switch config.LoggingType {
		case config.GLOG:
			panic("no longer used")
			// glog.WarningDepth(1, fmt.Sprintf(format, args))
		case config.GOLOG:
			err := log.Output(2, fmt.Sprintf("WARN: "+format, args...))
			if err != nil {
				panic(err)
			}
		case config.FMT:
			fmt.Printf("WARN: "+format+"\n", args...)
		default:
			panic("Invalid logging type")
		}
	}
}

// Warning logs a warning args.
func Warning(args ...interface{}) {
	if config.LoggingFmtLevel >= config.LOGWARNING {
		switch config.LoggingType {
		case config.GLOG:
			panic("no longer used")
			// glog.WarningDepth(1, args)
		case config.GOLOG:
			err := log.Output(2, fmt.Sprint("WARN: ", args))
			if err != nil {
				panic(err)
			}
		case config.FMT:
			fmt.Println("WARN: ", args)
		default:
			panic("Invalid logging type")
		}
	}
}

// Infof logs an info message args using format.
func Infof(format string, args ...interface{}) {
	if config.LoggingFmtLevel >= config.LOGINFO {
		switch config.LoggingType {
		case config.GLOG:
			panic("no longer used")
			// glog.InfoDepth(1, fmt.Sprintf(format, args))
		case config.GOLOG:
			err := log.Output(2, fmt.Sprintf("INFO: "+format, args...))
			if err != nil {
				panic(err)
			}
		case config.FMT:
			fmt.Printf("INFO: "+format+"\n", args...)
		default:
			panic("Invalid logging type")
		}
	}
}

// Info logs an info message args.
func Info(args ...interface{}) {
	if config.LoggingFmtLevel >= config.LOGINFO {
		switch config.LoggingType {
		case config.GLOG:
			panic("no longer used")
			// glog.InfoDepth(1, args)
		case config.GOLOG:
			err := log.Output(2, fmt.Sprint("INFO: ", args))
			if err != nil {
				panic(err)
			}
		case config.FMT:
			fmt.Println("INFO: ", args)
		default:
			panic("Invalid logging type")
		}
	}
}
