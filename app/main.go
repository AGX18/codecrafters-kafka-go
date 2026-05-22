package main

import (
	"bufio"
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
	rfm := bufio.NewReader(conn)
	request := parseRequest(rfm, logger)
	requestBody, err := parseApiVersionsRequestBody(rfm)
	logger.Info(fmt.Sprintf("request body: %v", requestBody))
	if err != nil {
		logger.Error("error while parsing the request body", "err", err.Error())
	}

	logger.Debug("reading the request header")

	w := bufio.NewWriter(conn)
	logger.Debug("Sending the response")
	resp := &ApiVersionsResponse{
		ErrorCode:      getErrorCode(request.Header.ApiVersion),
		ApiKeys:        []ApiKey{{ApiKey: request.Header.ApiKey, MinVersion: 0, MaxVersion: 4}},
		CorrelationId:  request.Header.CorrelationId,
		ThrottleTimeMs: 0,
	}
	resp.encode(w)
	w.Flush()

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

type Request struct {
	MessageSize int32
	Header      *RequestHeader
}

func parseRequest(r io.Reader, logger *slog.Logger) *Request {
	logger.Debug("BEGIN reading the request")
	var messageSize int32
	binary.Read(r, binary.BigEndian, &messageSize)

	requestHeader, err := parseRequestHeaderv2(r)
	if err != nil {
		logger.Error("error while parsing the request header", "err", err.Error())
	}
	logger.Debug("DONE reading the request header")
	return &Request{
		MessageSize: messageSize,
		Header:      requestHeader,
	}
}

type ApiVersionsRequestBody struct {
	ClientSoftwareName    string
	ClientSoftwareVersion string
	TagBuffer             byte
}

func parseApiVersionsRequestBody(r *bufio.Reader) (*ApiVersionsRequestBody, error) {
	nameLen, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, err
	}
	name := make([]byte, nameLen-1)
	_, err = io.ReadFull(r, name)
	if err != nil {
		return nil, err
	}

	versionLen, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, err
	}

	version := make([]byte, versionLen-1)
	_, err = io.ReadFull(r, version)
	if err != nil {
		return nil, err
	}

	tagBuffer, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	return &ApiVersionsRequestBody{
		ClientSoftwareName:    string(name),
		ClientSoftwareVersion: string(version),
		TagBuffer:             tagBuffer,
	}, nil
}

type ApiKey struct {
	ApiKey     int16
	MinVersion int16
	MaxVersion int16
}

type ApiVersionsResponse struct {
	ErrorCode      int16
	ApiKeys        []ApiKey
	ThrottleTimeMs int32
	CorrelationId  int32
}

func (r *ApiVersionsResponse) encode(w *bufio.Writer) error {
	// Encode the message size
	messageSize := 4 + 2 + calcUvarintSize(uint64(len(r.ApiKeys)+1)) + (7 * len(r.ApiKeys)) + 4 + 1
	binary.Write(w, binary.BigEndian, uint32(messageSize))

	// Encode the correlation ID
	binary.Write(w, binary.BigEndian, uint32(r.CorrelationId))

	// Encode the error code
	err := binary.Write(w, binary.BigEndian, r.ErrorCode)
	if err != nil {
		return err
	}

	// Encode the number of API keys
	writeUvarint(w, uint64(len(r.ApiKeys)+1))

	// Encode each API key
	for _, apiKey := range r.ApiKeys {
		err = binary.Write(w, binary.BigEndian, apiKey.ApiKey)
		if err != nil {
			return err
		}
		err = binary.Write(w, binary.BigEndian, apiKey.MinVersion)
		if err != nil {
			return err
		}
		err = binary.Write(w, binary.BigEndian, apiKey.MaxVersion)
		if err != nil {
			return err
		}
		// TAG_BUFFER for each API key entry
		w.WriteByte(0x00)
	}

	// Encode the throttle time
	err = binary.Write(w, binary.BigEndian, r.ThrottleTimeMs)
	if err != nil {
		return err
	}

	// TAG_BUFFER at the end
	w.WriteByte(0x00)

	return nil

}

func writeUvarint(buf *bufio.Writer, x uint64) error {
	for x >= 0x80 {
		err := buf.WriteByte(byte(x) | 0x80)
		if err != nil {
			return err
		}
		x >>= 7
	}
	return buf.WriteByte(byte(x))
}

func calcUvarintSize(x uint64) int {
	size := 0

	for x >= 0x80 {
		size++
		x >>= 7
	}
	size++

	return size
}
