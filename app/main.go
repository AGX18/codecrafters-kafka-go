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
	defer conn.Close()
	for {
		rfm := bufio.NewReader(conn)
		logger.Debug("reading the request header")
		request := parseRequest(rfm, logger)

		w := bufio.NewWriter(conn)

		switch request.Header.ApiKey {
		case 18: // ApiVersions
			requestBody, err := parseApiVersionsRequestBody(rfm)
			logger.Info(fmt.Sprintf("request body: %v", requestBody))
			if err == io.EOF {
				logger.Info("client closed the connection")
				return
			}
			if err != nil {
				logger.Error("error while parsing the request body", "err", err.Error())
				continue
			}
			// send ApiVersions response
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
			logger.Debug("Sending the response")
			resp.Encode(w)
		case 75: // DescribeTopicPartitions
			logger.Debug("Handling DescribeTopicPartitions request")
			// parse request, build and send response
			req, err := ParseTopicPartitionsRequest(rfm)
			if err == io.EOF {
				logger.Info("client closed the connection")
				return
			}
			if err != nil {
				logger.Error("error while parsing the request body", "err", err.Error())
				continue
			}

			res := NewTopicPartitionsResponse(request.Header.CorrelationId, req)
			res.Encode(w)

		default:
			// unknown API key
			return
		}

		w.Flush()
	}
}
