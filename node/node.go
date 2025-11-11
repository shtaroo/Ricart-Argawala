package main

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "module/proto"
)

// Node constructor
type Node struct {
	pb.UnimplementedRicartArgawalaServer

	id      int32
	state   string
	lamport int32
	peers   []string

	mu         sync.Mutex
	numReplies int
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

func (node *Node) SendRequestsToPeers() {
	node.state = "WANTED"
	node.lamport++ // Lamport increase on state-change

	for _, peer := range node.peers {
		// Send request message
		go func() {
			conn, err := grpc.NewClient(peer, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("Failed to connect to %s: %v", peer, err)
				return
			}
			defer conn.Close()

			peer_node := pb.NewRicartArgawalaClient(conn)
			peer_node.RequestCriticalSection(context.Background(), &pb.Request{
				NodeId:  node.id,
				Lamport: node.lamport,
			})
		}()
	}

	// Wait for len(peers) replies
	//TODO: STILL WIP
}

func (node *Node) RequestCriticalSection(ctx context.Context, in *pb.Request) (*pb.Empty, error) {
	node.mu.Lock()
	defer node.mu.Unlock()

	node.lamport = max(node.lamport, in.Lamport) + 1

	if (node.state == "HELD") || (node.state == "WANTED" && node.IsLessThanPeer(in.Lamport, in.NodeId)) {
		//queue req
	} else {
		//reply to req
	}

	return &pb.Empty{}, nil
}

func (node *Node) IncrementNumberOfReplies() {
	node.mu.Lock()
	node.numReplies++
	node.mu.Unlock()
}

func (node *Node) IsLessThanPeer(peerLamport int32, peerId int32) bool {
	if node.lamport < peerLamport {
		return true
	} else if node.lamport == peerLamport {
		return node.id < peerId
	} else {
		return false
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
	node_id32 := int32(node_id)
	port := os.Args[2]
	peers := os.Args[3:]

	// Create new Node
	node := &Node{
		id:      node_id32,
		state:   "RELEASED",
		lamport: 0,
		peers:   peers,
	}

	// Start node-server (start listening for requests)
	node.StartServer(port)

	// Wait 20 seconds so there is time to start all Nodes
	time.Sleep(20 * time.Second)

	// Continuously attempt to enter critical section
	for {
		node.SendRequestsToPeers()
		//node.waitForReplies()
		//node.enterCriticalSection()

		time.Sleep(5 * time.Second)
	}
}
