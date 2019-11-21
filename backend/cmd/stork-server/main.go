package main

import (
	"os"
	"fmt"
	"path"
	"runtime"

	log "github.com/sirupsen/logrus"

	"isc.org/stork/server"
)

func main() {
	// Setup logging
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
	log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		TimestampFormat: "2006-01-02 15:04:05",
		//PadLevelText: true,
		// FieldMap: log.FieldMap{
		// 	FieldKeyTime:  "@timestamp",
		// 	FieldKeyLevel: "@level",
		// 	FieldKeyMsg:   "@message",
		// },
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			// Grab filename and line of current frame and add it to log entry
			_, filename := path.Split(f.File)
			return "", fmt.Sprintf("%20v:%-5d", filename, f.Line)
		},
	})


	// Initialize global state of Stork Server
	storkServer := server.NewStorkServer()
	defer storkServer.Shutdown()

	storkServer.Serve();
}
