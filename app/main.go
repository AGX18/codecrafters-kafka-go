package main

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
)

// Ensures gofmt doesn't remove the "net" and "os" imports in stage 1 (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	l, err := net.Listen("tcp", "0.0.0.0:9092")
	if err != nil {
		fmt.Println("Failed to bind to port 9092")
		os.Exit(1)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			logger.Error(fmt.Sprintf("Error accepting connection: %v", err.Error()))
			os.Exit(1)
		}

		go handleConnection(conn, logger)
	}

}

func handleConnection(conn net.Conn, logger *slog.Logger) {
	for {
		rfm := bufio.NewReader(conn)
		request := parseRequest(rfm, logger)
		requestBody, err := parseApiVersionsRequestBody(rfm)
		logger.Info(fmt.Sprintf("request body: %v", requestBody))
		if err == io.EOF {
			logger.Info("client closed the connection")
			break
		}

		if err != nil {
			logger.Error("error while parsing the request body", "err", err.Error())
			continue
		}

		logger.Debug("reading the request header")

		w := bufio.NewWriter(conn)
		logger.Debug("Sending the response")
		resp := &ApiVersionsResponse{
			ErrorCode: getErrorCode(request.Header.ApiVersion),
			ApiKeys: []ApiKey{
				{
					ApiKey: 18, MinVersion: 0, MaxVersion: 4,
				}, {
					ApiKey: 75, MinVersion: 0, MaxVersion: 0,
				},
			},
			CorrelationId:  request.Header.CorrelationId,
			ThrottleTimeMs: 0,
		}
		resp.encode(w)
		w.Flush()
	}
}
