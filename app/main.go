package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
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
	defer conn.Close()

	reader := bufio.NewReader(conn)

	requestLine, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading request line: %v\n", err)
		return
	}

	fields := strings.Fields(requestLine)
	if len(fields) < 2 {
		fmt.Fprintf(os.Stderr, "Malformed request: %q\n", requestLine)
		return
	}

	method, path := fields[0], fields[1]
	fmt.Printf("Received request: %s %s\n", method, path)

	if len(fields) > 2 && fields[2] != "HTTP/1.1" {
		fmt.Printf("Unsupported HTTP version: %s\n", fields[2])
		return
	}

	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading header line: %v\n", err)
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Malformed header line: %q\n", line)
			continue
		}

		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		headers[key] = value
	}

	fmt.Println("Parsed headers:", headers)

	var response string

	switch {
	case method == "GET" && path == "/":
		response = "HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n"

	case method == "GET" && strings.HasPrefix(path, "/echo/"):
		param := strings.TrimPrefix(path, "/echo/")
		acceptEncodingHeader := headers["accept-encoding"]
		if acceptEncodingHeader != "" && strings.Contains(acceptEncodingHeader, "gzip") {
			response = fmt.Sprintf(
				"HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Encoding: gzip\r\nContent-Length: %d\r\n\r\n%s",
				len(param), param,
			)
		} else {
			response = fmt.Sprintf(
				"HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
				len(param), param,
			)
		}

	case method == "GET" && path == "/user-agent":
		ua := headers["user-agent"]
		response = fmt.Sprintf(
			"HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
			len(ua), ua,
		)

	case method == "GET" && strings.HasPrefix(path, "/files/"):
		fileName := strings.TrimPrefix(path, "/files/")
		fullPath := filepath.Join(directory, fileName)
		file, err := os.Open(fullPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file %s: %v\n", fullPath, err)
			response = "HTTP/1.1 404 Not Found\r\nContent-Length: 0\r\n\r\n"
		} else {
			defer file.Close()

			fileInfo, err := file.Stat()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting file info for %s: %v\n", fullPath, err)
				response = "HTTP/1.1 500 Internal Server Error\r\nContent-Length: 0\r\n\r\n"
			} else {
				response = fmt.Sprintf(
					"HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n",
					fileInfo.Size(),
				)
				fileContent := make([]byte, fileInfo.Size())
				if _, err := file.Read(fileContent); err != nil {
					fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", fullPath, err)
					response = "HTTP/1.1 500 Internal Server Error\r\nContent-Length: 0\r\n\r\n"
				} else {
					response += string(fileContent)
				}
			}
		}

	case method == "POST" && strings.HasPrefix(path, "/files/"):
		fileName := strings.TrimPrefix(path, "/files/")
		fullPath := filepath.Join(directory, fileName)

		lengthStr, ok := headers["content-length"]
		if !ok {
			fmt.Fprintf(os.Stderr, "Content-Length header missing\n")
			response = "HTTP/1.1 411 Length Required\r\nContent-Length: 0\r\n\r\n"
			break
		}

		var contentLength int
		_, err := fmt.Sscanf(lengthStr, "%d", &contentLength)
		if err != nil || contentLength <= 0 {
			fmt.Fprintf(os.Stderr, "Invalid Content-Length header: %s\n", lengthStr)
			response = "HTTP/1.1 400 Bad Request\r\nContent-Length: 0\r\n\r\n"
			break
		}

		body := make([]byte, contentLength)
		_, err = reader.Read(body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
			response = "HTTP/1.1 500 Internal Server Error\r\nContent-Length: 0\r\n\r\n"
			break
		}

		file, err := os.Create(fullPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating file %s: %v\n", fullPath, err)
			response = "HTTP/1.1 500 Internal Server Error\r\nContent-Length: 0\r\n\r\n"
			break
		}
		defer file.Close()

		_, err = file.Write(body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to file %s: %v\n", fullPath, err)
			response = "HTTP/1.1 500 Internal Server Error\r\nContent-Length: 0\r\n\r\n"
			break
		}
		response = "HTTP/1.1 201 Created\r\nContent-Length: 0\r\n\r\n"

	default:
		response = "HTTP/1.1 404 Not Found\r\nContent-Length: 0\r\n\r\n"
	}

	if _, err := conn.Write([]byte(response)); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing response: %v\n", err)
		return
	}

	fmt.Println("Response sent successfully")
	fmt.Println("Connection closed")
}
