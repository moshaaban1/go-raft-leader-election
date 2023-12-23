package server

import (
	"fmt"
	"sync"

	"github.com/gofiber/fiber/v2"
	"shaaban.com/raft-leader-election/internal/scheduler"
)

// Raft leader election algorithm is implemented with a state machine in which a process is in one of three states:
// 1. Follower state: process recognizes another one as the leader
// 2. Candidate state: process proposes itself as the leader
// 3. Leader state: process is the leader
const (
	NodeLeader    = "leader"
	NodeFollower  = "follower"
	NodeCandidate = "candidate"
)

const (
	LeaderHeartBeatTimeOut = "leader_heartbeat_timeout"
	LeaderHeartBeat        = "leader_heartbeat"
)

type Server struct {
	nodeType  string
	app       *fiber.App
	cfg       *Config
	scheduler *scheduler.Scheduler
	leader    string
	mutex     sync.Mutex
	term      int
}

func New(config *Config) *Server {

	s := &Server{
		// All processes begin their journey as followers.
		nodeType: NodeFollower,
		app: fiber.New(fiber.Config{
			EnablePrintRoutes: true,
		}),
		cfg:       config,
		scheduler: scheduler.New(),
	}

	s.app.Post("/leader/heartbeat", s.LeaderHeartbeat)
	s.app.Post("/candidate/election", s.CandidateElection)

	// If the follower doesn’t receive any heartbeat within a certain time period, a timeout fires and the leader is presumed dead.
	err := s.scheduler.AddTask(LeaderHeartBeatTimeOut, &scheduler.Task{
		Interval: s.cfg.HeartBeatTimeout,
		Execute:  s.LeaderHeartbeatTimeOut,
		ErrFunc:  s.leaderHeartbeatTimeoutErrorFunc,
	})

	if err != nil {
		panic(err)
	}

	return s
}

func (s *Server) Start() error {
	return s.app.Listen(fmt.Sprintf(":%d", s.cfg.Port))
}

func (s *Server) Shutdown() {

}
