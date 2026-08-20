package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

// repl 运行交互式命令行。
func repl(client *Client) {
	reader := bufio.NewReader(os.Stdin)
	for {
		prompt := fmt.Sprintf("%s:%d", client.cfg.Host, client.cfg.Port)
		if client.cfg.DB != 0 {
			prompt += fmt.Sprintf("[%d]", client.cfg.DB)
		}
		fmt.Printf("%s> ", prompt)

		line, err := reader.ReadString('\n')
		if err != nil {
			// EOF 或读取失败，正常退出
			fmt.Println()
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch strings.ToLower(line) {
		case "help", "?":
			printHelp()
			continue
		case "quit", "exit":
			return
		}

		args, err := tokenize(line)
		if err != nil {
			fmt.Printf("(error) %v\n", err)
			continue
		}
		if len(args) == 0 {
			continue
		}

		reply, err := client.Do(args...)
		if err != nil {
			// 连接可能已断开，尝试重连一次
			client.Close()
			if cerr := client.Connect(); cerr != nil {
				fmt.Printf("(error) %v\n", err)
				continue
			}
			reply, err = client.Do(args...)
			if err != nil {
				fmt.Printf("(error) %v\n", err)
				continue
			}
		}
		formatReply(reply)
	}
}

// tokenize 将命令行拆分为参数，支持双引号/单引号包裹及反斜杠转义。
func tokenize(s string) ([]string, error) {
	var args []string
	var cur strings.Builder
	var quote rune // 0 表示不在引号内
	escaped := false
	started := false

	flush := func() {
		args = append(args, cur.String())
		cur.Reset()
		started = false
	}

	for _, r := range s {
		if escaped {
			cur.WriteRune(r)
			escaped = false
			started = true
			continue
		}
		if quote != 0 {
			if r == '\\' && quote == '"' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
				continue
			}
			cur.WriteRune(r)
			started = true
			continue
		}
		switch {
		case r == '"' || r == '\'':
			quote = r
			started = true
		case r == ' ' || r == '\t':
			if started {
				flush()
			}
		default:
			cur.WriteRune(r)
			started = true
		}
	}
	if escaped {
		return nil, errors.New("unfinished escape")
	}
	if quote != 0 {
		return nil, fmt.Errorf("unfinished quote %c", quote)
	}
	if started {
		flush()
	}
	return args, nil
}

// printHelp 打印内置帮助。
func printHelp() {
	fmt.Println(`Usage:
  Type any Redis command to execute, e.g.:
    SET key value
    GET key
    LPUSH list a b c
    KEYS *
    CLUSTER INFO

Arguments can be quoted with " or ', use \ to escape inside double quotes.

Built-in commands:
  help, ?      Show this help
  quit, exit   Close the connection and exit`)
}
