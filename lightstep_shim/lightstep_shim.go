// This package contains convenience helpers for sending Opentracing data
// collected by the Lightstep tracer
// (https://github.com/lightstep/lightstep-tracer-go) to Honeycomb
// (https://honeycomb.io/).
package lightstep_shim

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
	Options Options

	builder *libhoney.Builder
}

type Options struct {
	WriteKey string
	Dataset  string
	Router   RouterFunc
	Sampler  SamplerFunc
}

// NewHoneycombSpanRecorder creates a new HoneycombSpanRecorder configured with
// the given options.
func NewHoneycombSpanRecorder(options Options) *HoneycombSpanRecorder {
	h := &HoneycombSpanRecorder{
		Options: options,
		builder: libhoney.NewBuilder(),
	}
	h.builder.WriteKey = options.WriteKey
	h.builder.Dataset = options.Dataset
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

	if h.Options.Router != nil {
		routeInfo := h.Options.Router(r)
		if routeInfo.WriteKey != "" {
			event.WriteKey = routeInfo.WriteKey
		}
		if routeInfo.Dataset != "" {
			event.Dataset = routeInfo.Dataset
		}
	}

	if h.Options.Sampler != nil {
		sampleRate, drop := h.Options.Sampler(r)
		if drop {
			return
		}
		event.SampleRate = sampleRate
	}

	// The SampleRate is considered advisory, meaning that if you set a
	// SampleRate of 10
	event.SendPresampled()
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
	Dataset  string
	WriteKey string
}

type RouterFunc func(lightstep.RawSpan) RouteInfo

type SamplerFunc func(lightstep.RawSpan) (sampleRate uint, drop bool)

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