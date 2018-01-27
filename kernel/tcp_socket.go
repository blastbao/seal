package kernel

import (
	"fmt"
	"io"
	"net"
	"time"
)

type TcpSock struct {
	net.Conn
	TimeOut uint32
}

func (conn *TcpSock) ExpectBytesFull(buf []uint8, size uint32) (err error) {

	err = conn.SetDeadline(time.Now().Add(time.Duration(conn.TimeOut) * time.Second))
	if err != nil {
		return
	}

	if _, err = io.ReadFull(conn.Conn, buf[:size]); err != nil {
		return
	}

	return
}

func (conn *TcpSock) SendBytes(buf []uint8) (err error) {

	err = conn.SetDeadline(time.Now().Add(time.Duration(conn.TimeOut) * time.Second))
	if err != nil {
		return
	}

	var n int
	if n, err = conn.Conn.Write(buf); err != nil {
		return
	}

	if n != len(buf) {
		err = fmt.Errorf("tcp sock, send bytes error, need send ", len(buf), ",actually send ", n)
		return
	}

	return
}
