package eventlogger

import (
	log "github.com/Sirupsen/logrus"

	"github.com/weaveworks/flux/balancer/events"
)

type EventLogger struct {
	events.DiscardOthers
}

func (EventLogger) Connection(ev *events.Connection) {
	log.Infoln("Connection", ev.Inbound, ev.InstanceAddr)
}

func (EventLogger) HttpExchange(ev *events.HttpExchange) {
	log.Infoln("Http exchange", ev.Inbound, ev.InstanceAddr,
		ev.Request.Method, ev.Request.URL, ev.Response.StatusCode,
		ev.RoundTrip, ev.TotalTime)
}
