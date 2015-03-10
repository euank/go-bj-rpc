package bjrpc

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"
)

func TestCallEncoding(t *testing.T) {
	var testrw bytes.Buffer
	testbjrpc := New(&testrw)
	go testbjrpc.Call("Test", "testarg1", 2)

	time.Sleep(1 * time.Millisecond)
	linewritten, err := testrw.ReadString(byte('\n'))
	if err != nil {
		t.Error("Unable to read written data", err)
	}

	expected := struct {
		JSONRPC string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
	}{}
	err = json.Unmarshal([]byte(linewritten), &expected)
	if err != nil {
		t.Error("written was not json")
	}

	if expected.Method != "Test" {
		t.Error("Incorrect method")
	}
	if len(expected.Params) != 2 {
		t.Fatal("Wrong number of args: ", len(expected.Params))
	}
	if expected.Params[0].(string) != "testarg1" {
		t.Error("Unexpected first arg")
	}
	// Json encoding is lossy, ints end up as float64 if you don't type coerse em bac
	if expected.Params[1].(float64) != 2 {
		t.Error("Unexpected second arg")
	}
}

type rw struct {
	io.Reader
	io.Writer
}

func TestRequestHandler(t *testing.T) {
	serverr, clientw := io.Pipe()
	clientr, serverw := io.Pipe()
	serverrw := rw{serverr, serverw}
	clientrw := rw{clientr, clientw}

	server := New(serverrw)
	client := New(clientrw)

	server.AddRequestHandler("TestMethod", func(a string) (*string, error) {
		str := "Called with " + a
		return &str, nil
	})

	go server.Go()
	go client.Go()

	var res string
	err := client.Call("TestMethod", "str").As(&res)
	if err != nil {
		t.Error(err)
	}
	if res != "Called with str" {
		t.Error("Wrong response")
	}
}
