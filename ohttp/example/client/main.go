package main

import (
	"context"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/moorara/observer"
	"github.com/moorara/observer/ohttp"
	"go.opentelemetry.io/otel/api/correlation"
	"go.opentelemetry.io/otel/label"
	"go.uber.org/zap"
)

const port = ":9001"

func main() {
	// Creating a new Observer and set it as the singleton
	obsv := observer.New(true,
		observer.WithMetadata("client", "0.1.0", "production", "ca-central-1", map[string]string{
			"domain": "auth",
		}),
		observer.WithLogger("info"),
		observer.WithPrometheus(),
		observer.WithJaeger("localhost:6831", "", "", ""),
	)
	defer obsv.Close()

	c := &http.Client{
		Timeout:   10 * time.Second,
		Transport: &http.Transport{},
	}

	client := ohttp.NewClient(c, obsv, ohttp.Options{})

	req, err := http.NewRequest("GET", "http://localhost:9000/users/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", nil)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	ctx = correlation.NewContext(ctx,
		label.String("tenant", "1234"),
	)

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	obsv.Logger().Info("received response",
		zap.String("content", string(bytes)),
	)

	http.Handle("/metrics", obsv)
	obsv.Logger().Info("starting http server on ...", zap.String("port", port))
	panic(http.ListenAndServe(port, nil))
}
