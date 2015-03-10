package json_rpc

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestUnmarshalRequest(t *testing.T) {
	msg := ServerMessage{}
	json.Unmarshal([]byte(`{"jsonrpc":"2.0","method":"test","params":[1,2,{"a":"b"}],"id":"1"}`), &msg)
	if !msg.IsRequest() {
		t.Fatal("Should have been request")
	}
	if msg.IsResponse() {
		t.Fatal("Should have been request")
	}
	if msg.Method != "test" {
		t.Error("Incorrect method")
	}

	if !reflect.DeepEqual(msg.Params, []json.RawMessage{json.RawMessage([]byte(`1`)), json.RawMessage([]byte(`2`)), json.RawMessage([]byte(`{"a":"b"}`))}) {
		t.Error("Incorrect params")
	}
	if *msg.ServerRequest.Id != "1" {
		t.Error("incorrect id")
	}
	if msg.IsNotification() {
		t.Error("Should not be notification")
	}
}

func TestClientMessage(t *testing.T) {
	msg := &ClientMessage{}
	err := json.Unmarshal([]byte(`{"jsonrpc":"2.0","method":"test","params":[],"id":"1"}`), &msg)
	if err != nil {
		t.Fatal(err)
	}
	if !msg.IsRequest() {
		t.Fatal("Message should be request")
	}
	if msg.IsResponse() {
		t.Fatal("Message should be request")
	}
	if msg.Method != "test" {
		t.Fatal("wrong method")
	}

	msg = &ClientMessage{}
	err = json.Unmarshal([]byte(`{"jsonrpc":"2.0","result":"abc","id":"1"}`), &msg)
	if err != nil {
		t.Fatal(err)
	}
	if msg.IsRequest() {
		t.Fatal("Message should be response")
	}
	if !msg.IsResponse() {
		t.Fatal("Message should be response")
	}
	if msg.Result.(string) != "abc" {
		t.Fatal("Wrong result")
	}
}

func TestMalformedServerMessage(t *testing.T) {
	msg := &ServerMessage{}
	err := json.Unmarshal([]byte(`{"jsonrpc":1}`), msg)
	if err == nil {
		t.Error("jsonrpc must be a string")
	}

	err = json.Unmarshal([]byte(`{"jsonrpc":"2.0","method":"m","params":null,"id":{}}`), msg)
	if err == nil {
		t.Error("Id can't be an object")
	}
}

func TestUnmarshalResponse(t *testing.T) {
	msg := &ServerMessage{}
	err := json.Unmarshal([]byte(`{"jsonrpc":"2.0","result":123}`), msg)
	if err != nil {
		t.Fatal(err)
	}
	if !msg.IsResponse() {
		t.Error("Is response")
	}
	if !reflect.DeepEqual(*msg.Result, json.RawMessage([]byte(`123`))) {
		t.Error("Wrong result")
	}

	err = json.Unmarshal([]byte(`{"jsonrpc":"2.0","error":{"code":1,"message":"asdf"}}`), msg)
	if err != nil {
		t.Fatal(err)
	}
	if !msg.IsResponse() {
		t.Error("Is response")
	}
	if msg.Error.Error() != "asdf" {
		t.Error("Did not print message")
	}

	err = json.Unmarshal([]byte(`{"jsonrpc":"2.0","method":"TestMethod","params":["str"],"id":"1"}`), &msg)
	if err != nil {
		t.Error("Should unmarshal")
	}
}

func TestMalformedError(t *testing.T) {
	msg := &ServerMessage{}
	err := json.Unmarshal([]byte(`{"jsonrpc":"2.0","result":{"asdf":"zxcv"}, "id":123}`), msg)
	if err == nil {
		t.Fatal("Id must be a string")
	}
}

func TestNewRequest(t *testing.T) {
	id := int64(10)
	req := NewRequest("Method", &id, "a", 'a', 1)
	if req.Method != "Method" {
		t.Error("Wrong method")
	}
	if len(req.Params) != 3 {
		t.Error("Wrong params len")
	}
	if *req.Id != "10" {
		t.Error("Wrong id")
	}
}

func TestConstructResponse(t *testing.T) {
	id := "10"
	req := &ServerRequest{Id: &id, JSONRPC: "2.0", Method: "asdf"}

	resp := req.Response(nil, errors.New("This broke"))
	if resp.Id != id {
		t.Error("Wrong id")
	}

	req = &ServerRequest{Id: &id, JSONRPC: "2.0", Method: "asdf"}
	resp = req.Response("The result", nil)

	if resp.Result.(string) != "The result" {
		t.Error("Wrong result")
	}
}
