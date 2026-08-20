package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strconv"
	"time"
)

// ClientConfig 是 Redis 客户端连接配置。
type ClientConfig struct {
	Host     string
	Port     int
	User     string        // 可选的 ACL 用户名，需配合 Password 使用
	Password string        // 连接密码
	DB       int           // 初始选择的数据库编号
	Timeout  time.Duration // 单条命令的超时时间，0 表示不超时
}

// Client 是一个支持任意 Redis 命令执行的客户端。
type Client struct {
	cfg    ClientConfig
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

// NewClient 根据配置创建一个新的 Redis 客户端（尚未建立连接）。
func NewClient(cfg ClientConfig) *Client {
	return &Client{cfg: cfg}
}

// Addr 返回 host:port 形式的地址。
func (c *Client) Addr() string {
	return net.JoinHostPort(c.cfg.Host, strconv.Itoa(c.cfg.Port))
}

// IsConnected 返回当前是否已建立连接。
func (c *Client) IsConnected() bool {
	return c.conn != nil
}

// Connect 建立 TCP 连接，并按需执行 AUTH 和 SELECT。
func (c *Client) Connect() error {
	if c.conn != nil {
		return nil
	}
	conn, err := net.DialTimeout("tcp", c.Addr(), c.cfg.Timeout)
	if err != nil {
		return err
	}
	c.conn = conn
	c.reader = bufio.NewReader(conn)
	c.writer = bufio.NewWriter(conn)

	if c.cfg.Password != "" {
		args := []string{"AUTH", c.cfg.Password}
		if c.cfg.User != "" {
			args = []string{"AUTH", c.cfg.User, c.cfg.Password}
		}
		reply, err := c.do(args...)
		if err != nil {
			c.Close()
			return fmt.Errorf("AUTH: %w", err)
		}
		if e, ok := reply.(ErrorReply); ok {
			c.Close()
			return fmt.Errorf("AUTH failed: %s", string(e))
		}
	}
	if c.cfg.DB != 0 {
		reply, err := c.do("SELECT", strconv.Itoa(c.cfg.DB))
		if err != nil {
			c.Close()
			return fmt.Errorf("SELECT: %w", err)
		}
		if e, ok := reply.(ErrorReply); ok {
			c.Close()
			return fmt.Errorf("SELECT failed: %s", string(e))
		}
	}
	return nil
}

// Do 执行一条 Redis 命令（第一个参数为命令名，其余为参数）。
// 若尚未连接会自动建立连接。返回解码后的响应。
func (c *Client) Do(args ...string) (Reply, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}
	reply, err := c.do(args...)
	if err != nil {
		c.Close()
		return nil, err
	}
	return reply, nil
}

// do 在已连接的前提下执行命令，不处理连接建立与关闭。
func (c *Client) do(args ...string) (Reply, error) {
	parts := make([][]byte, len(args))
	for i, a := range args {
		parts[i] = []byte(a)
	}
	if err := c.writeCommand(parts); err != nil {
		return nil, err
	}
	return c.readReply()
}

// writeCommand 编码并发送命令，附带写超时。
func (c *Client) writeCommand(args [][]byte) error {
	if c.cfg.Timeout > 0 {
		_ = c.conn.SetWriteDeadline(time.Now().Add(c.cfg.Timeout))
	}
	var buf bytes.Buffer
	buf.Grow(64)
	buf.WriteByte('*')
	buf.WriteString(strconv.Itoa(len(args)))
	buf.WriteString("\r\n")
	for _, arg := range args {
		buf.WriteByte('$')
		buf.WriteString(strconv.Itoa(len(arg)))
		buf.WriteString("\r\n")
		buf.Write(arg)
		buf.WriteString("\r\n")
	}
	if _, err := c.writer.Write(buf.Bytes()); err != nil {
		return err
	}
	return c.writer.Flush()
}

// readReply 读取响应并附带读超时。
func (c *Client) readReply() (Reply, error) {
	if c.cfg.Timeout > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.cfg.Timeout))
	}
	reply, err := readReply(c.reader)
	if err != nil {
		c.Close()
		return nil, err
	}
	if c.cfg.Timeout > 0 {
		_ = c.conn.SetReadDeadline(time.Time{})
	}
	return reply, nil
}

// Close 关闭连接。
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	c.reader = nil
	c.writer = nil
	return err
}
