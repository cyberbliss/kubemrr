package app

import (
	log "github.com/sirupsen/logrus"
	"os"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.WarnLevel)
	log.SetFormatter(&log.TextFormatter{})

}

func enableVerbose() {
	log.SetLevel(log.InfoLevel)
}
func enableVeryVerbose() {
	log.SetLevel(log.DebugLevel)
}

