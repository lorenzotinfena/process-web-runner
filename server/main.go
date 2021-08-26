package main

import (
	"context"
	"log"
	"net"
	"os"
	"strings"
	pb "app/protos/service"
	"google.golang.org/grpc"
	"syscall"
)

type server struct {
	pb.UnimplementedRunnerServer
}

type process struct {
	pid int // if -1, the process is not running
	command string
}

var processes []process

func (s *server) Add(ctx context.Context, command_pc *pb.Command) (*pb.Empty, error) {
	processes = append(processes, process{pid: -1, command: command_pc.GetName()})
	return &pb.Empty{}, nil
}

func (s *server) Run(ctx context.Context, id_pb *pb.Id) (*pb.Empty, error) {
	id := id_pb.GetId()
	if int(id) < len(processes) {
		command := processes[id].command
		log.Println("starting: " + command)
		comm := strings.Fields(command)
		p, err := os.StartProcess(comm[0], comm[1:], nil)
		if err == nil {
			processes = append(processes, process{pid: p.Pid, command: command})
		}
	}
	return &pb.Empty{}, nil
}

func (s *server) RequestStatus(ctx context.Context, command *pb.Empty) (*pb.Status, error) {
	// fix running processes
	for i, p := range processes{
		if p.pid != -1 {
			pr, err := os.FindProcess(p.pid)
			err = pr.Signal(syscall.Signal(0))
			if err != nil {
				processes[i].pid = -1
			}
		}
	}
	processes_status := []*pb.ProcessStatus{}

	for i, p := range processes{
		var active bool
		if p.pid != -1 {
			active = true
		} else {
			active = false
		}

		processes_status = append(processes_status, &pb.ProcessStatus{Command: &pb.Command{Name: processes[i].command },
													Id: &pb.Id{Id: uint32(i)},
													Active: active})
	}
	status := pb.Status{Processes: processes_status}
	return &status, nil
}

func (s *server) Stop(ctx context.Context, id_pb *pb.Id) (*pb.Empty, error) {
	id := id_pb.GetId()
	if int(id) < len(processes) {
		p := processes[id]
		if p.pid != -1 {
			pr, _ := os.FindProcess(p.pid)
			_ = pr.Release()
			err := pr.Kill()
			if err == nil{
				processes[id].pid = -1
			}
		}
	}
	return &pb.Empty{}, nil
}

func main() {
	processes = []process{}
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Println("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterRunnerServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Println("failed to serve: %v", err)
	}
}
