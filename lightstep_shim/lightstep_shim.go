// This package contains convenience helpers for sending Opentracing data
// collected by the Lightstep tracer
// (https://github.com/lightstep/lightstep-tracer-go) to Honeycomb
// (https://honeycomb.io/).
package lightstep_shim

import (
	"fmt"

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
// forwarding finished spans to the Honeycomb ingest API.
type HoneycombSpanRecorder struct {
	Options Options

	builder *libhoney.Builder
}

// Options configure authentication to the Honeycomb API, as well as optional
// routing and sampling.
type Options struct {
	// WriteKey is your Honeycomb team's write key, found at
	// https://ui.honeycomb.io/account.
	WriteKey string
	// Dataset is the name of the destination dataset for spans.
	Dataset string
	// Router is an optional function to override the write key and dataset
	// parameters on a per-span basis.
	Router RouterFunc
	// Sampler is an optional function for sampling spans that get sent.
	Sampler SamplerFunc
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
	if h.builder == nil {
		fmt.Println("HoneycombSpanRecorder wasn't initialized properly! Span transmission might not work.")
		h.builder = libhoney.NewBuilder()
	}
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

	// Using libhoney's fully-randomized sampling isn't likely to work very
	// well for tracing data. In real-world use cases, we'd probably want to
	// take trace ID into account and/or apply some sort of dynamic sampling
	// policy. So we expect Sampler implementations to tell us which events to
	// drop, and what sample rate to set on retained events. Thus it's
	// essential to call SendPresampled() instead of Send() here.
	event.SendPresampled()
}

// Close waits for any in-flight requests to the Honeycomb API to finish.
func (h *HoneycombSpanRecorder) Close() {
	libhoney.Close()
}

// RouteInfo describes configuration overrides to use when sending a span's
// data to Honeycomb. If you'd like to send spans from different services to
// different datasets or different teams, you can do that by
// supplying a custom RouterFunc implementation as Options.Router.
type RouteInfo struct {
	Dataset  string
	WriteKey string
}

// RouterFunc is the function signature for implementations of Options.Router.
type RouterFunc func(lightstep.RawSpan) RouteInfo

// SamplerFunc is the function signature for implementations of
// Options.Sampler. By providing a sampling function, you can send just a
// representative sample of spans to Honeycomb.
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
