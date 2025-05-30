package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// Ensures gofmt doesn't remove the "net" and "os" imports above (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

const CRLF = "\r\n"

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")
	// Uncomment this block to pass the first stage
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	defer l.Close()

	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	buf := make([]byte, 1024)
	_, err = conn.Read(buf)
	if err != nil {
		fmt.Println("Error accepting connection: ", err)
	}

	fmt.Println("Accepted connection from: ", conn.RemoteAddr())
	req := string(buf)
	lines := strings.Split(req, CRLF)
	path := strings.Split(lines[0], " ")[1]
	fmt.Println(path)

	var res string

	if path == "/" {
		res = "HTTP/1.1 200 OK\r\n\r\n"
	} else if path[:5] == "/echo" {
		dynamic_path := strings.Split(path, "/")[len(strings.Split(path, "/"))-1]
		res = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(dynamic_path), dynamic_path)
	} else {
		res = "HTTP/1.1 404 Not Found\r\n\r\n"
	}

	fmt.Println(res)
	conn.Write([]byte(res))
	fmt.Println("conn remote address", conn.RemoteAddr())
}
