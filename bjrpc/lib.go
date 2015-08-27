package bjrpc

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/euank/go-bj-rpc/json-rpc"
)

const DefaultRequestTimeout = 2 * time.Minute

type option func(*bjRPC)

func New(transport io.ReadWriter, opts ...option) *bjRPC {
	bjrpc := &bjRPC{
		transport:            transport,
		requestHandlers:      make(map[string]requestHandler),
		notificationHandlers: make(map[string]notificationHandler),
		requestIds:           make(map[string]func(*json_rpc.ServerResponse)),
	}
	for _, opt := range opts {
		opt(bjrpc)
	}
	return bjrpc
}

// SetTimeout sets the maximum timeout to wait for a request to be responded to
// TODO, IMPLEMENT THIS, IT IS NOT IMPLEMENTED
func SetTimeout(to time.Duration) option {
	return func(b *bjRPC) {
		b.requestTimeout = to
	}
}

// Go begins handling incoming data. It should be run for both clients and servers.
// TODO, write a context through this
func (b *bjRPC) Go() {
	scanner := bufio.NewScanner(b.transport)
	for scanner.Scan() {
		data := scanner.Bytes()

		b.handleData(data)
	}
}

// AddRequestHandler gives a function to call when a request for the given
// method name is recieved
func (b *bjRPC) AddRequestHandler(name string, f requestHandler) {
	b.requestHandlers[name] = f
}

// AddNotificationHandler gives a function to call when a notification for the
// given method name is recieved
func (b *bjRPC) AddNotificationHandler(name string, f notificationHandler) {
	b.notificationHandlers[name] = f
}

// SetUnknownHandler sets a function that will be called in the event that it
// does not match any registered RequestHandler or NotificationHandler
func (b *bjRPC) SetUnknownHandler(f unknownHandler) {
	b.fallback = f
}

// Call calls the given function over the transport with the given arguments.
// It will return a CallResult with can be checked for an error or transformed
// into a desired type
func (b *bjRPC) Call(method string, args ...interface{}) *CallResult {
	done := make(chan *CallResult)
	id := atomic.AddInt64(&b._id, 1)

	req := json_rpc.NewRequest(method, &id, args...)

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return &CallResult{req.Response(nil, json_rpc.JsonParseError(err))}
	}

	b.requestIdLock.Lock()
	b.requestIds[strconv.FormatInt(id, 10)] = func(resp *json_rpc.ServerResponse) {
		done <- &CallResult{resp}
	}
	b.requestIdLock.Unlock()

	b.transportLock.Lock()
	defer b.transportLock.Unlock()
	reqBytes = append(reqBytes, []byte("\r\n")...)
	_, err = b.transport.Write(reqBytes)
	if err != nil {
		// Make the assumption that we won't hear back; this could be a mistake if
		// e.g. this is a partial write but a newline still ends up on the wire
		// somehow. Oh well.
		b.requestIdLock.Lock()
		defer b.requestIdLock.Unlock()
		delete(b.requestIds, strconv.FormatInt(id, 10))

		return &CallResult{req.Response(nil, json_rpc.NewError(err))}
	}

	return <-done
}

func (b *bjRPC) Notify(method string, args ...interface{}) error {
	req := json_rpc.NewRequest(method, nil, args...)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return json_rpc.JsonParseError(err)
	}

	b.transportLock.Lock()
	defer b.transportLock.Unlock()
	reqBytes = append(reqBytes, []byte("\r\n")...)
	_, err = b.transport.Write(reqBytes)
	return err
}

func (b *bjRPC) handleData(data []byte) {
	msg := &json_rpc.ServerMessage{}
	err := json.Unmarshal(data, msg)
	if err != nil && msg.ServerRequest != nil {
		if msg.ServerRequest.Id != nil {
			b.writeResponse(msg.ServerRequest, nil, json_rpc.JsonParseError(err))
		} else {
			// TODO, configed error handler
		}
		return
	}

	if msg.IsRequest() {
		b.handleRequest(msg.ServerRequest)
	} else if msg.IsResponse() {
		b.handleResponse(msg.ServerResponse)
	} // else uh oh
}

func (b *bjRPC) handleRequest(req *json_rpc.ServerRequest) {
	var handler reflect.Value
	if req.IsNotification() {
		handlerfn, ok := b.notificationHandlers[req.Method]
		if !ok {
			if b.fallback != nil {
				b.fallback(req)
				return
			}
		}
		handler = reflect.ValueOf(handlerfn)
	} else {
		handlerfn, ok := b.requestHandlers[req.Method]
		if !ok {
			if b.fallback != nil {
				b.fallback(req)
				return
			}
			b.writeResponse(req, nil, json_rpc.MethodNotFoundError(errors.New("No such method: "+req.Method)))
			return
		}
		handler = reflect.ValueOf(handlerfn)
	}
	fnType := handler.Type()
	numArgs := fnType.NumIn()
	if numArgs != len(req.Params) {
		b.writeResponse(req, nil, json_rpc.InvalidParamsError(errors.New("Expected "+strconv.Itoa(numArgs)+" arguments")))
		return
	}
	callArgs := make([]reflect.Value, numArgs)
	for i := 0; i < numArgs; i++ {
		argType := fnType.In(i)
		arg := reflect.Indirect(reflect.New(argType))
		var iface interface{}
		if arg.CanAddr() {
			iface = arg.Addr().Interface()
		} else {
			iface = arg.Interface()
		}
		err := json.Unmarshal(req.Params[i], iface)
		if err != nil {
			b.writeResponse(req, nil, json_rpc.InvalidParamsError(err))
			return
		}

		callArgs[i] = arg
	}
	resp := handler.Call(callArgs)

	if !req.IsNotification() {
		// resp is interface{}, error; need to send it back
		// Permissively allow a response type of just 'interface{}' however for functiosn that never return errors.
		if len(resp) != 2 && len(resp) != 1 {
			b.writeResponse(req, nil, json_rpc.NewUnretriableError(errors.New("Incorrect return type for server function; must of have one or two returns")))
			return
		}
		result := resp[0]
		var rerr error

		if len(resp) == 2 {
			err := resp[1]
			erriface := err.Interface()
			var ok bool
			rerr, ok = erriface.(error)
			if !err.IsNil() && !ok {
				b.writeResponse(req, nil, json_rpc.NewUnretriableError(errors.New("Incorrect return type for server function; second argument *must* be an error if present")))
				return
			}
		}
		if rerr != nil {
			b.writeResponse(req, nil, rerr)
		} else {
			b.writeResponse(req, result.Interface(), nil)
		}
	}
}

func (b *bjRPC) handleResponse(resp *json_rpc.ServerResponse) {
	b.requestIdLock.RLock()
	callback, ok := b.requestIds[resp.Id]
	b.requestIdLock.RUnlock()
	if !ok {
		// TODO, configurable error handler
		return
	}
	callback(resp)
}

func (b *bjRPC) writeResponse(req *json_rpc.ServerRequest, res interface{}, err error) {
	resp := req.Response(res, err)
	data, err := json.Marshal(resp)
	if err != nil {
		// TODO, configurable error handler
		return
	}
	data = append(data, []byte("\r\n")...)

	b.transportLock.Lock()
	defer b.transportLock.Unlock()
	_, err = b.transport.Write(data)
	if err != nil {
		// Todo, configurable error handler
	}
}
