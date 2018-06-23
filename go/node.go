package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

// server msg protocols
var (
	HEARTBEAT   string = "HBT"
	DUPLICATE   string = "DUP"
	ALIVE       string = "ALV"
	SEARCH      string = "SRH"
	ABORT       string = "ABT"
	OKABORT     string = "OBT"
	FOUND       string = "FND"
	NOTFOUND    string = "NFD"
	FREE        string = "FRI"
	NOTFREE     string = "NFR"
	EXIT        string = "EXT"
	LOADFILE    string = "LOD"
	UNLOADFILE  string = "ULD"
	MEMORYFILES string = "MMF"
)

// seperator
var SEP string = "*|*"

// free or busy status
var STATUS string = FREE

// rank of node
var Rank int

// data content
var DATA [][]string

// keep track of files which loaded in memory
var filesInMem []string

func checkError(e error) {
	if e != nil {
		log.Fatal(e)
		panic(e)
	}
}

func loadFile(filename string, id int) []string {
	filePath := fmt.Sprintf("./data/node_%d/%s", id, filename)

	//checking file size
	fileStat, e := os.Stat(filePath)
	checkError(e)
	fileSize := fileStat.Size()
	log.Printf("Node %d opening file of size %d MB\n", id, fileSize/1024/1024)

	file, err := os.Open(filePath)
	checkError(err)
	defer file.Close()

	// buffer to load data in
	data := make([]byte, fileSize)
	bytesStream, e1 := file.Read(data)
	checkError(e1)
	log.Printf("Node %d streamed %d MB\n", id, bytesStream/1024/1024)

	dataS := string(data)
	flines := make([]string, 0)
	flines = strings.Split(dataS, "\n")

	return flines
}

func searchText(text string) int {
	// idx := sort.SearchStrings(data, text)
	// fmt.Println("Data length: ", len(DATA))
	idx := -1
	stopSearch := false
	for dataIdx := range DATA {
		if stopSearch {
			break
		}
		idx = len(DATA[dataIdx])
		for i_, val := range DATA[dataIdx] {
			if STATUS == NOTFREE {
				if val == text {
					idx = i_
					stopSearch = true
					break
				}
			} else {
				idx = -2
				stopSearch = true
				break
			}
		}
		if idx == len(DATA[dataIdx]) {
			idx = -1
		}
	}
	return idx
}

func delFromSlice(s []string, i int) []string {
	return append(s[:i], s[i+1:]...)
}

func delFromDATA(i int) {
	DATA = append(DATA[:i], DATA[i+1:]...)
}

func msgAction(msg string, sendmsgchan chan string) {
	command := msg[:3]
	var res string
	if command == HEARTBEAT {
		res = ALIVE
	} else if command == SEARCH {
		tokens := strings.Split(msg, SEP)
		searchString := tokens[1]
		clientID := tokens[2]
		// fmt.Printf("Now searching: %s, len: %d\n", searchString, len(searchString))
		STATUS = NOTFREE
		idx := searchText(searchString)
		if idx == -1 {
			log.Printf("%s not found\n", searchString)
		} else if idx == -2 {
			log.Printf("Searching for %s is terminated due to abort command\n", searchString)
		} else {
			log.Printf("%s found at index: %d\n", searchString, idx)
		}
		res = FOUND
		if idx == -1 {
			res = NOTFOUND
		}
		res += SEP + searchString
		if idx == -2 {
			return
			// res = fmt.Sprintf("%s%s%d", OKABORT, SEP, Rank)
		}
		res += SEP + clientID
	} else if command == LOADFILE {
		fileToLoad := strings.Split(msg, SEP)[1]
		for _, newFileToLoad := range strings.Split(fileToLoad, "--") {
			filesInMem = append(filesInMem, newFileToLoad)
			DATA = append(DATA, loadFile(newFileToLoad, Rank))
		}
		res = MEMORYFILES + SEP + strings.Join(filesInMem, "--")
	} else if command == UNLOADFILE {
		filesToUnload := strings.Split(strings.Split(msg, SEP)[1], "--")
		for _, fUnload := range filesToUnload {
			for mfIdx, mf := range filesInMem {
				if mf == fUnload {
					filesInMem = delFromSlice(filesInMem, mfIdx)
					delFromDATA(mfIdx)
				}
			}
		}
		res = MEMORYFILES + SEP + strings.Join(filesInMem, "--")
	} else if command == ABORT {
		log.Println("Abort command received from server")
		STATUS = FREE
		return
	}
	log.Printf("In Action res: %s\n", res)
	res = res + "\n"
	sendmsgchan <- res
}

func handleMessages(id int, con net.Conn, sendmsgchan chan string, recvmsgchan <-chan string) {
	for {
		select {
		case msg := <-sendmsgchan:
			// log.Printf("Node %d sending data: %s\n", id, msg)
			con.Write([]byte(msg))
		case msg := <-recvmsgchan:
			log.Printf("Node %d received data: %s\n", id, msg)
			if len(msg) < 3 {
				log.Fatal("Something is wrong. Receving invalid commands from server\n")
				continue
			}
			go msgAction(msg, sendmsgchan)
		}
	}
}

func getChunkList(id int) []string {
	files, err := ioutil.ReadDir(fmt.Sprintf("./data/node_%d/", id))
	if err != nil {
		log.Fatal("Failed to Read data files.\n", err)
		return nil
	}
	chunkList := make([]string, 0)
	for _, f := range files {
		if !f.IsDir() && len(f.Name()) >= 9 {
			chunkList = append(chunkList, f.Name())
		}
	}
	return chunkList
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Provide rank and server address")
	}
	rank, _ := strconv.Atoi(os.Args[1])
	Rank = rank
	serverAddress := os.Args[2]

	con, err := net.Dial("tcp", serverAddress)
	checkError(err)
	defer con.Close()
	log.Println("Connection with server established.")

	// channles
	sendmsgchan := make(chan string)
	recvmsgchan := make(chan string)

	// message handler
	go handleMessages(rank, con, sendmsgchan, recvmsgchan)

	// read the chunk names of node data
	chunkFiles := getChunkList(rank)
	if len(chunkFiles) == 0 {
		panic(fmt.Sprintf("No data in the node[%d] directory\n", rank))
	}

	// read from server
	buf := bufio.NewReader(con)
	var msg []byte

	con.Write([]byte(fmt.Sprintf("%d\n", rank)))
	msg, _, _ = buf.ReadLine()
	if string(msg) == DUPLICATE {
		fmt.Printf("Node with id [%d] is already present.\n", rank)
	}

	con.Write([]byte(strings.Join(chunkFiles, "--") + "\n"))
	// which replica to use by default
	// var fileChunk string
	// fileChunk = ""
	// for _, file := range chunkFiles {
	// 	if file[6:len(file)-4] == strconv.Itoa(rank) {
	// 		fileChunk = file
	// 	}
	// }
	// if fileChunk == "" {
	// 	fileChunk = chunkFiles[0]
	// }
	// filesInMem = append(filesInMem, fileChunk)
	// DATA = append(DATA, loadFile(fileChunk, rank))
	//
	// // acknowledging server which chunk is in memory
	// con.Write([]byte(strings.Join(filesInMem, "--") + "\n"))

	var cerr error
	for {
		msg, _, cerr = buf.ReadLine()
		if cerr != nil {
			con.Write([]byte(EXIT + SEP + strconv.Itoa(Rank) + "\n"))
		}
		recvmsgchan <- string(msg)
	}
}
