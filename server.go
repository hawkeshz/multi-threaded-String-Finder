package main

import (
	"bufio"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// server msg protocols
var (
	HEARTBEAT   string = "HBT"
	DUPLICATE   string = "DUP"
	OKAY        string = "OKY"
	ALIVE       string = "ALV"
	SEARCH      string = "SRH"
	ABORT       string = "ABT"
	FOUND       string = "FND"
	NOTFOUND    string = "NFD"
	EXIT        string = "EXT"
	OKABORT     string = "OBT"
	LOADFILE    string = "LOD"
	UNLOADFILE  string = "ULD"
	MEMORYFILES string = "MMF"
)

// seperator
var SEP string = "*|*"

// node structure
type Node struct {
	id          int
	con         net.Conn
	ch          chan string
	chunkFiles  []string
	memoryFiles []string
}

type Client struct {
	con         net.Conn
	ch          chan string
	recvch      chan string
	id          string
	rescounter  int
	searchquery []byte
}

func (node Node) ReadFromNode(ch chan<- string, noderemchan chan<- Node) {
	bufc := bufio.NewReader(node.con)
	for {
		line, _, err := bufc.ReadLine()
		if err != nil {
			noderemchan <- node
			break
		}
		ch <- fmt.Sprintf("%d%s%s", node.id, SEP, string(line))
	}
}

func (node Node) writeToNodeChan(ch <-chan string) {
	for msg := range ch {
		// fmt.Printf("Sending to node: %s\n", msg)
		node.con.Write([]byte(msg + "\n"))
	}
}

func (client Client) writeToClientChan(ch <-chan string) {
	for msg := range ch {
		// fmt.Printf("Sending to cient: %s\n", msg)
		client.con.Write([]byte(msg + "\n"))
		if msg == EXIT {
			delete(Clients, client.id)
		}
	}
}

// print node
func (node Node) print() {
	fmt.Printf("{Node id: %d, files, %s, memory files: %s}\n", node.id, node.chunkFiles, node.memoryFiles)
}

// to keep track of nodes
var Nodes = make(map[int]Node)

// to keep track of clients
var Clients = make(map[string]Client)

// to take actions on messages from node
func NodeMsgAction(msg string, nodeaddchan <-chan Node, noderemchan chan Node) {
	fmt.Printf("Message in NodeAction is %s\n", msg)
	tokens := strings.Split(msg, SEP)
	command := tokens[1][0:3]
	node, _ := strconv.Atoi(tokens[0])
	switch command {
	case FOUND:
		log.Printf("%s found by node: %d -> %s\n", tokens[2], node, msg)
		for _, anode := range Nodes {
			if anode.id != node {
				anode.ch <- ABORT
			}
		}
		reqClientId := tokens[3]
		aclient := Clients[reqClientId]
		Clients[reqClientId] = aclient
		aclient.recvch <- fmt.Sprintf("%s%s%d", FOUND, SEP, node)
		aclient.recvch <- EXIT

	case NOTFOUND:
		log.Printf("%s not found by node: %d -> %s\n", tokens[2], node, msg)
		reqClientId := tokens[3]
		aclient := Clients[reqClientId]
		aclient.rescounter += 1
		// log.Println("Counter is incremented", aclient.rescounter, string(aclient.searchquery))
		Clients[reqClientId] = aclient
		if len(Nodes) == aclient.rescounter {
			// fmt.Printf("Sending  NOT FOUND to client\n")
			aclient.recvch <- fmt.Sprintf("%s%s0", NOTFOUND, SEP)
			aclient.recvch <- EXIT
		}

	case MEMORYFILES:
		log.Printf("Nodes %d memory files: %s\n", node, tokens)
		// Nodes[node].memoryFiles = strings.Split(tokens[1], "--")
		mnode := Nodes[node]
		mnode.memoryFiles = strings.Split(tokens[2], "--")
		fmt.Println(mnode.memoryFiles)
		Nodes[node] = mnode

	case OKABORT:
		log.Printf("OK ABORT signal from node: %d\n", node)

	case EXIT:
		nodeId, _ := strconv.Atoi(tokens[2])
		noderemchan <- Nodes[nodeId]

	default:
		log.Println("Default msg: %s\n", msg)
	}
}

// handle file after node killed
func handleKilledNode(node Node) {
	fileHandled := false // true if file is handled
	// looking for files which are in current node memory
	for _, f := range node.memoryFiles {
		// looking for above file in all nodes
		for id, anode := range Nodes {
			if fileHandled {
				break
			}
			if node.id != id { // don't look if the node is same node
				// looking over files which are in nodes dir
				for _, chunkF := range anode.chunkFiles {
					if fileHandled {
						break
					}
					if chunkF == f { // if same file is in present in another node
						memFCounter := 0
						// if that file is already in another node memory or not
						for _, memF := range anode.memoryFiles {
							if memF != f {
								memFCounter += 1
							}
						}
						// it means file is not in memory but dir
						// so load file
						if memFCounter == len(anode.memoryFiles) {
							anode.ch <- LOADFILE + SEP + f
							fileHandled = true
							break
						}
					}
				}
			}
		}
	}
	delete(Nodes, node.id)
}

// handle heartbeat
// func handlrHeartbeat(tim int) {
// 	for nID := range Nodes {
//
// 	}
// }

// handle messages from node
func handleNodeMessages(nodemsgchan <-chan string, nodeaddchan <-chan Node, noderemchan chan Node) {
	for {
		select {
		case msg := <-nodemsgchan:
			// log.Printf("Node > %s\n", msg)
			go NodeMsgAction(msg, nodeaddchan, noderemchan)
		case node := <-nodeaddchan:
			log.Printf("New node %d is connected, address: %s\n", node.id, node.con.RemoteAddr().String())
			Nodes[node.id] = node
		case node := <-noderemchan:
			log.Printf("Node %d is disconnected, address %s\n", node.id, node.con.RemoteAddr().String())
			handleKilledNode(node)
		}
	}
}

// return true if node is present with given id
func isNodePresent(id int) bool {
	for ID := range Nodes {
		if ID == id {
			return true
		}
	}
	return false
}

// find unions in the list
func getUnions(s1 []string, s2 []string) []string {
	var unions []string
	for _, a1 := range s1 {
		for _, a2 := range s2 {
			if a1 == a2 {
				unions = append(unions, a1)
			}
		}
	}
	return unions
}

// files which are not loaded yet
func getAbsentFiles(newFiles []string) []string {
	absentFiles := make(map[string]int) // to store absent files in map
	for _, nf := range newFiles {       // over each new file
		isPresent := false
		for ID := range Nodes { // looking in each node
			if isPresent { // if present don't go further
				break
			}
			// in each node loop over each file in memory
			for _, pf := range Nodes[ID].memoryFiles {
				if nf == pf { // if new file is present in memory
					isPresent = true
					break
				}
			}
		}
		if !isPresent { // new file is not present file
			absentFiles[nf] = 0
		}
	}
	var absFiles []string
	for k := range absentFiles { // make array of absent files
		absFiles = append(absFiles, k) // appending each absent file
	}
	fmt.Println("List of new files which are not in memory: ", absFiles)
	return absFiles
}

// handle new node files
func manageNewNodeFiles(node *Node, buf *bufio.Reader) {
	absentFiles := getAbsentFiles(node.chunkFiles)
	if len(absentFiles) > 0 {
		node.con.Write([]byte(fmt.Sprintf("%s%s%s\n", LOADFILE, SEP, strings.Join(absentFiles, "--"))))
		msg, _, _ := buf.ReadLine()
		command := string(msg)
		fmt.Println("In Manage: ", command)
		if command[:3] == MEMORYFILES {
			tokens := strings.Split(command, SEP)
			node.memoryFiles = strings.Split(tokens[1], "--")
		}
	} else {
		for nodeID := range Nodes {
			if len(node.memoryFiles) >= len(Nodes[nodeID].memoryFiles)/2 {
				continue
			}
			if len(Nodes[nodeID].chunkFiles) > 1 {
				unionFiles := getUnions(node.chunkFiles, Nodes[nodeID].memoryFiles)
				noFiletoShift := 1
				if len(unionFiles) > 1 {
					noFiletoShift = len(Nodes[nodeID].memoryFiles) / len(unionFiles)
				}
				// TODO: shifting of files
				Nodes[nodeID].ch <- UNLOADFILE + SEP + strings.Join(unionFiles[:noFiletoShift], "--")
				node.con.Write([]byte(LOADFILE + SEP + strings.Join(unionFiles[:noFiletoShift], "--") + "\n"))
				msg, _, _ := buf.ReadLine()
				command := string(msg)
				fmt.Println("In Manage: ", command)
				if command[:3] == MEMORYFILES {
					tokens := strings.Split(command, SEP)
					node.memoryFiles = strings.Split(tokens[1], "--")
				}
			}
		}
	}
}

// handle connection from nodes
func handleNodeConnection(c net.Conn, nodemsgchan chan<- string, nodeaddchan chan<- Node, noderemchan chan<- Node) {
	defer c.Close()

	// new node
	var node Node
	node.con = c
	node.ch = make(chan string)
	// reading slave rank
	buf := bufio.NewReader(c)
	msg, _, _ := buf.ReadLine()
	node.id, _ = strconv.Atoi(string(msg))
	// check if node is already present with this id
	if isNodePresent(node.id) {
		c.Write([]byte(DUPLICATE + "\n"))
		c.Close()
		return
	}
	c.Write([]byte(OKAY + "\n"))
	// reading slave chunks
	msg, _, _ = buf.ReadLine()
	node.chunkFiles = strings.Split(string(msg), "--")

	// deal with new node files
	manageNewNodeFiles(&node, buf)

	// reading slave memory chunks
	// msg, _, _ = buf.ReadLine()
	// node.memoryFiles = strings.Split(string(msg), "--")
	// node.print()

	// adding node to channel
	nodeaddchan <- node
	go node.ReadFromNode(nodemsgchan, noderemchan)
	defer func() {
		log.Printf("Node %d is disconnected. Connection closed with %s\n", node.id, node.con.RemoteAddr().String())
		noderemchan <- node
	}()
	node.writeToNodeChan(node.ch)
}

func acceptNodeConnections(ln net.Listener, nodemsgchan chan<- string, nodeaddchan chan<- Node, noderemchan chan<- Node) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleNodeConnection(conn, nodemsgchan, nodeaddchan, noderemchan)
	}
}

