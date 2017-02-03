/*
Implements the solution to assignment 3 for UBC CS 416 2016 W2.

Usage:

Bootstrapping:
go run peer.go -b [physical-peerID] [peer-IP:port] [server ip:port]

Joining:
go run peer.go -j [physical-peerID] [peer-IP:port] [other-peer ip:port]

Example:
go run peer.go -b 0 127.0.0.1:19000 127.0.0.1:20000
go run peer.go -j 0 127.0.0.1:19001 127.0.0.1:19000

See run.sh for example running script.
*/

package main

import (
	"fmt"
	"os"
	"strconv"
	//can be deleted, only for example printing
	// "math/rand"
	// TODO
	"net/rpc"
	"net"
	"log"
	// "time"
)

//Modes of operation
const (
	BOOTSTRAP = iota
	JOIN
)

// Resource server type.
type RServer int

// Request that peer sends in call to RServer.InitSession
// Return value is int
type Init struct {
	IPaddr string // Must be set to ""
}

// Request that peer sends in call to RServer.GetResource
type ResourceRequest struct {
	SessionID int
	IPaddr    string // Must be set to ""
}

// Response that the server returns from RServer.GetResource
type Resource struct {
	Resource      string
	LogicalPeerID int
	NumRemaining  int
}

//An array of resources
type Resources []Resource

// My stuffs =======================================================

type PeerAddressAndStatus struct {
	Address string
	Status bool
}

type Peer int


type JoinRequest struct {
	MyAddress string
}

type JoinResponse struct {
	SessID int
	RServerAddress string
	// all except myIpPort
	AllPeers []PeerAddressAndStatus
	AllResources []string
}


var (
	sessionID int
	myIpPort string
	myID int
	serverIpPort string
	peerList []PeerAddressAndStatus
	resourceList []string
)


// Main workhorse method.
func main() {
	// Parse the command line args, panic if error
	mode, physicalPeerId, peerIpPort, otherIpPort, err := ParseArguments()
	if err != nil {
		panic(err)
	}

	// Example illustrating JoinPrint and FinalPrint usage:
	//
	// var res Resources
	// for i := 0; i < 5; i++ {
	// 	res = append(res, Resource{Resource: fmt.Sprintf("%d", i), LogicalPeerID: rand.Int(), NumRemaining: 5 - i})
	// }
	// JoinPrint(physicalPeerId)
	// res.FinalPrint(physicalPeerId)

	// TODO
	// if -j call Join RPC to otherIpPort, will receive sessionID, serverIpPort, peerList, resourceList

	myIpPort = peerIpPort
	myID = physicalPeerId

	done := make(chan int)

		// Set up RPC so peers can talk to each other
	go func() {
		pServer := rpc.NewServer()
		p := new(Peer)
		pServer.Register(p)

		l, err := net.Listen("tcp", myIpPort)
		checkError("", err, true)
		fmt.Println("Succeded in setting up an rpc server on address: ", myIpPort)

		for {
			conn, err := l.Accept()
			checkError("", err, false)
			go pServer.ServeConn(conn)
		}
	}()	


	// if -b get session id
	if mode == BOOTSTRAP {
		serverIpPort = otherIpPort
		peerList = append(peerList, PeerAddressAndStatus{myIpPort, true})

		raddr, err := net.ResolveTCPAddr("tcp", serverIpPort)
		checkError("Connecting to server: ", err, true)

		// Connection to the server
		serverConn, err := net.DialTCP("tcp", nil, raddr)
		checkError("Dialing the server: ", err, true)
		rServerConn := rpc.NewClient(serverConn)

		in := Init{
			IPaddr:   "",
		}
		err = rServerConn.Call("RServer.InitSession", in, &sessionID)
		checkError("", err, false)

		// to allow other peers on same machine to connect at different times to RServer
		// Having a connection with RServer seems to be required to be synchronously
		rServerConn.Close()

		fmt.Println("The RServer.InitSession responded with sessionID: ", sessionID)

		//TODO get resource, manage, delegate if exists peer, else wait till peer exists.
	} else {
		// TODO Join a peer

		var joinResp JoinResponse
		joinReq := JoinRequest{myIpPort}

		client, err := rpc.Dial("tcp", otherIpPort)
		checkError("rpc.Dial in Joining", err, false)
		err = client.Call("Peer.Join", joinReq, &joinResp)
		checkError("client.Call(Peer.Join: ", err, false)

		sessionID = joinResp.SessID
		serverIpPort = joinResp.RServerAddress
		peerList = joinResp.AllPeers
		peerList = append(peerList, PeerAddressAndStatus{myIpPort, true})
		resourceList = joinResp.AllResources

		fmt.Println("successfully called Peer.Join to: ", otherIpPort)
		fmt.Println("After Join, my sessionID: ", sessionID, " my serverIpPort: ", serverIpPort)
		fmt.Println(" my peerList: ", peerList, " my resourceList: ", resourceList)
	}

	

	// // for testing rpcs...
	// if mode == JOIN {
	// 	var reply = false
	// 	// req := true
	// 	id := myID
	// 	fmt.Println("About to rpc.Dial address: ", otherIpPort)
	// 	client, err := rpc.Dial("tcp", otherIpPort)
	// 	checkError("rpc.Dial: to address: ", err, false)
	// 	// err = client.Call("Peer.Exit", req, &reply)
	// 	err = client.Call("Peer.Ping", id, &reply)
	// 	checkError("Peer.Exit: ", err, false)
	// 	fmt.Println("succeded in making the Peer.Ping rpc call...!!!!")

	// 	// fmt.Println("succeded in making the Peer.Exit rpc call...!!!!")
	// }



	// blocks while threads exist alive
	<-done
	fmt.Println("Bye bye !!   :)")
}



