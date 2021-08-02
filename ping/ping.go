package ping

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes"
	"log"
	"time"
)

// PingService struct can work as Ping/Pong service between server and client to test the gRPC connection flow.
type PingService struct {
	// UnimplementedPingServiceServer can be embedded to have forward-compatible implementations
	UnimplementedPingServiceServer
}

// SendPingMsg function used by the client to send a ping message to the server.
func SendPingMsg(client PingServiceClient) (string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var resp *Response
	var err error

	fmt.Println("GRPC Client SendPingMsg -- ")
	resp, err = client.SendPongMsg(ctx, &Request{
		Message: "Testing Client-Server connection!",
	})

	if err != nil {
		fmt.Println("Error -- SendPingMsg : ", err.Error())
		return "", err
	}

	fmt.Println("GRPC Sent Ping Msg to Server .....")

	return resp.Pong.GetMessage(), nil
}

// SendPongMsg function used by the server to send pong messages to the client.
func (s *PingService) SendPongMsg(ctx context.Context, req *Request) (*Response, error) {
	log.Print("sending pong response")

	log.Println("GRPC Server Send...")

	return &Response{
		Pong: &Pong{
			Index:      1,
			Message:    req.GetMessage(),
			ReceivedOn: ptypes.TimestampNow(),
		},
	}, nil
}
