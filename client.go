package gorpc

// Call represents an active RPC call data.
type Call struct {
	ServiceMethod string      // name of the remote function, format "Service.Method".
	Args          interface{} // the arguments of the remote function.
	Reply         interface{} // the arguments of the remote function.
	Error         error
	Done          chan *Call // notice when call are complete.
}

func (c *Call) done() {
	select {
	case c.Done <- c:
		// done
	}
}
