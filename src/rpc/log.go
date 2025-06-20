package rpc

import (
	"encoding/json"
	"os"

	"github.com/illikainen/go-utils/src/logging"
	log "github.com/sirupsen/logrus"
)

type SanitizedJSONFormatter struct {
}

var hostname string

func (f *SanitizedJSONFormatter) Format(e *log.Entry) ([]byte, error) {
	if hostname == "" {
		h, err := os.Hostname()
		if err != nil {
			return nil, err
		}

		hostname = h
	}
	entry := *e
	entry.Message = hostname + ": " + entry.Message

	formatter := logging.SanitizedJSONFormatter{}
	out, err := formatter.Format(&entry)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(Log{
		Type:   LogType,
		Fields: string(out),
	})
	if err != nil {
		return nil, err
	}

	return append(data, '\n'), nil
}
