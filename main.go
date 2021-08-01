package main

import (
	"bytes"
	"fmt"
	"golang.org/x/sync/errgroup"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	//go func() {
	//	time.Sleep(1 * time.Second)
	//	tmpConn, _ := net.Dial("tcp", "localhost:8080")
	//	tmpConn.Write([]byte("GET /health HTTP/1.1\n\n"))
	//}()
	// Listen for incoming connections.
	log.Println("start gateway service")
	l, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	log.Println("start listening!")
	// Close the listener when the application closes.
	defer l.Close()
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go handleRequest(conn)
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn) {
	defer conn.Close()


	header, err := readUntilHttpHeaders(conn)
	if err != nil {
		log.Println("error happen: ", err)
		return
	}

	if isHealthy(header) {
		conn.Write([]byte("HTTP/1.1 200\n\nok!"))
		return
	}

	log.Println(header)

	if !containsAuthV1(header) {
		conn.Write([]byte("HTTP/1.1 403\n\nNot authorized"))
		return
	}

	apiConn, err := net.Dial("tcp", "localhost:1323")
	if err != nil {
		conn.Write([]byte("HTTP/1.1 404\n\nNot found"))
		return
	}

	var g errgroup.Group
	var rbuf, lbuf bytes.Buffer

	g.Go(func() error {
		_, err := io.Copy(conn, io.TeeReader(apiConn, &rbuf))
		return err
	})
	g.Go(func() error {
		_,_ = io.Copy(apiConn, strings.NewReader(header))
		_, err := io.Copy(apiConn, io.TeeReader(conn, &lbuf))
		return err
	})
	if err := g.Wait(); err != nil {
		// handle error
	}
}

func readUntilHttpHeaders(conn net.Conn) (string, error){
	var buffer strings.Builder
	tmp := make([]byte, 32)     // using small tmo buffer for demonstrating
	for {
		n, err := conn.Read(tmp)
		if err != nil && err != io.EOF {
			return "", err
		}
		tmpStr := string(tmp)
		buffer.WriteString(tmpStr)
		if strings.Contains(tmpStr, "\r\n\r\n") {
			break
		}
		if n < 32 {
			break
		}
	}
	return buffer.String(), nil
}

func isHealthy(header string) bool {
	if strings.Contains(header, "/health") {
		return true
	}
	return false
}

func containsAuthV1(header string) bool {
	if strings.Contains(header, "X-Auth: authorization") {
		return true
	}
	return false
}