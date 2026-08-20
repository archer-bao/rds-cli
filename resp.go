package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// Reply 是 Redis 响应值的抽象类型，可能为以下具体类型之一：
//   - SimpleString（简单字符串，+）
//   - ErrorReply（错误，-）
//   - IntReply（整数，:）
//   - BulkReply（批量字符串，$；nil 表示空值）
//   - ArrayReply（数组，*；nil 表示空数组）
type Reply interface{}

// SimpleString 对应 RESP 简单字符串。
type SimpleString string

// ErrorReply 对应 RESP 错误。
type ErrorReply string

// IntReply 对应 RESP 整数。
type IntReply int64

// BulkReply 对应 RESP 批量字符串，nil 表示空值（null bulk string）。
type BulkReply []byte

// ArrayReply 对应 RESP 数组，nil 表示空数组（null array）。
type ArrayReply []Reply

// writeCommand 将一个命令（参数数组）编码为 RESP 协议并写入 writer。
func writeCommand(w *bufio.Writer, args [][]byte) error {
	if len(args) == 0 {
		return errors.New("empty command")
	}
	header := make([]byte, 0, 32)
	header = append(header, '*')
	header = strconv.AppendInt(header, int64(len(args)), 10)
	header = append(header, '\r', '\n')
	if _, err := w.Write(header); err != nil {
		return err
	}
	for _, arg := range args {
		part := make([]byte, 0, len(arg)+16)
		part = append(part, '$')
		part = strconv.AppendInt(part, int64(len(arg)), 10)
		part = append(part, '\r', '\n')
		part = append(part, arg...)
		part = append(part, '\r', '\n')
		if _, err := w.Write(part); err != nil {
			return err
		}
	}
	return nil
}

// readReply 从 reader 读取并解码一个 RESP 响应。
func readReply(r *bufio.Reader) (Reply, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if len(line) < 3 || line[len(line)-2] != '\r' {
		return nil, errors.New("protocol error: malformed line")
	}
	prefix := line[0]
	body := line[1 : len(line)-2]

	switch prefix {
	case '+':
		return SimpleString(body), nil
	case '-':
		return ErrorReply(body), nil
	case ':':
		n, err := strconv.ParseInt(body, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("protocol error: invalid integer %q", body)
		}
		return IntReply(n), nil
	case '$':
		n, err := strconv.Atoi(body)
		if err != nil {
			return nil, fmt.Errorf("protocol error: invalid bulk length %q", body)
		}
		if n == -1 {
			return BulkReply(nil), nil
		}
		if n < 0 {
			return nil, errors.New("protocol error: negative bulk length")
		}
		data := make([]byte, n+2)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, err
		}
		if data[n] != '\r' || data[n+1] != '\n' {
			return nil, errors.New("protocol error: bulk string terminator")
		}
		return BulkReply(data[:n]), nil
	case '*':
		n, err := strconv.Atoi(body)
		if err != nil {
			return nil, fmt.Errorf("protocol error: invalid array length %q", body)
		}
		if n == -1 {
			return ArrayReply(nil), nil
		}
		if n < 0 {
			return nil, errors.New("protocol error: negative array length")
		}
		items := make(ArrayReply, n)
		for i := 0; i < n; i++ {
			item, err := readReply(r)
			if err != nil {
				return nil, err
			}
			items[i] = item
		}
		return items, nil
	default:
		return nil, fmt.Errorf("protocol error: unexpected type byte %q", string(prefix))
	}
}
