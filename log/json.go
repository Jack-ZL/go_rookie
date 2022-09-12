package log

import (
	"encoding/json"
	"fmt"
	"time"
)

type JsonFormatter struct {
	TimeDisplay bool
}

func (f *JsonFormatter) Format(params *LoggingFormatterParams) string {
	if params.LoggerFields == nil {
		params.LoggerFields = make(Fields)
	}
	if f.TimeDisplay {
		now := time.Now()
		params.LoggerFields["log_time"] = now.Format("2006/01/02 - 15:04:05")
	}

	params.LoggerFields["msg"] = params.Msg
	logStr, err := json.Marshal(params.LoggerFields)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s", logStr)
}
