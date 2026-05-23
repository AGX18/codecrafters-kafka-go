package main

import (
	"bufio"
	"encoding/binary"
)

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

func (r *ApiVersionsResponse) Encode(w *bufio.Writer) error {
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

type TopicPartitionsResponse struct {
	CorrelationID  int32
	ThrottleTimeMs int32
	Topics         []TopicResponseV0
	NextCursor     int8
}

// TODO: implement the partitions
type TopicResponseV0 struct {
	ErrorCode                 int16
	TopicName                 string
	TopicID                   [16]byte
	IsInternal                bool
	TopicAuthorizedOperations int32
}

func NewTopicPartitionsResponse(correlationID int32, req *TopicPartitionsRequest) *TopicPartitionsResponse {
	topics := make([]TopicResponseV0, len(req.Topics))

	for i, topic := range req.Topics {
		topics[i] = TopicResponseV0{
			ErrorCode:                 3,
			TopicName:                 topic.TopicName,
			TopicID:                   [16]byte{},
			IsInternal:                false,
			TopicAuthorizedOperations: 0,
		}
	}

	return &TopicPartitionsResponse{
		CorrelationID:  correlationID,
		ThrottleTimeMs: 0,
		Topics:         topics,
		NextCursor:     -1,
	}
}

func (r *TopicPartitionsResponse) Encode(writer *bufio.Writer) error {
	size := r.calculateSize()
	err := binary.Write(writer, binary.BigEndian, uint32(size))
	if err != nil {
		return err
	}

	// header v1
	binary.Write(writer, binary.BigEndian, r.CorrelationID)
	writer.WriteByte(0x00) // TAG_BUFFER

	// body
	binary.Write(writer, binary.BigEndian, r.ThrottleTimeMs)

	// topics array
	writer.WriteByte(byte(len(r.Topics) + 1)) // compact array encoding

	for _, topic := range r.Topics {
		binary.Write(writer, binary.BigEndian, topic.ErrorCode)
		writer.WriteByte(byte(len(topic.TopicName) + 1)) // compact string length
		writer.WriteString(topic.TopicName)
		writer.Write(topic.TopicID[:])
		if topic.IsInternal {
			writer.WriteByte(0x01)
		} else {
			writer.WriteByte(0x00)
		}
		writer.WriteByte(0x01) // empty partitions array
		binary.Write(writer, binary.BigEndian, topic.TopicAuthorizedOperations)
		writer.WriteByte(0x00) // TAG_BUFFER
	}

	writer.WriteByte(byte(r.NextCursor)) // next_cursor (-1 = 0xff)
	writer.WriteByte(0x00)               // TAG_BUFFER

	return nil
}

func (r *TopicPartitionsResponse) calculateSize() int32 {
	size := 0

	size += 4 // correlation_id
	size += 1 // TAG_BUFFER (header)
	size += 4 // throttle_time_ms
	size += 1 // topics array length
	for _, topic := range r.Topics {
		size += 2                    // error_code
		size += 1                    // topic name length
		size += len(topic.TopicName) // topic name
		size += 16                   // topic_id
		size += 1                    // is_internal
		size += 1                    // partitions array (empty)
		size += 4                    // topic_authorized_operations
		size += 1                    // TAG_BUFFER
	}
	size += 1 // next_cursor
	size += 1 // TAG_BUFFER (body)

	return int32(size)
}
