package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

var (
	FOUND    string = "FND"
	NOTFOUND string = "NFD"
	EXIT     string = "EXT"
)

var SEP string = "*|*"

func main() {

	if len(os.Args) < 3 {
		log.Fatal("Provide server and text to search.\n")
	}
	serverAddress := os.Args[1]
	queryString := os.Args[2]

	con, err := net.Dial("tcp", serverAddress)
	if err != nil {
		log.Fatal("Error in dialing to server.\n")
		panic(err)
	}

	fmt.Printf("String to search: %s.\n", queryString)
	fmt.Fprintf(con, queryString+"\n")
	fmt.Println("Waiting for the server response.")

	buf := bufio.NewReader(con)
	msg, _, _ := buf.ReadLine()
	// log.Printf("Server Response: %s\n", string(msg))
	command := strings.Split(string(msg), SEP)
	if command[0] == FOUND {
		fmt.Printf("%s is FOUND by Node [%s]\n", queryString, command[1])
	} else {
		fmt.Printf("%s is NOT FOUND\n", queryString)
	}
	// fmt.Println(command)
	msg, _, _ = buf.ReadLine()
	// log.Printf("Server Response 2: %s\n", string(msg))
}
