package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
)

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

type TopicPartitionsRequest struct {
	Topics                 []TopicRequestV0
	ResponsePartitionLimit int32
	Cursor                 []byte // nullable, -1 = null
}

type TopicRequestV0 struct {
	TopicName string
}

func ParseTopicPartitionsRequest(reader *bufio.Reader) (*TopicPartitionsRequest, error) {
	req := &TopicPartitionsRequest{}

	arrayLen, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	topicCount := int(arrayLen) - 1

	for i := 0; i < topicCount; i++ {
		nameLen, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}

		nameBytes := make([]byte, int(nameLen)-1)
		_, err = io.ReadFull(reader, nameBytes)
		if err != nil {
			return nil, err
		}

		topic := TopicRequestV0{TopicName: string(nameBytes)}
		req.Topics = append(req.Topics, topic)

		// Read the tag buffer for each topic
		_, err = reader.ReadByte()
		if err != nil {
			return nil, err
		}
	}

	err = binary.Read(reader, binary.BigEndian, &req.ResponsePartitionLimit)
	if err != nil {
		return nil, err
	}

	cursor, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}

	if cursor != 0xff {
		return nil, fmt.Errorf("non-null cursor not supported")
	}

	req.Cursor = nil

	// discard TAG_BUFFER
	_, err = reader.ReadByte()
	if err != nil {
		return nil, err
	}

	return req, nil
}