func (p *Peer) Ping(PeerId int, reply *bool) error {
	*reply = true
	fmt.Println("I received a Ping rpc call from peerId: ", PeerId)
	return nil
}


func (p *Peer) Exit(Request bool, reply *bool) error {	
	os.Exit(0)
	*reply = true
	return nil
}

func (p *Peer) Join(JReq JoinRequest, JResp *JoinResponse) error {
	// TODO have to lock stuff???  Probably.
	fmt.Println("Before updating peerList when receive Peer.Join rpc, peerList: ", peerList)
	*JResp = JoinResponse{sessionID, serverIpPort, peerList, resourceList}
	peerList = append(peerList, PeerAddressAndStatus{JReq.MyAddress, true})
	fmt.Println("After updating peerList when receive Peer.Join rpc, peerList: ", peerList)
	// TODO broadcast new peerList to all peers (except this one??)
	return nil
}

// TODO Implement Join(), AddPeer(), AddResource(), GetNextResource()


func checkError(msg string, err error, exit bool) {
	if err != nil {
		log.Println(msg, err)
		if exit {
			os.Exit(-1)
		}
	}
}

// Parses the command line arguments, two cases are valid
func ParseArguments() (mode int, physicalPeerId int, peerIpPort string, serverOrJoinerIpPort string, err error) {
	args := os.Args[1:]
	if len(args) != 4 {
		err = fmt.Errorf("Please supply 4 command line arguments as seen in the spec http://www.cs.ubc.ca/~bestchai/teaching/cs416_2016w2/assign3/index.html")
		return
	}
	switch args[0] {
	case "-b":
		mode = BOOTSTRAP
	case "-j":
		mode = JOIN
	default:
		err = fmt.Errorf("Valid modes are bootstrapping -b and joining -j not %s", args[0])
		return
	}
	physicalPeerId, err = strconv.Atoi(args[1])
	if err != nil {
		err = fmt.Errorf("unable to parse physical peer id (argument 2) please supply an integer Error:%s", err.Error())
	}
	peerIpPort = args[2]
	serverOrJoinerIpPort = args[3]
	return
}

/////////////////// Use functions below for state notifications (DO NOT MODIFY).

func JoinPrint(physicalPeerId int) {
	fmt.Printf("JOINED: using %d\n", physicalPeerId)
}

//
func (r Resource) String() string {
	return fmt.Sprintf("Resource: %s\tLogicalPeerID: %d\tNumRemaining: %d", r.Resource, r.LogicalPeerID, r.NumRemaining)
}

//
func (r Resources) String() (rString string) {
	for _, resource := range r {
		rString += resource.String() + "\n"
	}
	return
}

//
func (r Resources) FinalPrint(physicalPeerId int) {
	for _, resource := range r {
		fmt.Printf("ALLOC: %d %d %s\n", physicalPeerId, resource.LogicalPeerID, resource.Resource)
	}
}