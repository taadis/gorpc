package gorpc

type Header struct {
	Sequence      string // sequence number chosen by client
	ServiceMethod string // format "Service.Method"
	Error         error
}
