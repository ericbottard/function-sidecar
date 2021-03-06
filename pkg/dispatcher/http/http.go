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

package http

import (
	dispatcher "github.com/sk8sio/function-sidecar/pkg/dispatcher"
	"net/http"
	"bytes"
	"io/ioutil"
	"log"
	"time"
	"net"
	retry "github.com/giantswarm/retry-go"


)

const UseTimeout = 10000000 // "Infinite" number of retries to override default and use the Timeout approach instead
const ConnectionAttemptTimeout = 1 * time.Minute
const ConnectionAttemptInterval = 100 * time.Millisecond

type httpDispatcher struct {
}

func (httpDispatcher) Dispatch(in interface{}) (interface{}, error) {
	slice := ([]byte)(in.(string))

	client := http.Client{
		Timeout: time.Duration(60 * time.Second),
	}
	resp, err := client.Post("http://localhost:8080", "text/plain", bytes.NewReader(slice))

	if err != nil {
		log.Printf("Error invoking http://localhost:8080: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response", err)
		return nil, err
	}

	return string(out), nil
}

func NewHttpDispatcher() dispatcher.Dispatcher {
	attemptDial := func() error {
		log.Println("Waiting for function to accept connection on localhost:8080")
		_, err := net.Dial("tcp", "localhost:8080")
		return err
	}

	err := retry.Do(attemptDial,
		retry.Timeout(ConnectionAttemptTimeout),
		retry.Sleep(ConnectionAttemptInterval),
		retry.MaxTries(UseTimeout))
	if err != nil {
		panic(err)
	}
	return httpDispatcher{}
}
