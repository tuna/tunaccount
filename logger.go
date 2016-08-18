package main

import (
	"os"

	"gopkg.in/op/go-logging.v1"
)

var logger = logging.MustGetLogger("tunaccount")

func initLogger(verbose, debug, withSystemd bool) {
	var fmtString string
	if withSystemd {
		fmtString = "[%{level:.6s}] %{message}"
	} else {
		if debug {
			fmtString = "%{color}[%{time:06-01-02 15:04:05}][%{level:.6s}][%{shortfile}]%{color:reset} %{message}"
		} else {
			fmtString = "%{color}[%{time:06-01-02 15:04:05}][%{level:.6s}]%{color:reset} %{message}"
		}
	}
	format := logging.MustStringFormatter(fmtString)
	logging.SetFormatter(format)
	logging.SetBackend(logging.NewLogBackend(os.Stdout, "", 0))

	if debug {
		logging.SetLevel(logging.DEBUG, "tunaccount")
	} else if verbose {
		logging.SetLevel(logging.INFO, "tunaccount")
	} else {
		logging.SetLevel(logging.NOTICE, "tunaccount")
	}
}
