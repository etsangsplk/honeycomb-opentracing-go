Proof-of-concept level shim to mirror Opentracing data collected by
[Lightstep](github.com/lightstep/lightstep-tracer-go) to Honeycomb.

Usage:

```
options := lightstep.Options{
  // ...
}

options.Recorder := honeycomb.NewSpanRecorder(
    honeycomb.WithWriteKey("YOUR_WRITE_KEY"),
    honeycomb.WithDataset("YOUR_DATASET"),
)

tracer := lightstep.NewTracer(options)

opentracing.InitGlobalTracer(tracer)
```
