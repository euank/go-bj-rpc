package json_rpc

import (
	"encoding/json"
	"errors"
	"strconv"
)

// Message is a type that represents either a Request or Response. It would be a union type if Go had unions.
// It is json encodable and decodable
type ClientMessage struct {
	*ClientRequest
	*ClientResponse
}

type ServerMessage struct {
	*ServerRequest
	*ServerResponse
}

func (msg *ClientMessage) IsRequest() bool {
	return msg.ClientRequest != nil
}

func (msg *ClientMessage) IsResponse() bool {
	return msg.ClientResponse != nil
}

func (msg *ServerMessage) IsRequest() bool {
	return msg.ServerRequest != nil
}

func (msg *ServerMessage) IsResponse() bool {
	return msg.ServerResponse != nil
}

type intermediateMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	Method  *string          `json:"method"`
	Result  *json.RawMessage `json:"result"`
	Error   *RPCError        `json:"error"`
}

func (msg *ServerMessage) UnmarshalJSON(data []byte) error {
	intermediate := intermediateMessage{}
	err := json.Unmarshal(data, &intermediate)
	if err != nil {
		return err
	}
	// if intermediate.JSONRPC != "2.0" { return errors.New("Wrong version") } // meh, who cares?
	if intermediate.Method != nil {
		req := ServerRequest{}
		err = json.Unmarshal(data, &req)
		if err != nil {
			return err
		}
		msg.ServerRequest = &req
		return nil
	} else if intermediate.Result != nil || intermediate.Error != nil {
		resp := ServerResponse{}
		err = json.Unmarshal(data, &resp)
		if err != nil {
			return err
		}
		msg.ServerResponse = &resp
		return nil
	}

	return InvalidRequestObject(errors.New("All valid JSONRPC requests have a 'method', 'result', or 'error' field."))
}

// Request is the low-level json rpc request object
// http://www.jsonrpc.org/specification#request_object
//
// Differences from the spec:
// * Params *MUST* be positional, not by name
// * Id *MUST* be a string, null, or omitted (with null having the same meaning
//   as omitted in the spec) and all ids created by this will be stringified ints
type ClientRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      *string       `json:"id",omitempty`
}

type ServerRequest struct {
	JSONRPC string            `json:"jsonrpc"`
	Method  string            `json:"method"`
	Params  []json.RawMessage `json:"params"`
	Id      *string           `json:"id",omitempty`
}

func NewRequest(method string, id *int64, params ...interface{}) *ClientRequest {
	req := &ClientRequest{
		JSONRPC: "2.0",
		Method:  method,
	}
	if id != nil {
		reqId := strconv.FormatInt(*id, 10)
		req.Id = &reqId
	} // else notification
	req.Params = params
	return req
}

func (req *ServerRequest) IsNotification() bool {
	return req.Id == nil
}

func (req *ServerRequest) Response(result interface{}, err error) *ClientResponse {
	resp := &ClientResponse{JSONRPC: "2.0"}
	resp.Id = *req.Id
	if result != nil {
		resp.Result = result
	} else if err != nil {
		resp.Error = NewError(err)
	} else {
		// All valid responses have a result or an error
		resp.Error = InvalidRequestObject(errors.New("Server responded without result or error, oh my"))
	}
	return resp
}

func (req *ClientRequest) Response(result json.RawMessage, err error) *ServerResponse {
	resp := &ServerResponse{JSONRPC: "2.0"}
	resp.Id = *req.Id
	if result != nil {
		resp.Result = &result
	} else {
		if err != nil {
			resp.Error = NewError(err)
		}
	}
	return resp
}

// Response is the low-level json rpc response object
// http://www.jsonrpc.org/specification#response_object
type ClientResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
	Error   *RPCError   `json:"error",omitempty`
	Id      string      `json:"id"`
}

type ServerResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	Result  *json.RawMessage `json:"result",omitempty`
	Error   *RPCError        `json:"error",omitempty`
	Id      string           `json:"id"`
}
