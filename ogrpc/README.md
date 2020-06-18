# ogrpc

This package can be used for making [gRPC](https://grpc.io) servers and clients observable.
It uses _interceptors_ to intercept your gRPC methods and provides logs, metrics, and traces out-of-the-box.

## Quick Start

Here is a snippet of what you need to do on server-side:

```go
obsv := observer.New(true, observer.Options{
  Name:     "client",
  LogLevel: "info",
})
defer obsv.Close()

si := ogrpc.NewServerInterceptor(obsv, ogrpc.Options{})
opts := si.ServerOptions()
server := grpc.NewServer(opts...)
zonePB.RegisterZoneManagerServer(server, &ZoneServer{})
```

And a snippet of what you need to do on client-side:

```go
obsv := observer.New(true, observer.Options{
  Name:     "client",
  LogLevel: "info",
})
defer obsv.Close()

ci := ogrpc.NewClientInterceptor(obsv, ogrpc.Options{})
opts := ci.DialOptions()
conn, _ := grpc.Dial(grpcServer, opts...)
defer conn.Close()
client := zonePB.NewZoneManagerClient(conn)
```

You can find the full example [here](./example).
