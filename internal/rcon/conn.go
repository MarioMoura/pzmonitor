package rcon

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// Source RCON packet types.
const (
	typeResponseValue = 0 // SERVERDATA_RESPONSE_VALUE
	typeExecCommand   = 2 // SERVERDATA_EXECCOMMAND
	typeAuthResponse  = 2 // SERVERDATA_AUTH_RESPONSE (same value, server->client)
	typeAuth          = 3 // SERVERDATA_AUTH
)

// maxPacketSize bounds a single packet body to guard against a corrupt
// size field; PZ fragments large responses at ~4KB.
const maxPacketSize = 1 << 20

// conn is a Source RCON connection that copes with Project Zomboid's
// server behavior: command responses are produced asynchronously (they
// arrive on a later game tick) and large responses are fragmented across
// multiple packets. Responses are matched by request ID, and each command
// is followed by an empty SERVERDATA_RESPONSE_VALUE marker packet that the
// server echoes back in order, which bounds the response deterministically.
type conn struct {
	tcp     net.Conn
	nextID  int32
	timeout time.Duration
}

// dialConn connects, authenticates, and returns a ready-to-use connection.
func dialConn(addr, password string, timeout time.Duration) (*conn, error) {
	tcp, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	c := &conn{tcp: tcp, nextID: 1, timeout: timeout}
	if err := c.auth(password); err != nil {
		tcp.Close()
		return nil, err
	}
	return c, nil
}

func (c *conn) Close() error {
	return c.tcp.Close()
}

// auth performs the RCON handshake. The server may send an empty
// RESPONSE_VALUE before the AUTH_RESPONSE; an AUTH_RESPONSE with ID -1
// means the password was rejected.
func (c *conn) auth(password string) error {
	id := c.id()
	if err := c.writePacket(id, typeAuth, password); err != nil {
		return fmt.Errorf("auth write: %w", err)
	}
	for {
		pid, ptype, _, err := c.readPacket()
		if err != nil {
			return fmt.Errorf("auth read: %w", err)
		}
		if ptype != typeAuthResponse {
			continue
		}
		if pid == -1 {
			return errors.New("authentication refused")
		}
		if pid == id {
			return nil
		}
	}
}

// execute sends a command and returns its complete response body, however
// many packets it spans and however late the server produces it. Packets
// with unknown IDs (stale responses from earlier commands) are discarded.
func (c *conn) execute(cmd string) (string, error) {
	cmdID := c.id()
	markerID := c.id()
	if err := c.writePacket(cmdID, typeExecCommand, cmd); err != nil {
		return "", fmt.Errorf("write command: %w", err)
	}
	if err := c.writePacket(markerID, typeResponseValue, ""); err != nil {
		return "", fmt.Errorf("write marker: %w", err)
	}
	var body strings.Builder
	for {
		pid, _, payload, err := c.readPacket()
		if err != nil {
			return "", fmt.Errorf("read response: %w", err)
		}
		switch pid {
		case cmdID:
			body.Write(payload)
		case markerID:
			return body.String(), nil
		}
	}
}

func (c *conn) id() int32 {
	id := c.nextID
	c.nextID++
	return id
}

func (c *conn) writePacket(id, ptype int32, body string) error {
	if err := c.tcp.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	size := int32(4 + 4 + len(body) + 2)
	buf := make([]byte, 0, 4+size)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(size))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(id))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(ptype))
	buf = append(buf, body...)
	buf = append(buf, 0, 0)
	_, err := c.tcp.Write(buf)
	return err
}

func (c *conn) readPacket() (id, ptype int32, body []byte, err error) {
	if err := c.tcp.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return 0, 0, nil, err
	}
	var sizeBuf [4]byte
	if _, err := io.ReadFull(c.tcp, sizeBuf[:]); err != nil {
		return 0, 0, nil, err
	}
	size := int32(binary.LittleEndian.Uint32(sizeBuf[:]))
	if size < 10 || size > maxPacketSize {
		return 0, 0, nil, fmt.Errorf("invalid packet size %d", size)
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(c.tcp, data); err != nil {
		return 0, 0, nil, err
	}
	id = int32(binary.LittleEndian.Uint32(data[0:4]))
	ptype = int32(binary.LittleEndian.Uint32(data[4:8]))
	return id, ptype, data[8 : size-2], nil
}
