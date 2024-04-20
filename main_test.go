package main_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strconv"
	"testing"

	"google.golang.org/grpc"

	"github.com/otakakot/sample-go-grpc-stream/pb"
)

func TestSingle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cc, err := grpc.Dial("localhost:8080", grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	defer cc.Close()

	cli := pb.NewStreamClient(cc)

	read, err := os.Open("sample.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer read.Close()

	stream, err := cli.Single(ctx)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)

	if err := stream.Send(&pb.SingleRequest{
		Value: &pb.SingleRequest_Id{
			Id: 1,
		},
	}); err != nil {
		t.Fatal(err)
	}

	for {
		n, err := read.Read(buf)
		if err != nil {
			break
		}

		if err := stream.Send(&pb.SingleRequest{
			Value: &pb.SingleRequest_Chunk{
				Chunk: &pb.Chunk{
					Data:     buf,
					Position: int64(n),
				},
			},
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := stream.CloseSend(); err != nil {
		t.Fatal(err)
	}

	id := int64(0)

	wt := bytes.NewBuffer([]byte{})

	for {
		res, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		if res.GetId() != 0 {
			id = res.GetId()
		}

		if res.GetChunk() != nil {
			if _, err := wt.Write(res.GetChunk().GetData()); err != nil {
				t.Fatal(err)
			}
		}
	}

	write, err := os.Create("photo/client/single_" + strconv.Itoa(int(id)) + ".jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer write.Close()

	if _, err := wt.WriteTo(write); err != nil {
		t.Fatal(err)
	}

	want, err := os.ReadFile("sample.jpg")
	if err != nil {
		t.Fatal(err)
	}

	{
		got, err := os.ReadFile("photo/client/single_" + strconv.Itoa(int(id)) + ".jpg")
		if err != nil {
			t.Fatal(err)
		}

		if bytes.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	{
		got, err := os.ReadFile("photo/server/single_" + strconv.Itoa(int(id)) + ".jpg")
		if err != nil {
			t.Fatal(err)
		}

		if bytes.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestMultiple(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cc, err := grpc.Dial("localhost:8080", grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	defer cc.Close()

	cli := pb.NewStreamClient(cc)

	for id := range 10 {
		read, err := os.Open("sample.jpg")
		if err != nil {
			t.Fatal(err)
		}
		defer read.Close()

		stream, err := cli.Multiple(ctx)
		if err != nil {
			t.Fatal(err)
		}

		buf := make([]byte, 1024)

		for {
			n, err := read.Read(buf)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatal(err)
			}

			if err := stream.Send(&pb.MultipleRequest{
				Values: []*pb.Value{
					{
						Id: int64(id + 1),
						Chunk: &pb.Chunk{
							Data:     buf,
							Position: int64(n),
						},
					},
				},
			}); err != nil {
				t.Fatal(err)
			}
		}

		if err := stream.CloseSend(); err != nil {
			t.Fatal(err)
		}

		values := map[int64]*bytes.Buffer{}

		for {
			res, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatal(err)
			}

			for _, value := range res.GetValues() {
				id := value.GetId()

				if values[id] == nil {
					values[id] = bytes.NewBuffer([]byte{})
				}

				if _, err := values[id].Write(value.GetChunk().GetData()); err != nil {
					t.Fatal(err)
				}
			}
		}

		for id, wt := range values {
			write, err := os.Create("photo/client/multiple_" + strconv.Itoa(int(id)) + ".jpg")
			if err != nil {
				t.Fatal(err)
			}
			defer write.Close()

			if _, err := wt.WriteTo(write); err != nil {
				t.Fatal(err)
			}
		}
	}

	want, err := os.ReadFile("sample.jpg")
	if err != nil {
		t.Fatal(err)
	}

	for id := range 10 {
		{
			got, err := os.ReadFile("photo/client/multiple_" + strconv.Itoa(int(id+1)) + ".jpg")
			if err != nil {
				t.Fatal(err)
			}

			if bytes.Equal(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
		}
		{
			got, err := os.ReadFile("photo/server/multiple_" + strconv.Itoa(int(id+1)) + ".jpg")
			if err != nil {
				t.Fatal(err)
			}

			if bytes.Equal(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
		}
	}
}
