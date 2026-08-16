package rcon

import (
	"encoding/binary"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

// fakePZServer emulates Project Zomboid's RCON behavior: an empty
// RESPONSE_VALUE before the auth response, command responses delivered
// late and fragmented across multiple packets, and in-order echo of
// marker packets.
type fakePZServer struct {
	ln        net.Listener
	responses map[string][]string // command -> response fragments
}

func newFakePZServer(t *testing.T, responses map[string][]string) *fakePZServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &fakePZServer{ln: ln, responses: responses}
	go s.serve()
	t.Cleanup(func() { ln.Close() })
	return s
}

func (s *fakePZServer) serve() {
	conn, err := s.ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	var pending [][3]interface{} // queued (id, type, body) replies
	for {
		id, ptype, body, err := readTestPacket(conn)
		if err != nil {
			return
		}
		switch ptype {
		case typeAuth:
			if body == "hunter2" {
				writeTestPacket(conn, id, typeResponseValue, "")
				writeTestPacket(conn, id, typeAuthResponse, "")
			} else {
				writeTestPacket(conn, -1, typeAuthResponse, "")
			}
		case typeExecCommand:
			frags, ok := s.responses[body]
			if !ok {
				frags = []string{"Unknown command " + body}
			}
			for _, f := range frags {
				pending = append(pending, [3]interface{}{id, int32(typeResponseValue), f})
			}
		case typeResponseValue:
			// Marker: flush pending command responses first (the "async
			// tick"), then echo the marker, preserving order.
			pending = append(pending, [3]interface{}{id, int32(typeResponseValue), ""})
			time.Sleep(50 * time.Millisecond)
			for _, p := range pending {
				writeTestPacket(conn, p[0].(int32), p[1].(int32), p[2].(string))
			}
			pending = nil
		}
	}
}

func readTestPacket(conn net.Conn) (int32, int32, string, error) {
	var sizeBuf [4]byte
	if _, err := io.ReadFull(conn, sizeBuf[:]); err != nil {
		return 0, 0, "", err
	}
	size := binary.LittleEndian.Uint32(sizeBuf[:])
	data := make([]byte, size)
	if _, err := io.ReadFull(conn, data); err != nil {
		return 0, 0, "", err
	}
	id := int32(binary.LittleEndian.Uint32(data[0:4]))
	ptype := int32(binary.LittleEndian.Uint32(data[4:8]))
	return id, ptype, string(data[8 : size-2]), nil
}

func writeTestPacket(conn net.Conn, id, ptype int32, body string) {
	size := int32(4 + 4 + len(body) + 2)
	buf := make([]byte, 0, 4+size)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(size))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(id))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(ptype))
	buf = append(buf, body...)
	buf = append(buf, 0, 0)
	conn.Write(buf)
}

func TestExecuteMultiPacketAsync(t *testing.T) {
	srv := newFakePZServer(t, map[string][]string{
		"stats performance all": {strings.Repeat("a", 4086), strings.Repeat("b", 2718)},
		"stats connection all":  {"\nzombies-killed-today: 8.0\n"},
	})

	c, err := dialConn(srv.ln.Addr().String(), "hunter2", 5*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	resp, err := c.execute("stats performance all")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(resp) != 4086+2718 {
		t.Errorf("fragmented response not reassembled: got %d bytes, want %d", len(resp), 4086+2718)
	}

	resp, err = c.execute("stats connection all")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	stats := ParseStatsResponse(resp)
	if stats["zombies-killed-today"] != 8.0 {
		t.Errorf("zombies-killed-today = %v, want 8", stats["zombies-killed-today"])
	}
}

func TestExecuteEmptyResponse(t *testing.T) {
	srv := newFakePZServer(t, map[string][]string{
		"stats network all": {""},
	})

	c, err := dialConn(srv.ln.Addr().String(), "hunter2", 5*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	resp, err := c.execute("stats network all")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if resp != "" {
		t.Errorf("expected empty response, got %q", resp)
	}
}

func TestAuthRefused(t *testing.T) {
	srv := newFakePZServer(t, nil)

	_, err := dialConn(srv.ln.Addr().String(), "wrong", 5*time.Second)
	if err == nil || !strings.Contains(err.Error(), "authentication refused") {
		t.Fatalf("expected authentication refused, got %v", err)
	}
}
