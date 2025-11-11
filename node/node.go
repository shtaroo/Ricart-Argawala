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

	mu              sync.Mutex
	numReplies      int
	deferredReplies []int32
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

func (node *Node) DoesEverythinginator() {
	node.state = "WANTED"
	node.lamport++ // Lamport increase on state-change

	// Send requests to peers
	for _, peer := range node.peers {
		go func() {
			// Connect to peer
			conn, err := grpc.NewClient(peer, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("Failed to connect to %s: %v", peer, err)
				return
			}
			defer conn.Close()

			// Send request message to peer
			peer_node := pb.NewRicartArgawalaClient(conn)
			peer_node.RequestCriticalSection(context.Background(), &pb.Request{
				NodeId:  node.id,
				Lamport: node.lamport,
			})
		}()
	}

	// Wait for replies from all peers
	for {
		node.mu.Lock()
		if node.numReplies >= len(node.peers) {
			break
		}
		node.mu.Unlock()

		time.Sleep(100 * time.Millisecond) // Cap the loop to run 10 times per sec instead of thousands
	}

	// Enter critical section
	log.Printf("🚨💀 Node %d entered the critical section 💀🚨", node.id)

	// Pretend something cool is happening in the critical section...
	time.Sleep(3 * time.Second)

	// Leave critical section
	log.Printf("🚶💨 Node %d left the critical section 🚶💨", node.id)

	// Reply to all in queue
	node.mu.Lock()
	node.state = "RELEASED"
	for id := range node.deferredReplies {
		go node.SendReply(int32(id))
	}
	node.deferredReplies = []int32{}
	node.mu.Unlock()
}

func (node *Node) RequestCriticalSection(ctx context.Context, in *pb.Request) (*pb.Empty, error) {
	node.mu.Lock()
	defer node.mu.Unlock()

	// Correct lamport timestamp
	node.lamport = max(node.lamport, in.Lamport) + 1

	// 'On receive' part from lecture 7 slide 15/52
	if (node.state == "HELD") || (node.state == "WANTED" && node.IsLessThanPeer(in.Lamport, in.NodeId)) {
		// Queue reply
		node.deferredReplies = append(node.deferredReplies, in.NodeId)
	} else {
		// Direct reply
		go node.SendReply(in.NodeId)
	}

	return &pb.Empty{}, nil
}

func (node *Node) SendReply(peer_id int32) {
	// ChatGPT black magic
	if int(peer_id) >= len(node.peers)+1 || peer_id == node.id {
		return
	}

	// Connect to peer
	peer := node.peers[peer_id-1]
	conn, err := grpc.NewClient(peer, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect to %s: %v", peer, err)
		return
	}
	defer conn.Close()

	// Respond to peer
	peer_node := pb.NewRicartArgawalaClient(conn)
	peer_node.RespondToCSRequest(context.Background(), &pb.Response{NodeId: node.id, Lamport: node.lamport})
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

func (node *Node) RespondToCSRequest(ctx context.Context, in *pb.Response) (*pb.Empty, error) {
	node.mu.Lock()
	defer node.mu.Unlock()

	node.numReplies++ // Increment number of replies received
	return &pb.Empty{}, nil
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
		node.DoesEverythinginator()

		time.Sleep(5 * time.Second)
	}
}
