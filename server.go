package gorpc

import (
	"reflect"
	"sync"
)

// method represents registered every method of service.
type method struct {
	method    reflect.Method
	argsType  reflect.Type
	replyType reflect.Type
}

// service represents registered every service.
type service struct {
	name    string             // name of service
	valueof reflect.Value      // receiver of methods for the service
	typeof  reflect.Type       // type of the receiver
	methods map[string]*method // registered methods
}

// getServiceMethods return methods of service type.
func getServiceMethods(typeof reflect.Type) map[string]*method {
	methods := make(map[string]*method)

	for i, numMethod := 0, typeof.NumMethod(); i < numMethod; i++ {
		me := typeof.Method(i)
		meType := me.Type
		meName := me.Name

		// method must be exportable
		if me.PkgPath != "" {
			continue
		}

		// method needs 3 ins: , receiver, *args, *reply.
		if meType.NumIn() != 3 {
			continue
		}

		// first in must be a pointer.
		argsType := meType.In(1)
		if argsType.Kind() != reflect.Ptr {
			continue
		}

		// second in must be a pointer.
		replyType := meType.In(2)
		if replyType.Kind() != reflect.Ptr {
			continue
		}

		// method needs 1 out.
		if meType.NumOut() != 1 {
			continue
		}

		// the out type must be error.
		if outType := meType.Out(0); outType != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}

		methods[meName] = &method{
			method:    me,
			argsType:  argsType,
			replyType: replyType,
		}
	}
	return methods
}

// Server represents an RPC Server.
type Server struct {
	services sync.Map // map[string]*service
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Register(receiver interface{}) error {
	srv := new(service)
	srv.typeof = reflect.TypeOf(receiver)
	srv.valueof = reflect.ValueOf(receiver)
	srv.name = reflect.Indirect(srv.valueof).Type().Name()
	srv.methods = getServiceMethods(srv.typeof)

	s.services.Store(srv.name, srv)
	return nil
}
