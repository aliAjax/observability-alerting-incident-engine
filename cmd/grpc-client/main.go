package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
)

type rawMessage struct {
	Data []byte
}

func (m *rawMessage) Reset()         { m.Data = m.Data[:0] }
func (m *rawMessage) String() string { return string(m.Data) }
func (m *rawMessage) ProtoMessage()  {}

type rawCodec struct{}

func (rawCodec) Name() string { return "jsonraw" }
func (rawCodec) Marshal(v any) ([]byte, error) {
	msg, ok := v.(*rawMessage)
	if !ok {
		return nil, fmt.Errorf("raw codec expects *rawMessage")
	}
	return msg.Data, nil
}
func (rawCodec) Unmarshal(data []byte, v any) error {
	msg, ok := v.(*rawMessage)
	if !ok {
		return fmt.Errorf("raw codec expects *rawMessage")
	}
	msg.Data = append(msg.Data[:0], data...)
	return nil
}

func init() {
	encoding.RegisterCodec(rawCodec{})
}

func main() {
	addr := flag.String("addr", "localhost:9090", "gRPC server address")
	method := flag.String("method", "EvaluateNow", "method name")
	body := flag.String("body", "{}", "JSON request body")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, *addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.CallContentSubtype("jsonraw")),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	req := &rawMessage{Data: []byte(*body)}
	resp := &rawMessage{}
	if err := conn.Invoke(ctx, "/observability.alerting.Engine/"+*method, req, resp); err != nil {
		fmt.Fprintf(os.Stderr, "invoke: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(resp.Data))
}
