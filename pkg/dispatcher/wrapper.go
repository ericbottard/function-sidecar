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

package dispatcher

import "log"

type wrapper struct {
	old    OldDispatcher
	input  chan<- Message
	output <-chan Message
}

func (w wrapper) Input() chan<- Message {
	return w.input
}

func (w wrapper) Output() <-chan Message {
	return w.output
}

// PropagatedHeaders is the set of header names that will be copied from the incoming message
// to the outgoing message
var PropagatedHeaders = []string{"correlationId"}

// copy headers from incomingMessage that need to be propagated into resultMessage.Headers
func propagateHeaders(incomingMessage *Message, resultMessage *Message) {
	for _, h := range PropagatedHeaders {
		if value, ok := incomingMessage.Headers[h]; ok {
			if resultMessage.Headers == nil {
				resultMessage.Headers = make(map[string]interface{})
			}
			resultMessage.Headers[h] = value
		}
	}
}

func NewWrapper(old OldDispatcher) (wrapper, error) {
	i := make(chan Message)
	o := make(chan Message)

	go func() {
		for {
			select {
			case in, open := <-i:
				if open {
					log.Printf("Wrapper received %v\n", in)
					payload, headers, err := old.Dispatch(in.Payload, in.Headers)
					if err != nil {
						log.Printf("Error calling old dispatcher %v\n", err)
					}
					message := Message{Payload: payload, Headers: headers}
					propagateHeaders(&in, &message)
					log.Printf("Wrapper about to forward %v\n", message)
					o <- message
				} else {
					close(o)
					log.Print("Shutting down wrapper")
					return
				}
			}
		}
	}()

	return wrapper{old: old, input: i, output: o}, nil
}
