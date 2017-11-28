/*
 * Copyright 2017 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */


// Package wireformat deals with how to serialize/deserialize a dispatcher.Message on a Kafka topic.
// Currently uses a custom encoding scheme for headers, until Kafka 0.11 headers are supported by go client lib
package wireformat

import (
	"encoding/json"
	"encoding/binary"
	"errors"
	"github.com/Shopify/sarama"
	"github.com/projectriff/function-sidecar/pkg/dispatcher"
)

func FromKafka(kafka *sarama.ConsumerMessage) (dispatcher.Message, error) {
	return extractMessage(kafka.Value)
}

func ToKafka(message dispatcher.Message) (*sarama.ProducerMessage, error) {
	bytesOut, err := encodeMessage(message)
	if err != nil {
		return nil, err
	}
	return &sarama.ProducerMessage{Value: sarama.ByteEncoder(bytesOut)}, nil
}

func extractMessage(bytes []byte) (dispatcher.Message, error) {
	offset := uint32(0)
	if bytes[offset] != 0xff {
		return dispatcher.Message{}, errors.New("expected 0xff as the leading byte")
	}
	offset++

	headerCount := bytes[offset]
	offset++

	headers := make(map[string]interface{}, headerCount)
	if headerCount == 0 {
		headers = nil
	}
	for i := byte(0); i < headerCount; i = i + 1 {
		len := uint32(bytes[offset])
		offset++

		name := string(bytes[offset:offset+len])
		offset += len

		len = binary.BigEndian.Uint32(bytes[offset:offset+4])
		offset += 4
		var value interface{}
		err := json.Unmarshal(bytes[offset:offset+len], &value)
		if err != nil {
			return dispatcher.Message{}, err
		}
		headers[name] = value
		offset += len
	}
	var payload interface{}
	if len(bytes[offset:]) == 0 {
		payload = nil
	} else {
		payload = bytes[offset:]
	}
	return dispatcher.Message{payload, headers}, nil
}

func encodeMessage(message dispatcher.Message) ([]byte, error) {
	length := 0
	length++ // initial 0xff
	length++ // no of headers

	headerValues := make(map[string][]byte, len(message.Headers))
	for k,v := range message.Headers {
		length += 1 // 1 byte to encode len(k)
		length += len(k)
		var err error
		headerValues[k], err = json.Marshal(v)
		if err != nil {
			return nil, err
		}
		length += 4 // 4bytes to encode len(hv[i])
		length += len(headerValues[k])
	}

	if message.Payload != nil {
		length += len(message.Payload.([]byte))
	}

	result := make([]byte, length)
	offset := 0

	result[offset] = 0xff
	offset++

	result[offset] = byte(len(message.Headers))
	offset++

	for k,_ := range message.Headers {
		l := len(k)
		result[offset] = byte(l)
		offset++

		copy(result[offset:offset+l], []byte(k))
		offset += l

		binary.BigEndian.PutUint32(result[offset:offset+4], uint32(len(headerValues[k])))
		offset += 4
		copy(result[offset:], headerValues[k])
		offset += len(headerValues[k])
	}
	if message.Payload != nil {
		copy(result[offset:], message.Payload.([]byte))
	}
	return result, nil
}