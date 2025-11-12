package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
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
	node.state = "WANTED" // Lecture 7 slide 15/50 'On enter do'

	// Send requests to peers. Lecture 7 slide 15/50 'multicast ‘req(T,p)’'
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
	log.Printf("Node #%d sent requests to its peers", node.id)

	// Lecture 7 slide 15/50 'wait for N-1 replies'
	for {
		node.mu.Lock()
		if node.numReplies >= len(node.peers) {
			break
		}
		node.mu.Unlock()
		time.Sleep(100 * time.Millisecond) // Cap the loop to run 10 times per sec instead of thousands
	}
	node.mu.Unlock()

	node.state = "HELD" // Lecture 7 slide 15/50 'state := HELD'

	// Enter critical section
	log.Printf("🚨💀 Node #%d entered the critical section 💀🚨", node.id)

	// Pretend something cool is happening in the critical section...
	time.Sleep(3 * time.Second)

	// Leave critical section
	log.Printf("🚶💨 Node #%d left the critical section 🚶💨", node.id)

	// Reply to all in queue. Lecture 7 slide 15/50 'On exit do'
	node.mu.Lock()
	node.state = "RELEASED" // Reset state
	node.numReplies = 0     // Reset number of received replies
	for peer_id := range node.deferredReplies {
		println(peer_id)
		node.SendReply(int32(peer_id + 1))
		log.Printf("Node #%d replied to Node #%d", node.id, peer_id)
	}
	node.deferredReplies = []int32{} // Reset deferred replies slice
	node.mu.Unlock()
}

func (node *Node) RequestCriticalSection(ctx context.Context, in *pb.Request) (*pb.Empty, error) {
	node.mu.Lock()
	defer node.mu.Unlock()

	// Correct lamport timestamp
	node.lamport = max(node.lamport, in.Lamport) + 1
	log.Printf("Node #%d corrected its Lamport timestamp to %d", node.id, node.lamport)

	// 'On receive' part from lecture 7 slide 15/50
	if (node.state == "HELD") || (node.state == "WANTED" && node.IsLessThanPeer(in.Lamport, in.NodeId)) {
		// Queue reply
		node.deferredReplies = append(node.deferredReplies, in.NodeId)
		log.Printf("Node #%d deferred replying to %d", node.id, in.NodeId)
	} else {
		// Direct reply
		go node.SendReply(in.NodeId)
		log.Printf("Node #%d replied directly to %d", node.id, in.NodeId)
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
	//logging setup
	logFile := setupLogging()
	defer logFile.Close()

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
		state:   "RELEASED", // Lecture 7 slide 15/50 'On initialisation do'
		lamport: 0,
		peers:   peers,
	}

	// Start node-server (start listening for requests)
	go node.StartServer(port)

	// Wait 10 seconds so there is time to start all Nodes
	time.Sleep(10 * time.Second)

	// Continuously attempt to enter critical section
	for {
		node.DoesEverythinginator()

		time.Sleep(5 * time.Second)
	}
}

func setupLogging() *os.File {
	//create logging magic
	logDir := filepath.Join(".", "logs")

	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		log.Fatalf("failed to create log directory: %v", err)
	}

	//create log file
	timestamp := time.Now().Format("20060102-150405")
	fileName := filepath.Join(logDir, fmt.Sprintf("log-%s.txt", timestamp))

	//open file for append or create
	logFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}

	//log both to console and file
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	return logFile
}
