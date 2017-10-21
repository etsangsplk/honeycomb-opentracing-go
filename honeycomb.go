// This package contains convenience helpers for sending Opentracing data
// collected by the Lightstep tracer
// (https://github.com/lightstep/lightstep-tracer-go) to Honeycomb
// (https://honeycomb.io/).
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

// HoneycombSpanRecorder implements the lightstep.SpanRecorder interface by
// forwarding span data to Honeycomb.
type HoneycombSpanRecorder struct {
	WriteKey string
	Dataset  string
	Router   RouterFunc

	builder *libhoney.Builder
}

type RecorderOption func(*HoneycombSpanRecorder)

// WithWriteKey sets the default write key used to send span data to Honeycomb.
func WithWriteKey(writeKey string) func(*HoneycombSpanRecorder) {
	return func(hc *HoneycombSpanRecorder) {
		hc.WriteKey = writeKey
	}
}

// WithDataset sets the default destination dataset for span data
func WithDataset(dataset string) func(*HoneycombSpanRecorder) {
	return func(hc *HoneycombSpanRecorder) {
		hc.Dataset = dataset
	}
}

// WithRouter sets an optional function to control span routing and sampling.
func WithRouter(router RouterFunc) func(*HoneycombSpanRecorder) {
	return func(hc *HoneycombSpanRecorder) {
		hc.Router = router
	}
}

// NewSpanRecorder creates a new HoneycombSpanRecorder.
func NewSpanRecorder(opts ...RecorderOption) *HoneycombSpanRecorder {
	h := &HoneycombSpanRecorder{}
	for _, o := range opts {
		o(h)
	}

	h.builder = libhoney.NewBuilder()
	h.builder.WriteKey = h.WriteKey
	h.builder.Dataset = h.Dataset
	return h
}

func (h *HoneycombSpanRecorder) RecordSpan(r lightstep.RawSpan) {
	event := h.builder.NewEvent()
	event.AddField("traceID", r.Context.TraceID)
	event.AddField("id", r.Context.SpanID)
	event.AddField("parentID", r.ParentSpanID)
	event.AddField("operationName", r.Operation)
	event.Timestamp = r.Start
	event.AddField("durationMs", r.Duration.Nanoseconds()/1e6)
	event.AddField("logs", r.Logs)
	addTagsToEvent(r, event)

	if h.Router != nil {
		routeInfo := h.Router(r)
		if routeInfo.Drop {
			return
		}
		event.Dataset = routeInfo.Dataset
		event.WriteKey = routeInfo.WriteKey
		event.SampleRate = routeInfo.SampleRate
	}

	event.Send()
}

// Close waits for any in-flight requests to the Honeycomb API to finish.
func (h *HoneycombSpanRecorder) Close() {
	libhoney.Close()
}

// RouteInfo describes configuration overrides to use when sending a span's
// data to Honeycomb. If you'd like to send spans from different services to
// different datasets, or dynamically sample spans, you can do that by
// supplying a custom RouterFunc implementation to your HoneycombSpanRecorder.
type RouteInfo struct {
	Dataset    string
	WriteKey   string
	SampleRate uint
	Drop       bool
}

type RouterFunc func(lightstep.RawSpan) RouteInfo

func addTagsToEvent(r lightstep.RawSpan, ev *libhoney.Event) {
	if r.Tags != nil {
		for k, v := range r.Tags {
			// If a tag name collides with a reserved attribute (unlikely but
			// potentially confusing), munge its name by prepending "tag."
			if reserved, ok := reservedTags[k]; ok {
				k = reserved
			}
			ev.AddField(k, v)
		}
	}
}
