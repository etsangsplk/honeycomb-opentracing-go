This repository contains helpers to send [OpenTracing](http://opentracing.io/)
data to [Honeycomb](https://honeycomb.io) from Go services.

## For Lightstep Users

If you're using the [Lightstep tracer](https://github.com/lightstep/lightstep-tracer-go),
you can use the helpers in the `lightstep_shim` package to send its trace data to
Honeycomb as well.

Basic usage looks like this:

```
options := lightstep.Options{
  // ...
}

options.Recorder := lightstep_shim.NewSpanRecorder(
    lightstep_shim.Options{
        WriteKey: "YOUR_WRITE_KEY",
        Dataset: "YOUR_DATASET",
    },
)

tracer := lightstep.NewTracer(options)
opentracing.InitGlobalTracer(tracer)
```

For full configuration options and more examples, please see the
[Godoc](https://godoc.org/github.com/honeycombio/honeycomb-opentracing-go/lightstep_shim).

## Other Tracers

Integration with additional client libraries is coming; please
don't hesitate to get in touch by filing an issue or contacting
[support@honeycombio](mailto:support@honeycomb.io).
