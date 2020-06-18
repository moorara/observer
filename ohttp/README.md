# ohttp

This package can be used for making [gRPC](https://grpc.io) servers and clients Observable.
It uses _middleware_ to wrap your http handlers and provides logs, metrics, and traces out-of-the-box.

## Quick Start

Here is a snippet of what you need to do on server-side:

```go
obsv := observer.New(true, observer.Options{
  Name:     "client",
  LogLevel: "info",
})
defer obsv.Close()

mid := ohttp.NewMiddleware(obsv, ohttp.Options{})
wrapped := mid.Wrap(handler)
```

And a snippet of what you need to do on client-side:

```go
obsv := observer.New(true, observer.Options{
  Name:     "client",
  LogLevel: "info",
})
defer obsv.Close()

c := &http.Client{}
client := ohttp.NewClient(c, obsv, ohttp.Options{})
```

You can find the full example [here](./example).