func handleClientConnection(c net.Conn) {
	defer c.Close()

	var client Client
	client.con = c
	client.ch = make(chan string)
	client.recvch = make(chan string)
	client.rescounter = 0

	buf := bufio.NewReader(c)
	msg, _, _ := buf.ReadLine()

	client.searchquery = make([]byte, 0)
	client.searchquery = msg
	client.id = genClientId()
	Clients[client.id] = client

	makeSearch(client)
	client.writeToClientChan(client.recvch)
}

func makeSearch(client Client) {
	readyQuery := SEARCH + SEP + string(client.searchquery) + SEP + client.id
	for _, anode := range Nodes {
		go func(mch chan<- string) {
			mch <- readyQuery
		}(anode.ch)
	}
}

func acceptClientConnections(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
			continue
		}
		log.Printf("New client is connected, addresss: %s\n", conn.RemoteAddr().String())
		go handleClientConnection(conn)
	}
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Provide port for nodes and client")
	}
	portForNodes := os.Args[1]
	portForClient := os.Args[2]

	// necessary channels
	nodemsgchan := make(chan string)
	nodeaddchan := make(chan Node)
	noderemchan := make(chan Node)

	go handleNodeMessages(nodemsgchan, nodeaddchan, noderemchan)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%s", portForNodes))
	if err != nil {
		log.Fatal("Error in listening for nodes.")
		log.Fatal(err)
		os.Exit(-1)
	}
	log.Printf("server is listening for nodes on port %s.\n", portForNodes)
	defer ln.Close()
	go acceptNodeConnections(ln, nodemsgchan, nodeaddchan, noderemchan)
	// time.Sleep(time.Second * 10)

	// handling clients
	clientLn, errCl := net.Listen("tcp", fmt.Sprintf(":%s", portForClient))
	if errCl != nil {
		log.Fatal("Error in listening for clients.\n")
		panic(errCl)
	}
	log.Printf("Server is listening for clients on port %s.\n", portForClient)
	defer clientLn.Close()
	go acceptClientConnections(clientLn)

	// handling http clients
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/nodes", getNodesHandler)
	log.Printf("Server is listening for http on port %s.\n", "8080")
	http.ListenAndServe(":8080", nil)
}

