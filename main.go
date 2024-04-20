package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/otakakot/sample-go-grpc-stream/pb"
)

func main() {
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	srv := grpc.NewServer()

	pb.RegisterStreamServer(srv, &StreamServer{})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	defer stop()

	go func() {
		slog.Info("start server. listening on :8080")

		if err := srv.Serve(lis); err != nil {
			panic(err)
		}
	}()

	<-ctx.Done()

	slog.Info("start server shutdown")

	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	srv.GracefulStop()

	slog.Info("done server shutdown")
}

var _ pb.StreamServer = (*StreamServer)(nil)

type StreamServer struct{}

// Single implements pb.StreamServer.
func (s *StreamServer) Single(
	stream pb.Stream_SingleServer,
) error {
	id := int64(0)

	wt := bytes.NewBuffer([]byte{})

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return err
		}

		if req.GetId() != 0 {
			id = req.GetId()
		}

		if req.GetChunk() != nil {
			if _, err := wt.Write(req.GetChunk().GetData()); err != nil {
				return err
			}
		}
	}

	write, err := os.Create("photo/server/single_" + strconv.Itoa(int(id)) + ".jpg")
	if err != nil {
		return err
	}
	defer write.Close()

	if _, err := wt.WriteTo(write); err != nil {
		return err
	}

	read, err := os.Open("photo/server/single_" + strconv.Itoa(int(id)) + ".jpg")
	if err != nil {
		return err
	}
	defer read.Close()

	if err := stream.Send(&pb.SingleResponse{
		Value: &pb.SingleResponse_Id{
			Id: id,
		},
	}); err != nil {
		return err
	}

	buf := make([]byte, 1024)

	for {
		n, err := read.Read(buf)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		if err := stream.Send(&pb.SingleResponse{
			Value: &pb.SingleResponse_Chunk{
				Chunk: &pb.Chunk{
					Data:     buf,
					Position: int64(n),
				},
			},
		}); err != nil {
			return err
		}
	}

	return nil
}

// Multiple implements pb.StreamServer.
func (s *StreamServer) Multiple(
	stream pb.Stream_MultipleServer,
) error {
	values := map[int64]*bytes.Buffer{}

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		for _, value := range req.GetValues() {
			id := value.GetId()

			if values[id] == nil {
				values[id] = bytes.NewBuffer([]byte{})
			}

			if value.GetChunk() != nil {
				if _, err := values[id].Write(value.GetChunk().GetData()); err != nil {
					return err
				}
			}

		}
	}

	for id, wt := range values {
		write, err := os.Create("photo/server/multiple_" + strconv.Itoa(int(id)) + ".jpg")
		if err != nil {
			return err
		}
		defer write.Close()

		if _, err := wt.WriteTo(write); err != nil {
			return err
		}
	}

	for id := range values {
		read, err := os.Open("photo/server/multiple_" + strconv.Itoa(int(id)) + ".jpg")
		if err != nil {
			return err
		}
		defer read.Close()

		buf := make([]byte, 1024)

		for {
			n, err := read.Read(buf)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return err
			}

			if err := stream.Send(&pb.MultipleResponse{
				Values: []*pb.Value{
					{
						Id: id,
						Chunk: &pb.Chunk{
							Data:     buf,
							Position: int64(n),
						},
					},
				},
			}); err != nil {
				return err
			}
		}
	}

	return nil
}
