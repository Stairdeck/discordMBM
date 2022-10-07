package minecraft

/*
	Author: https://github.com/njhanley
	Repo: https://github.com/njhanley/mollstam
*/

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	bits = 7
	msb  = 1 << bits
)

var (
	errUnexpectedPacket = errors.New("unexpected packet")
	errLengthMismatch   = errors.New("length mismatch")
)

type reader interface {
	io.Reader
	io.ByteReader
}

type writer interface {
	io.Writer
	io.ByteWriter
}

func readVarInt(r io.ByteReader) (x int32, err error) {
	var ux uint32
	for i := 0; ; i += bits {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		ux |= uint32(b&^msb) << i
		if b&msb == 0 {
			return int32(ux), nil
		}
	}
}

func writeVarInt(w io.ByteWriter, x int32) error {
	for ux := uint32(x); ; ux >>= bits {
		b := byte(ux) &^ msb
		if uint32(b) < ux {
			b |= msb
		}
		err := w.WriteByte(b)
		if err != nil {
			return err
		}
		if b&msb == 0 {
			return nil
		}
	}
}

func readString(r reader) (s string, err error) {
	n, err := readVarInt(r)
	if err != nil {
		return "", err
	}
	buf := new(strings.Builder)
	_, err = io.CopyN(buf, r, int64(n))
	return buf.String(), err
}

func writeString(w writer, s string) error {
	err := writeVarInt(w, int32(len(s)))
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, s)
	return err
}

func readPacket(r reader) (id int32, data []byte, err error) {
	n, err := readVarInt(r)
	if err != nil {
		return 0, nil, err
	}
	buf := new(bytes.Buffer)
	_, err = io.CopyN(buf, r, int64(n))
	if err != nil {
		return 0, nil, err
	}
	id, err = readVarInt(buf)
	if err != nil {
		return 0, nil, err
	}
	return id, buf.Bytes(), err
}

func writePacket(w writer, id int32, data []byte) error {
	buf := new(bytes.Buffer)
	err := writeVarInt(buf, id)
	if err != nil {
		return err
	}
	err = writeVarInt(w, int32(buf.Len()+len(data)))
	if err != nil {
		return err
	}
	_, err = buf.WriteTo(w)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func splitAddress(addr string) (host string, port uint16, err error) {
	host, _port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}
	u, err := strconv.ParseUint(_port, 10, 16)
	return host, uint16(u), err
}

func queryMinecraft(addr string, timeout time.Duration) (online int, players []string, err error) {
	host, port, err := splitAddress(addr)
	if err != nil {
		return 0, nil, err
	}

	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return 0, nil, err
	}
	defer conn.Close()

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	buf := new(bytes.Buffer)

	// handshake
	writeVarInt(buf, -1) // protocol version
	writeString(buf, host)
	binary.Write(buf, binary.BigEndian, port)
	writeVarInt(buf, 1) // next state
	err = writePacket(w, 0, buf.Bytes())
	if err != nil {
		return 0, nil, err
	}
	err = w.Flush()
	if err != nil {
		return 0, nil, err
	}
	buf.Reset()

	// request
	err = writePacket(w, 0, buf.Bytes())
	if err != nil {
		return 0, nil, err
	}
	err = w.Flush()
	if err != nil {
		return 0, nil, err
	}
	buf.Reset()

	// response
	id, data, err := readPacket(r)
	if err != nil {
		return 0, nil, err
	}
	if id != 0 {
		return 0, nil, fmt.Errorf("%w: id=%d, expected=%d", errUnexpectedPacket, id, 0)
	}
	buf = bytes.NewBuffer(data)
	s, err := readString(buf)
	if err != nil {
		return 0, nil, err
	}

	var status struct {
		Players struct {
			Online int `json:"online"`
			Sample []struct {
				Name string `json:"name"`
			} `json:"sample"`
		} `json:"players"`
	}
	err = json.Unmarshal([]byte(s), &status)
	if err != nil {
		return 0, nil, err
	}

	players = make([]string, len(status.Players.Sample))
	for i := range status.Players.Sample {
		players[i] = status.Players.Sample[i].Name
	}
	sort.Strings(players)

	return status.Players.Online, players, nil
}
