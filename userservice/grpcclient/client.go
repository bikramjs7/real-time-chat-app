package grpcclient

import (
	"context"
	"time"

	pb "userservice/proto"

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

func (c *Client) Logout(userId string) (*pb.LogoutResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &pb.LogoutRequest{Userid: userId}
	return c.client.Logout(ctx, req)
}
