package gorpc

import (
	"bufio"
	"encoding/gob"
	"io"
)

type Header struct {
	Sequence      string // sequence number chosen by client
	ServiceMethod string // format "Service.Method"
	Error         error
}

type Codec interface {
	ReadHeader(header *Header) error
	ReadBody(body interface{}) error
	Write(header *Header, body interface{}) error
}

type gobCodec struct {
	conn     io.ReadWriteCloser
	decoder  *gob.Decoder
	encoder  *gob.Encoder
	writeBuf *bufio.Writer
}

func newGobCodec(conn io.ReadWriteCloser) Codec {
	writeBuf := bufio.NewWriter(conn)
	return &gobCodec{
		conn:     conn,
		decoder:  gob.NewDecoder(conn),
		encoder:  gob.NewEncoder(writeBuf),
		writeBuf: writeBuf,
	}
}

func (c *gobCodec) ReadHeader(header *Header) error {
	return c.decoder.Decode(header)
}

func (c *gobCodec) ReadBody(body interface{}) error {
	return c.decoder.Decode(body)
}

func (c *gobCodec) Write(header *Header, body interface{}) error {
	defer func() {
		if c.writeBuf.Flush() != nil {
			c.conn.Close()
		}
	}()

	if err := c.encoder.Encode(header); err != nil {
		return err
	}

	if err := c.encoder.Encode(body); err != nil {
		return err
	}

	return nil
}
