package ping

import (
	"fmt"
	"log"
	"time"
	"context"
	"github.com/golang/protobuf/ptypes"
)

type PingService struct {
	UnimplementedPingServiceServer
}

func SendPingMsg(client PingServiceClient) error {

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var resp *Response
	var err error

	fmt.Println("Client SendPingMsg -- ")
	resp, err = client.SendPingMsg(ctx, &Request{
		Message: "Testing Client-Server connection!",
	})

	if err != nil {
		fmt.Println("Error -- SendPingMsg : ", err.Error())
		return fmt.Errorf("failed receive response from server %s", err)
	}

	respMsg := resp.Pong.GetMessage()
	fmt.Println("Pong Msg -- ", respMsg)

	return nil
}

func (s *PingService) SendPingMsg(ctx context.Context, req *Request) (*Response, error) {
	log.Print("sending ping response")

	log.Println("GRPC Server Send...")
	
	return &Response{
		Pong: &Pong{
			Index:      1,
			Message:    req.GetMessage(),
			ReceivedOn: ptypes.TimestampNow(),
		},
	}, nil
}