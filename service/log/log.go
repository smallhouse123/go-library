package log

type Log interface {
	// write log to destination file
	WriteLog(logName string, requestEvent *RequestEvent)
	// close logger instance
	Close()
}

type UserEvent struct {
	EventType string                 `json:"eventType"`
	Metadata  map[string]interface{} `json:"metadata"`
	Count     int                    `json:"count"`
}

// make some fields to be pointer, so it's default value can be null
type RequestCommon struct {
	MicroTimestamp float64 `json:"microTimestamp"`
	VisitorId      string  `json:"visitorId"`
	IsNewVisitor   bool    `json:"isNewVisitor"`
	UserName       string  `json:"userName"`
	UserId         *string `json:"userId"`
}

type RequestEvent struct {
	RequestCommon *RequestCommon `json:"requestCommon"`
	UserEvents    []*UserEvent   `json:"userEvents"`
}
