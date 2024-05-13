package logger

import (
	"github.com/charmbracelet/log"
	"os"
)

var Logger = log.NewWithOptions(os.Stderr, log.Options{
	ReportCaller:    true,
	ReportTimestamp: true,
	Prefix:          "Violet",
})
