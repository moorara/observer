package main

import (
	"context"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/moorara/observer"
	"github.com/moorara/observer/ogrpc"
	"github.com/moorara/observer/ogrpc/example/zonePB"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	httpPort   = ":9002"
	grpcServer = "localhost:9000"
)

func main() {
	// Creating a new Observer and set it as the singleton
	obsv := observer.New(true,
		observer.WithMetadata("client", "0.1.0", "production", "ca-central-1", map[string]string{
			"domain": "core",
		}),
		observer.WithLogger("info"),
		observer.WithPrometheus(),
		observer.WithJaeger("localhost:6831", "", "", ""),
	)
	defer obsv.End(context.Background())

	ci := ogrpc.NewClientInterceptor(obsv, ogrpc.Options{})

	go func() {
		http.Handle("/metrics", obsv)
		obsv.Logger().Info("starting http server on ...", zap.String("port", httpPort))
		panic(http.ListenAndServe(httpPort, nil))
	}()

	opts := ci.DialOptions()
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(grpcServer, opts...)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := zonePB.NewZoneManagerClient(conn)
	obsv.Logger().Sugar().Infof("client connected to server at %s", grpcServer)

	for {
		getContainingZone(client)
		getPlacesInZone(client)
		getUsersInZone(client)
		getUsersInZones(client)
	}
}

func getContainingZone(client zonePB.ZoneManagerClient) {
	// A random delay between 1s to 5s
	d := 1 + rand.Intn(4)
	time.Sleep(time.Duration(d) * time.Second)

	ctx := context.Background()

	header := new(metadata.MD)
	stream, err := client.GetContainingZone(ctx, grpc.Header(header))
	if err != nil {
		panic(err)
	}

	locations := []*zonePB.Location{
		{Latitude: 43.662892, Longitude: -79.395684},
		{Latitude: 43.658776, Longitude: -79.379327},
	}

	for _, loc := range locations {
		err := stream.Send(loc)
		if err != nil {
			panic(err)
		}
	}

	_, err = stream.CloseAndRecv()
	if err != nil {
		panic(err)
	}
}

func getPlacesInZone(client zonePB.ZoneManagerClient) {
	// A random delay between 1s to 5s
	d := 1 + rand.Intn(4)
	time.Sleep(time.Duration(d) * time.Second)

	ctx := context.Background()
	zone := &zonePB.Zone{
		Location: &zonePB.Location{Latitude: 43.645844, Longitude: -79.379742},
		Radius:   1200,
	}

	header := new(metadata.MD)
	_, err := client.GetPlacesInZone(ctx, zone, grpc.Header(header))
	if err != nil {
		panic(err)
	}
}

func getUsersInZone(client zonePB.ZoneManagerClient) {
	// A random delay between 1s to 5s
	d := 1 + rand.Intn(4)
	time.Sleep(time.Duration(d) * time.Second)

	ctx := context.Background()
	zone := &zonePB.Zone{
		Location: &zonePB.Location{Latitude: 43.645844, Longitude: -79.379742},
		Radius:   1200,
	}

	header := new(metadata.MD)
	stream, err := client.GetUsersInZone(ctx, zone, grpc.Header(header))
	if err != nil {
		panic(err)
	}

	for {
		_, err := stream.Recv()
		if err != nil && err != io.EOF {
			panic(err)
		}

		if err == io.EOF {
			return
		}
	}
}

func getUsersInZones(client zonePB.ZoneManagerClient) {
	// A random delay between 1s to 5s
	d := 1 + rand.Intn(4)
	time.Sleep(time.Duration(d) * time.Second)

	ctx := context.Background()
	zones := []*zonePB.Zone{
		{
			Location: &zonePB.Location{Latitude: 45.424688, Longitude: -75.699565},
			Radius:   1500,
		},
		{
			Location: &zonePB.Location{Latitude: 43.472920, Longitude: -80.542378},
			Radius:   1000,
		},
	}

	header := new(metadata.MD)
	stream, err := client.GetUsersInZones(ctx, grpc.Header(header))
	if err != nil {
		panic(err)
	}

	waitc := make(chan struct{})

	// Receiving
	go func() {
		for {
			_, err := stream.Recv()
			if err != nil && err != io.EOF {
				panic(err)
			}

			if err == io.EOF {
				close(waitc)
				return
			}
		}
	}()

	// Sending
	for _, zone := range zones {
		err := stream.Send(zone)
		if err != nil {
			panic(err)
		}
	}

	err = stream.CloseSend()
	if err != nil {
		panic(err)
	}

	<-waitc
}
