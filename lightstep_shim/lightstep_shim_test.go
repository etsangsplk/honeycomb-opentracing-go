package lightstep_shim_test

import (
	"testing"
	"time"

	"github.com/honeycombio/honeycomb-opentracing-go/lightstep_shim"
	libhoney "github.com/honeycombio/libhoney-go"
	lightstep "github.com/lightstep/lightstep-tracer-go"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
)

func ExampleHoneycombSpanRecorder() {
	options := lightstep.Options{
	// ...
	}

	options.Recorder = lightstep_shim.NewHoneycombSpanRecorder(
		lightstep_shim.Options{
			WriteKey: "YOUR_WRITE_KEY",
			Dataset:  "YOUR_DATASET",
		},
	)

	tracer := lightstep.NewTracer(options)
	opentracing.InitGlobalTracer(tracer)
}

func ExampleRouter() {
	router := func(span lightstep.RawSpan) (r lightstep_shim.RouteInfo) {
		if env, ok := span.Tags["env"]; ok {
			if env == "prod" {
				r.Dataset = "prod_spans"
			} else {
				r.Dataset = "dev_spans"
			}
		}
		return r
	}

	recorder := lightstep_shim.NewHoneycombSpanRecorder(
		lightstep_shim.Options{
			WriteKey: "YOUR_WRITE_KEY",
			Dataset:  "YOUR_DATASET",
			Router:   router,
		},
	)

	tracer := lightstep.NewTracer(lightstep.Options{
		Recorder: recorder,
	})
	opentracing.InitGlobalTracer(tracer)
}

func ExampleSampler() {
	// Assuming trace IDs are generated randomly, send all the spans for one
	// out of every ten traces.
	sampler := func(span lightstep.RawSpan) (sampleRate uint, drop bool) {
		if span.Context.TraceID%10 == 0 {
			return 10, false
		} else {
			return 0, true
		}
	}

	recorder := lightstep_shim.NewHoneycombSpanRecorder(
		lightstep_shim.Options{
			WriteKey: "YOUR_WRITE_KEY",
			Dataset:  "YOUR_DATASET",
			Sampler:  sampler,
		},
	)

	tracer := lightstep.NewTracer(lightstep.Options{
		Recorder: recorder,
	})
	opentracing.InitGlobalTracer(tracer)
}

func TestSpanRecording(t *testing.T) {
	honeycombMock := &libhoney.MockOutput{}
	libhoney.Init(libhoney.Config{
		Output: honeycombMock,
	})
	recorder := lightstep_shim.NewHoneycombSpanRecorder(
		lightstep_shim.Options{
			WriteKey: "test",
			Dataset:  "test",
		})

	tracer := lightstep.NewTracer(lightstep.Options{
		AccessToken:   "-",
		ReportTimeout: time.Millisecond, // Stop the lightstep tracer from hanging on close
		LightStepAPI:  lightstep.Endpoint{Host: "localhost", Port: 9, Plaintext: false},
		Recorder:      recorder,
	})

	span := tracer.StartSpan("testOperation")
	span.SetTag("exampleTag", "value")
	span.Finish()

	tracer.Close()
	libhoney.Close()
	assert.Equal(t, len(honeycombMock.Events()), 1)
	assert.Equal(t, honeycombMock.Events()[0].Fields()["operationName"], "testOperation")
	assert.Equal(t, honeycombMock.Events()[0].Fields()["exampleTag"], "value")
}

func TestSampling(t *testing.T) {
	honeycombMock := &libhoney.MockOutput{}
	libhoney.Init(libhoney.Config{
		Output: honeycombMock,
	})

	sampler := func(span lightstep.RawSpan) (sampleRate uint, drop bool) {
		if span.Context.TraceID%10 == 0 {
			return 10, false
		} else {
			return 0, true
		}
	}

	recorder := lightstep_shim.NewHoneycombSpanRecorder(
		lightstep_shim.Options{
			WriteKey: "test",
			Dataset:  "test",
			Sampler:  sampler,
		})

	tracer := lightstep.NewTracer(lightstep.Options{
		AccessToken:   "-",
		ReportTimeout: time.Millisecond, // Stop the lightstep tracer from hanging on close
		LightStepAPI:  lightstep.Endpoint{Host: "localhost", Port: 9, Plaintext: false},
		Recorder:      recorder,
	})

	for i := 0; i < 100; i++ {
		span := tracer.StartSpan("testOperation")
		span.SetTag("exampleTag", "value")
		span.Finish()
	}

	tracer.Close()
	libhoney.Close()
	assert.True(t, 0 < len(honeycombMock.Events()) && len(honeycombMock.Events()) < 20)
	for _, ev := range honeycombMock.Events() {
		assert.Equal(t, ev.SampleRate, uint(10))
	}
}
