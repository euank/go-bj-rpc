package bjrpc

import (
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/euank/bj-json-rpc/json-rpc"
)

//type requestHandler func(...interface{}) (interface{}, error)
type requestHandler interface{}

//type notificationHandler func(...interface{})
type notificationHandler interface{}

type unknownHandler func(*json_rpc.ServerRequest) *json_rpc.ClientResponse

type bjRPC struct {
	transportLock sync.Mutex
	transport     io.ReadWriter

	requestIdLock  sync.RWMutex
	requestIds     map[string]func(*json_rpc.ServerResponse)
	requestTimeout time.Duration

	requestHandlers      map[string]requestHandler
	notificationHandlers map[string]notificationHandler
	fallback             unknownHandler

	_id int64
}

type CallResult struct {
	response *json_rpc.ServerResponse
}

func (c *CallResult) As(arg interface{}) error {
	if c.response.Error != nil {
		return c.response.Error
	}

	return json.Unmarshal(*c.response.Result, arg)
}

// note, it would make sense to rename this function if it's desirable that
// this implement the error interface
func (c *CallResult) Error() error {
	return c.response.Error
}
