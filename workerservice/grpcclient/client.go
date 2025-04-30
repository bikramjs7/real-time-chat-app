package grpcclient

import (
	"context"
	"time"

	pb "workerservice/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.NotificationServiceClient
}

func NewClient(address string) (*Client, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		return nil, err
	}

	client := pb.NewNotificationServiceClient(conn)
	return &Client{conn: conn, client: client}, nil
}

func (c *Client) Close() {
	c.conn.Close()
}

func (c *Client) PushMessage(sender string, recipients []string, message string) (*pb.PushMessageResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &pb.PushMessageRequest{
		Sender:     sender,
		Recipients: recipients,
		Message:    message,
	}

	return c.client.PushMessage(ctx, req)
}
