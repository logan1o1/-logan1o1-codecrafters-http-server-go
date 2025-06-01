package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Ensures gofmt doesn't remove the "net" and "os" imports above (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

const CRLF = "\r\n"

var directory string

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	dirFlag := flag.String("directory", ".", "directory to serve files from")
	flag.Parse()
	directory = *dirFlag

	// Uncomment this block to pass the first stage
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go concurent(conn)
	}

}

func concurent(conn net.Conn) {
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error accepting connection: ", err)
	}

	// fmt.Println("Accepted connection from: ", conn.RemoteAddr())
	req := string(buf)
	lines := strings.Split(req, CRLF)
	path := strings.Split(lines[0], " ")[1]
	method := strings.Split(lines[0], " ")[0]
	fmt.Println(method)
	fmt.Println(path)

	var res string

	if path == "/" {
		res = "HTTP/1.1 200 OK\r\n\r\n"
	} else if path[:5] == "/echo" {
		dynamicPath := strings.Split(path, "/")[len(strings.Split(path, "/"))-1]
		res = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(dynamicPath), dynamicPath)
	} else if path == "/user-agent" {
		userAgent := lines[len(lines)-3]
		userAgentVal := strings.Split(userAgent, " ")[len(strings.Split(userAgent, " "))-1]
		res = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(userAgentVal), userAgentVal)
	} else if strings.HasPrefix(path, "/files/") {
		filename := strings.TrimPrefix(path, "/files/")
		filePath := filepath.Join(directory, filename)
		fmt.Println(filePath)

		if method == "GET" {
			file, err := os.Open(filePath)
			if err != nil {
				res = "HTTP/1.1 404 Not Found\r\n\r\n"
			} else {
				defer file.Close()
				fileContent, err := io.ReadAll(file)
				if err != nil {
					res = "HTTP/1.1 500 Internal Server Error\r\n\r\n"
				} else {
					res = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(fileContent), fileContent)
				}
			}
		} else if method == "POST" {
			headersBody := strings.SplitN(string(buf[:n]), "\r\n\r\n", 2)
			headers := strings.Split(headersBody[0], "\r\n")
			body := headersBody[1]
			contentLength := 0

			for _, line := range headers {
				if strings.HasPrefix(line, "Content-Length:") {
					lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
					contentLength, err = strconv.Atoi(lengthStr)
					if err != nil {
						res = "HTTP/1.1 400 Bad Request\r\n\r\n"
						return
					}
					break
				}
			}

			for len(body) < contentLength {
				moreBuf := make([]byte, contentLength-len(body))
				m, err := conn.Read(moreBuf)
				if err != nil {
					res = "HTTP/1.1 500 Internal Server Error\r\n\r\n"
					return
				}
				body += string(moreBuf[:m])
			}
			err := os.WriteFile(filePath, []byte(body), 0644)
			if err != nil {
				res = "HTTP/1.1 500 Internal Server Error\r\n\r\n"
			} else {
				res = "HTTP/1.1 201 Created\r\n\r\n"
			}
		}

	} else {
		res = "HTTP/1.1 404 Not Found\r\n\r\n"
	}

	fmt.Println(res)
	conn.Write([]byte(res))
	// fmt.Println("conn remote address", conn.RemoteAddr())
}
