package main

import (
	proto "ITUServer/grpc"
	"context"
)

type ITU_databaseServer struct {
	proto.UnimplementedITUDatabaseServer
	students []string
}

func (s *ITU_databaseServer) GetStudents(ctx context.Context, in *proto.Empty) (*proto.Students, error) {
	return &proto.Students{Students: s.students}, nil
}

func main() {

}