func getNodesJsonStr() string {
	var nodesInfo string
	nodesInfo = "[ "
	for _, anode := range Nodes {
		nodesInfo += fmt.Sprintf("{\"name\": \"%d\", \"chunks\": \"%s\", \"memory\": \"%s\"}", anode.id, anode.chunkFiles, anode.memoryFiles)
		nodesInfo += ","
	}
	nodesInfo = nodesInfo[:len(nodesInfo)-1] + "]"
	return nodesInfo
}

func getNodesHandler(res http.ResponseWriter, req *http.Request) {
	nodesInfo := getNodesJsonStr()
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(nodesInfo))
}

func rootHandler(res http.ResponseWriter, req *http.Request) {
	editTemplate, err := template.ParseFiles("./view/index.html")
	if err != nil {
		log.Fatal("Error in parsing template file")
		log.Fatal(err)
	}
	editTemplate.Execute(res, &Page{title: "CDS-Project"})
}

func searchHandler(res http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		fmt.Fprintf(res, "ParseForm() err: %v", err)
		return
	}
	query := req.FormValue("query")

	var client Client
	//TODO: make something but not nil
	client.con = nil
	client.ch = make(chan string)
	client.recvch = make(chan string)
	client.rescounter = 0
	client.searchquery = make([]byte, 0)
	client.searchquery = []byte(query)
	client.id = genClientId()
	Clients[client.id] = client

	makeSearch(client)

	var searchResult string
	searchResult = <-client.recvch
	resultTokens := strings.Split(searchResult, SEP)
	searchResult = <-client.recvch

	delete(Clients, client.id)

	result := 0
	node := 0
	if resultTokens[0] == FOUND {
		result = 1
		node, _ = strconv.Atoi(resultTokens[1])
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(fmt.Sprintf("{\"result\": %d, \"node\": \"%d\"}", result, node)))
}

type Page struct {
	title string
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func genClientId() string {
	return RandString(13)
}
