package memcodec

import (
	"errors"
	"net/rpc"
	"reflect"
)

// Codec is an in memory net/rpc codec.
type Codec struct {
	// Error is the error returned by the rpc response.
	Error error

	method string
	args   interface{}
	reply  interface{}
}

// New returns an in memory codec.
func New(method string, args, reply interface{}) *Codec {
	return &Codec{
		method: method,
		args:   args,
		reply:  reply,
	}
}

// ReadRequestHeader reads the request header.
func (c *Codec) ReadRequestHeader(req *rpc.Request) error {
	req.ServiceMethod = c.method
	return nil
}

// ReadRequestBody reads the response body.
func (c *Codec) ReadRequestBody(args interface{}) error {
	if args == nil {
		return errors.New("args cannot be nil")
	}

	src := reflect.Indirect(reflect.ValueOf(c.args))
	dst := reflect.Indirect(reflect.ValueOf(args))
	dst.Set(src)
	return nil
}

// WriteResponse writes the response.
func (c *Codec) WriteResponse(resp *rpc.Response, reply interface{}) error {
	if resp.Error != "" {
		c.Error = errors.New(resp.Error)
		return nil
	}

	src := reflect.Indirect(reflect.ValueOf(reply))
	dst := reflect.Indirect(reflect.ValueOf(c.reply))
	dst.Set(src)
	return nil
}

// Close closes the codec.
func (c *Codec) Close() error {
	return nil
}
