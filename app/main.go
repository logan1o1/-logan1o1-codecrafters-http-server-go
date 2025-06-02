package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
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

		go handleConnection(conn)
	}

}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	reqLine, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading request line: %v\n", err)
		return
	}

	fields := strings.Fields(reqLine)
	if len(fields) < 2 || (len(fields) > 2 && fields[2] != "HTTP/1.1") {
		fmt.Fprintf(os.Stderr, "Malformed or unsupported request: %q\n", reqLine)
		return
	}

	method, path := fields[0], fields[1]
	fmt.Printf("Received request: %s %s\n", method, path)

	headers := parseHeaders(reader)
	fmt.Println("Parsed headers:", headers)

	response := handleRequest(method, path, headers, reader)
	if _, err := conn.Write(response); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing response: %v\n", err)
	}

	fmt.Println("Response sent successfully")
	fmt.Println("Connection closed")
}

func parseHeaders(reader *bufio.Reader) map[string]string {
	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading header line: %v\n", err)
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Malformed header: %q\n", line)
			continue
		}
		headers[strings.ToLower(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])
	}
	return headers
}

func handleRequest(method, path string, headers map[string]string, reader *bufio.Reader) []byte {
	switch {
	case method == "GET" && path == "/":
		return []byte("HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n")

	case method == "GET" && strings.HasPrefix(path, "/echo/"):
		param := strings.TrimPrefix(path, "/echo/")
		if strings.Contains(headers["accept-encoding"], "gzip") {
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			_, err := gz.Write([]byte(param))
			gz.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Gzip write error: %v\n", err)
				return []byte("HTTP/1.1 500 Internal Server Error\r\nContent-Length: 0\r\n\r\n")
			}
			headers := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Encoding: gzip\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n", buf.Len())
			return append([]byte(headers), buf.Bytes()...)
		}
		return []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(param), param))

	case method == "GET" && path == "/user-agent":
		ua := headers["user-agent"]
		return []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(ua), ua))

	case method == "GET" && strings.HasPrefix(path, "/files/"):
		filePath := filepath.Join(directory, strings.TrimPrefix(path, "/files/"))
		file, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "File not found: %v\n", err)
			return []byte("HTTP/1.1 404 Not Found\r\nContent-Length: 0\r\n\r\n")
		}
		return []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(file), file))

	case method == "POST" && strings.HasPrefix(path, "/files/"):
		filePath := filepath.Join(directory, strings.TrimPrefix(path, "/files/"))
		lengthStr, ok := headers["content-length"]
		if !ok {
			return []byte("HTTP/1.1 411 Length Required\r\nContent-Length: 0\r\n\r\n")
		}
		var contentLength int
		_, err := fmt.Sscanf(lengthStr, "%d", &contentLength)
		if err != nil || contentLength <= 0 {
			return []byte("HTTP/1.1 400 Bad Request\r\nContent-Length: 0\r\n\r\n")
		}
		body := make([]byte, contentLength)
		if _, err := reader.Read(body); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading body: %v\n", err)
			return []byte("HTTP/1.1 500 Internal Server Error\r\nContent-Length: 0\r\n\r\n")
		}
		if err := os.WriteFile(filePath, body, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			return []byte("HTTP/1.1 500 Internal Server Error\r\nContent-Length: 0\r\n\r\n")
		}
		return []byte("HTTP/1.1 201 Created\r\nContent-Length: 0\r\n\r\n")
	}

	return []byte("HTTP/1.1 404 Not Found\r\nContent-Length: 0\r\n\r\n")
}
