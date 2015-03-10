package json_rpc

import "encoding/json"

// RPCError is the low-level json rpc error object
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// NewError creates a simple server error from a given error
func NewError(err error) *RPCError {
	if rpcerr, ok := err.(*RPCError); ok {
		return rpcerr
	}
	return &RPCError{
		Code:    -32000,
		Message: err.Error(),
	}
}

// Error returns the message associated with an RPC Error
func (err *RPCError) Error() string {
	return err.Message
}

// Retry returns true if the error is retriable
func (err *RPCError) Retry() bool {
	if err.Code == -32603 {
		return true
	}
	if err.Code <= -32600 {
		return false
	}
	if err.Code == -32001 {
		return false
	}
	return true
}

func NewUnretriableError(err error) *RPCError {
	return &RPCError{
		Code:    -32001,
		Message: err.Error(),
	}
}

func JsonParseError(err error) *RPCError {
	return &RPCError{
		Code:    -32700,
		Message: err.Error(),
	}
}

func MethodNotFoundError(err error) *RPCError {
	return &RPCError{
		Code:    -32601,
		Message: err.Error(),
	}
}

func InvalidParamsError(err error) *RPCError {
	return &RPCError{
		Code:    -32602,
		Message: err.Error(),
	}
}

func InvalidRequestObject(err error) *RPCError {
	return &RPCError{
		Code:    -32600,
		Message: err.Error(),
	}
}
