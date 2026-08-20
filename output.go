package main

import (
	"fmt"
	"strings"
)

// formatReply 以类似 redis-cli 的风格打印一条命令的响应。
func formatReply(reply Reply) {
	switch v := reply.(type) {
	case SimpleString:
		fmt.Println(string(v))
	case ErrorReply:
		fmt.Printf("(error) %s\n", string(v))
	case IntReply:
		fmt.Printf("(integer) %d\n", int64(v))
	case BulkReply:
		if v == nil {
			fmt.Println("(nil)")
		} else {
			fmt.Println(quoteString(string(v)))
		}
	case ArrayReply:
		writeArray(v, "", "")
	default:
		fmt.Printf("%v\n", reply)
	}
}

// writeArray 打印数组元素，prefix 为当前行起始前缀，pad 为续行对齐空白。
func writeArray(v ArrayReply, prefix, pad string) {
	if v == nil {
		fmt.Printf("%s(nil)\n", prefix)
		return
	}
	if len(v) == 0 {
		fmt.Printf("%s(empty array)\n", prefix)
		return
	}
	for i, item := range v {
		idx := fmt.Sprintf("%d) ", i+1)
		if i == 0 {
			writeItem(item, prefix+idx, pad+strings.Repeat(" ", len(idx)))
		} else {
			writeItem(item, pad+idx, pad+strings.Repeat(" ", len(idx)))
		}
	}
}

// writeItem 打印数组中的一个元素（可能嵌套）。
func writeItem(item Reply, prefix, pad string) {
	switch v := item.(type) {
	case SimpleString:
		fmt.Printf("%s%s\n", prefix, string(v))
	case ErrorReply:
		fmt.Printf("%s(error) %s\n", prefix, string(v))
	case IntReply:
		fmt.Printf("%s(integer) %d\n", prefix, int64(v))
	case BulkReply:
		if v == nil {
			fmt.Printf("%s(nil)\n", prefix)
		} else {
			fmt.Printf("%s%s\n", prefix, quoteString(string(v)))
		}
	case ArrayReply:
		writeArray(v, prefix, pad)
	default:
		fmt.Printf("%s%v\n", prefix, item)
	}
}

// quoteString 将字符串转义并用双引号包裹，便于阅读。
func quoteString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 32 || r == 127 {
				fmt.Fprintf(&b, `\x%02x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
