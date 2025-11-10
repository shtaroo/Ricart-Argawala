package main

import (
	log "log"
	net "net"
	os "os"
	strconv "strconv"
	time "time"

	grpc "google.golang.org/grpc"

	pb "module/proto"
)

// Node constructor
type Node struct {
	pb.UnimplementedRicartArgawalaServer

	id      int
	state   string
	lamport int
	peers   []string
}

func (node *Node) StartServer(port string) {
	// Create listener
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create and register server
	server := grpc.NewServer()
	pb.RegisterRicartArgawalaServer(server, node)

	// Log for transparency
	log.Printf("Node #%d now listening on localhost%s", node.id, port)

	// Serve
	if err := server.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func main() {
	if len(os.Args) < 4 {
		// Even though we're really looking for 3 arguments, we expect 4 or more because
		// the first (0th) argument always directs to the .exe of the .go file that's run...
		println("Usage: go run node.go <node_id> :<port_num> <peer_1> [<peer_2> ...]")
		return
	}

	// Parse args to create a new Node and server
	node_id, _ := strconv.Atoi(os.Args[1])
	port := os.Args[2]
	peers := os.Args[3:]

	// Create new Node
	node := &Node{
		id:      node_id,
		state:   "RELEASED",
		lamport: 0,
		peers:   peers,
	}

	// Start node-server (start listening for requests)
	node.StartServer(port)

	// Wait 20 seconds so there is time to start all Nodes
	time.Sleep(20 * time.Second)

	// Continuously attempt to enter critical section
	// code goes here ...
}
