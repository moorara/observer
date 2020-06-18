package main

import (
	"io"
	"net/http"
	"time"

	"github.com/moorara/observer"
	"github.com/moorara/observer/ohttp"
	"go.uber.org/zap"
)

const port = ":9000"

func main() {
	// Creating a new Observer and set it as the singleton
	obsv := observer.New(true, observer.Options{
		Name:        "service-b",
		Version:     "0.1.0",
		Environment: "production",
		Region:      "ca-central-1",
		Tags: map[string]string{
			"domain": "auth",
		},
		LogLevel:            "info",
		JaegerAgentEndpoint: "localhost:6831",
	})
	defer obsv.Close()

	mid := ohttp.NewMiddleware(obsv, ohttp.Options{})

	handler := mid.Wrap(func(w http.ResponseWriter, req *http.Request) {
		time.Sleep(50 * time.Millisecond)

		ctx := req.Context()
		ctx, span := obsv.Tracer().Start(ctx, "database-read")
		defer span.End()

		time.Sleep(100 * time.Millisecond)

		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "Hello, world!")

		observer.LoggerFromContext(ctx).Debug("responded back!")
	})

	http.Handle("/users/", handler)
	http.Handle("/metrics", obsv)
	obsv.Logger().Info("starting http server on ...", zap.String("port", port))
	panic(http.ListenAndServe(port, nil))
}
