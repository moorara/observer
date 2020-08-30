[![Go Doc][godoc-image]][godoc-url]

# ohttp

This package can be used for making http servers and clients observable.
It uses _middleware_ to wrap your http handlers and provides logs, metrics, and traces out-of-the-box.

## Quick Start

Here is a snippet of what you need to do on server-side:

```go
obsv := observer.New(true,
  observer.WithMetadata("server", "", "", "", nil),
  observer.WithLogger("info"),
})
defer obsv.Close()

mid := ohttp.NewMiddleware(obsv, ohttp.Options{})
wrapped := mid.Wrap(handler)
```

And a snippet of what you need to do on client-side:

```go
obsv := observer.New(true,
  observer.WithMetadata("client", "", "", "", nil),
  observer.WithLogger("info"),
})
defer obsv.Close()

c := &http.Client{}
client := ohttp.NewClient(c, obsv, ohttp.Options{})
```

You can find the full example [here](./example).


[godoc-url]: https://pkg.go.dev/github.com/moorara/observer/ohttp
[godoc-image]: https://godoc.org/github.com/moorara/observer/ohttp?status.svg
