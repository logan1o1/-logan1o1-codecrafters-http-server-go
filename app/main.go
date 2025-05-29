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

type HttpRequest struct {
	Method  string
	URL     string
	Version string
}

type HttpResponse struct {
	version string
	Status  string
	URL     string
}

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go run(conn)
	}
}

func run(conn net.Conn) {
	byt := make([]byte, 1)
	acc := make([]byte, 0)

	for {
		n, _ := conn.Read(byt)
		acc = append(acc, byt[:n]...)
		if end := strings.Contains(string(acc), "\r\n\r\n"); end {
			break
		}
	}

	// fmt.Println(string(acc))
	hreq := ParseStreamToHttpReq(acc)
	hresp := ConstructResp(hreq)
	encResp := Encode(hresp)
	fmt.Println(string(encResp))
	SendResp(conn, encResp)
}

func SeparateSequenceByLines(byt []byte) [][]byte {
	lines := make([][]byte, 0)
	line := make([]byte, 0)
	for i := 0; i < len(byt)-1; i++ {
		if string(byt[i:i+2]) != "\r\n" {
			line = append(line, byt[i])
		} else {
			lines = append(lines, line)
			line = []byte("")
		}
	}
	return lines
}

func ParseStreamToHttpReq(byt []byte) HttpRequest {
	lineStr := SeparateSequenceByLines(byt)

	var hreq HttpRequest
	splitStr := strings.Split(string(lineStr[0]), " ")
	hreq.Method = splitStr[0]
	hreq.URL = splitStr[1]
	hreq.Version = splitStr[2]
	return hreq
}

func Encode(hresp HttpResponse) []byte {
	byt := make([]byte, 0)
	byt = append(byt, []byte(hresp.version)...)
	byt = append(byt, []byte(" ")...)
	byt = append(byt, []byte(hresp.Status)...)
	byt = append(byt, []byte("\r\n\r\n")...)
	return byt
}

func SendResp(conn net.Conn, respbyt []byte) {
	conn.Write(respbyt)
}

func ConstructResp(hreq HttpRequest) HttpResponse {
	var hresp HttpResponse

	hresp.version = hreq.Version
	hresp.URL = hreq.URL
	if hreq.URL == "/" {
		hresp.Status = "200 OK"
	} else {
		hresp.Status = "404 Not Found"
	}
	fmt.Printf("ver: %s, status: %s, url: %s\n", hresp.version, hresp.Status, hreq.URL)
	return hresp
}
