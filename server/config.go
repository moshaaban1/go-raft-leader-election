package server

import "time"

type Config struct {
	Port              int
	Node              string
	Peers             []string
	HeartBeatInterval time.Duration
	HeartBeatTimeout  time.Duration
}
