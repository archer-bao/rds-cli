package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

const version = "0.1.0"

func main() {
	fs := flag.NewFlagSet("rds-cli", flag.ContinueOnError)

	host := fs.String("h", "127.0.0.1", "Server hostname")
	port := fs.Int("p", 6379, "Server port")
	password := fs.String("a", "", "Password to use when connecting to the server (fallback to $REDISCLI_AUTH)")
	user := fs.String("u", "", "Username for AUTH when using Redis ACL (requires -a)")
	db := fs.Int("n", 0, "Database number")
	timeout := fs.Int("t", 60, "Per-command timeout in seconds (0 = no timeout)")
	showVersion := fs.Bool("v", false, "Print version and exit")
	showHelp := fs.Bool("help", false, "Show this help")

	fs.Usage = func() {
		out := fs.Output()
		fmt.Fprintf(out, "rds-cli %s - a lightweight Redis command-line client\n\n", version)
		fmt.Fprintf(out, "Usage: %s [OPTIONS] [cmd [arg [arg ...]]]\n\n", fs.Name())
		fmt.Fprintf(out, "With a command, run it once and exit. Without a command, start an interactive shell.\n\n")
		fmt.Fprintf(out, "Options:\n")
		fs.PrintDefaults()
		fmt.Fprintf(out, "\nExamples:\n")
		fmt.Fprintf(out, "  %s PING\n", fs.Name())
		fmt.Fprintf(out, "  %s -h 10.0.0.1 -p 6380 -a secret SET foo bar\n", fs.Name())
		fmt.Fprintf(out, "  %s -n 1 GET mykey\n", fs.Name())
		fmt.Fprintf(out, "  %s\n", fs.Name())
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if *showHelp {
		fs.Usage()
		return
	}
	if *showVersion {
		fmt.Printf("rds-cli %s\n", version)
		return
	}
	if *password == "" {
		if env := os.Getenv("REDISCLI_AUTH"); env != "" {
			*password = env
		}
	}

	client := NewClient(ClientConfig{
		Host:     *host,
		Port:     *port,
		User:     *user,
		Password: *password,
		DB:       *db,
		Timeout:  time.Duration(*timeout) * time.Second,
	})

	args := fs.Args()

	// 交互模式
	if len(args) == 0 {
		if err := client.Connect(); err != nil {
			fmt.Fprintf(os.Stderr, "Could not connect to Redis at %s:%d: %v\n", *host, *port, err)
			os.Exit(1)
		}
		fmt.Printf("Connected to Redis at %s:%d\n", *host, *port)
		fmt.Println("Type 'help' for help, 'quit' to exit.")
		repl(client)
		return
	}

	// 单条命令模式
	reply, err := client.Do(args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to Redis at %s:%d: %v\n", *host, *port, err)
		os.Exit(1)
	}
	formatReply(reply)
}
