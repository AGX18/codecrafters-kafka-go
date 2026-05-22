package main

import (
	"encoding/binary"
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
	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}
	var messageSize int32
	binary.Read(conn, binary.BigEndian, &messageSize)
	logger.Debug("reading the request header")
	requestHeader, err := parseRequestHeaderv2(conn)
	if err != nil {
		logger.Error("error while parsing the request header", "err", err.Error())
	}
	logger.Debug("DONE reading the request header")

	buf := make([]byte, 0, 10)
	buf = binary.BigEndian.AppendUint32(buf, uint32(messageSize))
	buf = binary.BigEndian.AppendUint32(buf, uint32(requestHeader.CorrelationId))
	buf = binary.BigEndian.AppendUint16(buf, uint16(getErrorCode(requestHeader.ApiVersion))) // error code for unsupported version
	logger.Debug("Sending the response")
	conn.Write(buf)

}

type RequestHeader struct {
	ApiKey        int16
	ApiVersion    int16
	CorrelationId int32
	ClientId      *string
}

func parseRequestHeaderv2(r io.Reader) (*RequestHeader, error) {
	header := &RequestHeader{}
	binary.Read(r, binary.BigEndian, &header.ApiKey)
	binary.Read(r, binary.BigEndian, &header.ApiVersion)
	binary.Read(r, binary.BigEndian, &header.CorrelationId)

	// reading the length
	var clientIdLen int16
	binary.Read(r, binary.BigEndian, &clientIdLen)
	// if the length is -1 then it's null
	if clientIdLen == -1 {
		header.ClientId = nil
	} else {
		buf := make([]byte, clientIdLen)

		if _, err := io.ReadFull(r, buf); err != nil {
			return header, fmt.Errorf("error while reading client id: %v", err)
		}
		s := string(buf)
		header.ClientId = &s
	}
	tagBuffer := make([]byte, 1)
	io.ReadFull(r, tagBuffer)
	// TODO: implement the tag buffer feature
	return header, nil
}

func getErrorCode(apiVersion int16) int16 {
	if apiVersion >= 0 && apiVersion <= 4 {
		return 0
	}
	return 35
}
