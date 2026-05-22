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
