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
 
 ```````````````````````````````
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
   conn , err := client.GetConn()
   if err != nil {
      return
   }
   fmt.Println("GRPC Client Connected State= ", conn.GetState().String())
}
```````````````````````````````

<b> Server </b>

```````````````````````````````
import (
	"fmt"
	grpc "github.com/Deeptiman/grpc-connection-library/grpc"
)

func main() {
	fmt.Println("GRPC Ping Server Example!")

	server, err := grpc.NewGRPCConnection(grpc.WithConnectionType(grpc.Server))
	if err != nil {
		fmt.Println("Failed to create GRPC Connection - ", err.Error())
		return
	}
	server.ListenAndServe()
}

```````````````````````````````

