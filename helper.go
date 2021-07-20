package main

import log "github.com/sirupsen/logrus"

func setLogLevel(l string) {
	switch l {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}
