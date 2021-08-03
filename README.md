# grpc-connection-library
grpc-connection-library is a library that supports the gRPC client-server connection interface for the developers to use as a gRPC middleware in the application. The library is written in Golang with a concurrency pipeline design pattern to synchronize the gRPC connection pool system.

# Features
- The gRPC connection flow among server/client synchronized using the Ping/Pong services.
- gRPC connection library supports connection pool reuse the gRPC client connection instance.
- Concurrency Pipeline design pattern synchronizes the data flow among several stages while creating the connection pool.
- The selection process of the gRPC connection instance from the pool is designed using the <a href="https://pkg.go.dev/reflect#SelectCase">reflect.SelectCase</a> that supports pseudo-random technique for choosing among different cases.
- <a href="https://github.com/Deeptiman/go-batch">go-batch</a> processing library implemented to divide the connection instances from the pool into batches.
- The grpc-retry policy helps to retry the failure of gRPC connections with backoff strategy.
- The <a href="https://pkg.go.dev/google.golang.org/grpc/grpclog">grpclog</a> will show the internal connection lifecycle that will be useful to debug the connection flow.

# Installation

`go get github.com/Deeptiman/go-connection-library`

# Demo


# Go Docs

# Example 
<b>Client</b>:
 
```````````````````````````````go
package main

import (
	"fmt"
	grpc "github.com/Deeptiman/grpc-connection-library/grpc"
)

func main() {
	address := "localhost:50051"
	client, err := grpc.NewGRPCConnection(grpc.WithAddress(address), grpc.WithConnectionType(grpc.Client))
	if err != nil {
		fmt.Println("Failed to create GRPC Connection - ", err.Error())
		return
	}
	conn, err := client.GetConn()
	if err != nil {
		return
	}
	fmt.Println("GRPC Client Connected State= ", conn.GetState().String())
}
```````````````````````````````

<b> Server </b>

```````````````````````````````go
package main

import (
	"fmt"
	grpc "github.com/Deeptiman/grpc-connection-library/grpc"
)

func main() {
	
	server, err := grpc.NewGRPCConnection(grpc.WithConnectionType(grpc.Server))
	if err != nil {
		fmt.Println("Failed to create GRPC Connection - ", err.Error())
		return
	}
	server.ListenAndServe()
}
```````````````````````````````

# gRPC connection Ping/Pong service
The library provides a Ping/Pong service facility to test the client/server connection flow. The service helps to establish connection health check status.

<b>protos:</b>
`````````````````````````````````````proto
syntax = "proto3";

package ping;

option go_package = ".;ping";

import "google/protobuf/timestamp.proto";

service PingService {
    rpc SendPongMsg(Request) returns (Response) {}
}

message Request {
    string message = 1;
}

message Pong {
    int32 index = 1;
    string message = 2;
    google.protobuf.Timestamp received_on = 3;
}

message Response {
    Pong pong = 1;
}
`````````````````````````````````````
<b> Server:</b>
1. ListenAndServe will start listening to the specific serverPort with the <b>"tcp"</b> network type. 
2. After the server listener socket is opened, the initial Ping request gets registered that can be used by any gRPC clients to sent Ping-Pong request to check the client-server gRPC connection health status.

``````````````````````````````````go
grpcServer := grpc.NewServer(serverOptions...)
// Register the Ping service request
pb.RegisterPingServiceServer(grpcServer, &pb.PingService{})
if err = grpcServer.Serve(listener); err != nil {
	g.log.Errorln("failed start server - %v", err)
	g.log.Fatal(err)
	return err
}
``````````````````````````````````

<b> Client: </b>
1. gRPC client connection instance creates Ping service to test connection health.
2. The <b>SendPingMsg</b> sends a test msg to the target server address to get the Pong response msg back to verify the connection flow.

`````````````````````````````````go
conn, err := grpc.Dial(address, opts...)
if err != nil {
    c.Log.Fatal(err)
    return nil, err
}
client := pb.NewPingServiceClient(c.Conn)
respMsg, err := pb.SendPingMsg(client)
if err != nil {
   return nil, err
}
c.Log.Infoln("GRPC Pong msg - ", respMsg)
`````````````````````````````````

