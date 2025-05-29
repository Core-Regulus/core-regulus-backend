package main

import (
    "context"
    "log"
    "net"
    "google.golang.org/grpc"
		"google.golang.org/grpc/reflection"
    pb "core-regulus/service.pb" // путь к сгенерированному пакету
)

type server struct {
    pb.UnimplementedGreeterServer
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
    return &pb.HelloReply{Message: "Hello, " + in.GetName()}, nil
}

func main() {
    lis, err := net.Listen("tcp", ":5000")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    s := grpc.NewServer()
    pb.RegisterGreeterServer(s, &server{})
		reflection.Register(s) 

    log.Println("gRPC server listening on :5000")
    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}