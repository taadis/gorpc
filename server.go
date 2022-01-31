package gorpc

import (
	"errors"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
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

// call the service method and codec to reply.
func (s *service) call(me *method, argv reflect.Value, replyv reflect.Value) error {
	function := me.method.Func
	outValues := function.Call([]reflect.Value{s.valueof, argv, replyv})
	// out value just is an error.
	return outValues[0].Interface().(error)
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

// Accept accept connections on the listener
// and serves for each incoming connection.
func (s *Server) Accept(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print("gorpc server: listener accept error: " + err.Error())
			continue
		}

		go s.ServeConn(conn)
	}
}

// ServeConn serve a single connection.
func (s *Server) ServeConn(conn io.ReadWriteCloser) {
	codec := newGobCodec(conn)
	s.ServeCodec(codec)
}

// ServeCodec is like ServeConn but use the specified codec to
// decode requests and encode responses.
func (s *Server) ServeCodec(codec Codec) {
	s.serveRequest(codec)
}

// serveRequest will read request and call service method and write response.
func (s *Server) serveRequest(codec Codec) {
	req, err := s.readRequest(codec)
	if err != nil {
		s.writeResponse(codec, req.header, invalidRequest)
		return
	}

	err = req.service.call(req.method, req.argv, req.replyv)
	if err != nil {
		req.header.Error = err
		s.writeResponse(codec, req.header, invalidRequest)
		return
	}

	s.writeResponse(codec, req.header, req.replyv.Interface())
}

func (s *Server) checkRequestHeader(header *Header) (*service, *method, error) {
	dot := strings.LastIndex(header.ServiceMethod, ".")
	if dot < 0 {
		return nil, nil, errors.New("gorpc server: service method format error: " + header.ServiceMethod)
	}
	serviceName := header.ServiceMethod[:dot]
	methodName := header.ServiceMethod[dot+1:]

	// look up the service and method
	srvi, ok := s.services.Load(serviceName)
	if !ok {
		return nil, nil, errors.New("gorpc server: not found service: " + header.ServiceMethod)
	}

	srv := srvi.(*service)
	meType := srv.methods[methodName]
	if meType == nil {
		return nil, nil, errors.New("gorpc server: not found method: " + header.ServiceMethod)
	}

	return srv, meType, nil
}

func (s *Server) readRequestHeader(codec Codec) (req *request, err error) {
	header := &Header{}
	err = codec.ReadHeader(header)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		}
		err = errors.New("gorpc server: read request header error: " + err.Error())
		return
	}

	req = &request{}
	req.header = header
	req.service, req.method, err = s.checkRequestHeader(header)
	if err != nil {
		return
	}

	return
}

type request struct {
	header  *Header       // header of request
	service *service      // call service of request
	method  *method       // call method of request
	argv    reflect.Value // arg value of request
	replyv  reflect.Value // reply value of request
}

func (s *Server) readRequest(codec Codec) (*request, error) {
	req, err := s.readRequestHeader(codec)
	if err != nil {
		return nil, err
	}

	req.argv = reflect.New(req.method.argsType.Elem())

	err = codec.ReadBody(req.argv.Interface())
	if err != nil {
		log.Printf("gorpc server: read body error: " + err.Error())
		return req, err
	}

	return req, nil
}

// invalidRequest as a placeholder response.
var invalidRequest = struct{}{}

func (s *Server) writeResponse(codec Codec, header *Header, body interface{}) {
	err := codec.Write(header, body)
	if err != nil {
		log.Printf("gorpc server: write response error: " + err.Error())
		return
	}
}