# Concurrency Pipeline for gRPC Connection Pool
1. <a href="https://github.com/Deeptiman/grpc-connection-library/blob/master/pool/connection_pool.go#L170">ConnectionPoolPipeline()</a> follows the concurrency pipeline technique to create a connection pool in a higher concurrent scenarios. The pipeline has several stages that use the <b>Fan-In, Fan-Out</b> technique to process the data pipeline using channels.
2. The entire process of creating the connection pool becomes a powerful function using the pipeline technique. The four stages work as a generator pattern for the connection pool.

## Pipeline Stages
### Stage-1: 
This stage will create the initial gRPC connection instance that gets passed to the next pipeline stage for replication.
	
 `````````````````````````````````````````````````go
 connInstancefn := func(done chan interface{}) <-chan *grpc.ClientConn {

   connCh := make(chan *grpc.ClientConn)

   conn, err := c.ClientConn()
   if err != nil {
	done <- err
   }

    go func() {
       c.Log.Infoln("1#connInstance ...")
       defer close(connCh)
	 select {
	   case connCh <- conn:
	     c.Log.Infoln("GRPC Connection Status - ", conn.GetState().String())
	 }
    }()
  return connCh 
}
`````````````````````````````````````````````````
### Stage-2: 
The cloning process of the initial gRPC connection object will begin here. The connection instance gets passed to the next stage iteratively via channels.

`````````````````````````````````````````````````go
connReplicasfn := func(connInstanceCh <-chan *grpc.ClientConn) <-chan *grpc.ClientConn {
	connInstanceReplicaCh := make(chan *grpc.ClientConn)
		go func() {
			c.Log.Infoln("2#connReplicas ...")
			defer close(connInstanceReplicaCh)
			for conn := range connInstanceCh {
				for i := 0; uint64(i) < c.MaxPoolSize; i++ {
					select {
					case connInstanceReplicaCh <- conn:
					}
				}
			}
		}()
	return connInstanceReplicaCh
}
`````````````````````````````````````````````````
### Stage-3: 
This stage will start the batch processing using the <a href="https://github.com/Deeptiman/go-batch">github.com/Deeptiman/go-batch</a> library. The MaxPoolSize is divided into multiple batches and released via a supply channel from go-batch library internal implementation.

`````````````````````````````````````````````````go
connBatchfn := func(connInstanceCh <-chan *grpc.ClientConn) chan []batch.BatchItems {

		go func() {
			c.Log.Infoln("3#connBatch ...")
			c.ConnInstanceBatch.StartBatchProcessing()
			for conn := range connInstanceCh {
				select {
				case c.ConnInstanceBatch.Item <- conn:
				}
			}
		}()
	return c.ConnInstanceBatch.Consumer.Supply.ClientSupplyCh
}

`````````````````````````````````````````````````
### Stage-4:
The connection queue reads through the go-batch client supply channel and stores the connection instances as channel case in <b>[]reflect.SelectCase</b>. So, whenever the client requests a connection instance, <b>reflect.SelectCase</b> retrieves the conn instances from the case using the pseudo-random technique.

`````````````````````````````````````````````````go
connEnqueuefn := func(connSupplyCh <-chan []batch.BatchItems) <-chan batch.BatchItems {
	receiveBatchCh := make(chan batch.BatchItems)
		go func() {
			c.Log.Infoln("4#connEnqueue ...")
			defer close(receiveBatchCh)
			for supply := range connSupplyCh {

				for _, s := range supply {
					c.EnqueConnBatch(s)
					select {
					case receiveBatchCh <- s:
					}
				}
			}
		}()
	return receiveBatchCh
}
`````````````````````````````````````````````````
### Run the pipeline:
`````````````````````````````````````````````````go
for s := range connEnqueuefn(connBatchfn(connReplicasfn(connInstancefn(done)))) {
	go func(s batch.BatchItems) {
		atomic.AddUint64(&ConnPoolPipeline, 1)
		if c.GetConnPoolSize() == c.MaxPoolSize {
			select {
			case pipelineDoneChan <- "Done":
				return
			}
		}
	}(s)
}
`````````````````````````````````````````````````
