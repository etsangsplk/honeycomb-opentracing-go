package honeycomb

import (
	libhoney "github.com/honeycombio/libhoney-go"
	lightstep "github.com/lightstep/lightstep-tracer-go"
)

var reservedTags = map[string]string{
	"traceID":       "tag.traceID",
	"id":            "tag.id",
	"operationName": "tag.operationName",
	"durationMs":    "tag.durationMs",
	"logs":          "tag.logs",
}

type HoneycombSpanRecorder struct {
	WriteKey string
	Dataset  string
	Router   RouterFunc
}

type RecorderOption func(*HoneycombSpanRecorder)

func WithWriteKey(writeKey string) func(*HoneycombSpanRecorder) {
	return func(hc *HoneycombSpanRecorder) {
		hc.WriteKey = writeKey
	}
}

func WithDataset(dataset string) func(*HoneycombSpanRecorder) {
	return func(hc *HoneycombSpanRecorder) {
		hc.Dataset = dataset
	}
}

func WithRouter(router RouterFunc) func(*HoneycombSpanRecorder) {
	return func(hc *HoneycombSpanRecorder) {
		hc.Router = router
	}
}

func NewSpanRecorder(opts ...RecorderOption) *HoneycombSpanRecorder {
	h := &HoneycombSpanRecorder{}
	for _, o := range opts {
		o(h)
	}
	libhoney.Init(libhoney.Config{WriteKey: h.WriteKey, Dataset: h.Dataset})
	return h
}

func (h *HoneycombSpanRecorder) RecordSpan(r lightstep.RawSpan) {
	event := libhoney.NewEvent()
	event.AddField("traceID", r.Context.TraceID)
	event.AddField("id", r.Context.SpanID)
	event.AddField("operationName", r.Operation)
	event.Timestamp = r.Start
	event.AddField("durationMs", r.Duration.Nanoseconds()/1e6)
	event.AddField("logs", r.Logs)
	addTagsToEvent(r, event)

	if h.Router != nil {
		routeInfo := h.Router(r)
		event.Dataset = routeInfo.Dataset
		event.WriteKey = routeInfo.WriteKey
		event.SampleRate = routeInfo.SampleRate
	}

	event.Send()
}

func (h *HoneycombSpanRecorder) Close() {
	libhoney.Close()
}

type RouteInfo struct {
	Dataset    string
	WriteKey   string
	SampleRate uint
}

type RouterFunc func(lightstep.RawSpan) RouteInfo

func addTagsToEvent(r lightstep.RawSpan, ev *libhoney.Event) {
	if r.Tags != nil {
		for k, v := range r.Tags {
			if reserved, ok := reservedTags[k]; ok {
				k = reserved
			}
			ev.AddField(k, v)
		}
	}
}
