package main

import (
	"os"
	"strconv"

	pb "module/proto"
)

// Node constructor
type Node struct {
	pb.UnimplementedNodeServer

	node_id int
	state   string
	lamport int
	peers   []string
}

func main() {
	if len(os.Args) < 4 {
		// Even though we're really looking for 3 arguments, we check for less than 4.
		// This is because the first (0th) argument always directs to the .exe of the .go file that's run
		println("Usage (from root dir): go run node/node.go <node_id> :<port_num> <peer_1> [<peer_2> ...]")
		return
	}

	// Parse args to create a new Node
	node_id, _ := strconv.Atoi(os.Args[1])
	port := os.Args[2]
	peers := os.Args[3:]

	// Create new Node
	node := &Node{
		node_id: node_id,
		state:   "RELEASED",
		lamport: 0,
		peers:   peers,
	}

	// temp / debug
	println(os.Args[0])
	println(port)
	println(node)
	println(peers)
}
