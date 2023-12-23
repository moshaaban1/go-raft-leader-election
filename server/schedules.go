package server

import (
	"sync"

	"github.com/dyweb/gommon/errors"

	"golang.org/x/exp/slog"
	"shaaban.com/raft-leader-election/internal/scheduler"
)

// A follower expects to receive a periodic heartbeat from the leader containing the election term the leader was elected in.
// If the follower doesn’t receive any heartbeat within a certain time period, a timeout fires and the leader is presumed dead
func (s *Server) LeaderHeartbeatTimeOut() error {
	var (
		wg    sync.WaitGroup
		err   []error
		peers = s.cfg.Peers
		term  = s.incrementTerm() // start a new election by incrementing current election term
	)

	// Update current node's state to candidate
	// The process remains in the candidate state until one of three things happens:
	// 1. The candidate wins the election
	// 2. Another candidate wins the election
	// 3. A period of time goes by with no winner
	s.updateCurrentNodeToCandidate()

	wg.Add(len(peers))

	for _, peer := range peers {
		go func(peer string) {
			defer wg.Done()

			// Send candidate proposal to peers
			e := s.sendLeaderElectionToCandidates(peer, term)

			if e != nil {
				err = append(err, e)
			}

		}(peer)
	}

	wg.Wait()

	if len(err) > len(peers)/2 {
		// The candidate loses the election or there is no winner
		return errors.New("error encountered when sending candidate proposal to peers")
	}

	// The candidate wins the election and transition to leader state
	s.upgradeSelfNodeToLeader()

	return nil
}

func (s *Server) leaderHeartbeatTimeoutErrorFunc(err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.nodeType = NodeFollower

	slog.Error("error encountered on task leader_heartbeat_timeout", "error", err)
}

func (s *Server) heartbeatFunc() error {
	var (
		wg   sync.WaitGroup
		merr = errors.NewMultiErrSafe()
		term = s.currTerm()
	)
	slog.Info("running leader heartbeat scheduled task", "node", s.cfg.Node, "term", term)

	wg.Add(len(s.cfg.Peers))

	for _, peer := range s.cfg.Peers {
		go func(peer string) {
			err := s.sendHeartbeatToPeer(peer, term)
			if err != nil {
				merr.Append(err)
			}
			wg.Done()
		}(peer)
	}

	wg.Wait()
	if merr.HasError() && merr.Len() > len(s.cfg.Peers)/2 {
		slog.Error("error encountered when sending heartbeat to peers", "error", merr, "term", term)
		return merr
	}

	return nil
}

func (s *Server) heartbeatErrorFunc(err error) {
	// TODO
}

func (s *Server) incrementTerm() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.term++

	return s.term
}

func (s *Server) currTerm() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.term
}

func (s *Server) updateCurrentNodeToCandidate() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.nodeType = NodeCandidate
}

func (s *Server) upgradeSelfNodeToLeader() {
	slog.Info("upgrade self node to leader")
	defer s.mutex.Unlock()

	s.mutex.Lock()
	s.nodeType = NodeLeader
	s.leader = ""

	s.scheduler.RemoveTask(LeaderHeartBeatTimeOut)

	s.scheduler.AddTask(LeaderHeartBeat, &scheduler.Task{
		Interval: s.cfg.HeartBeatInterval,
		Execute:  s.heartbeatFunc,
		ErrFunc:  s.heartbeatErrorFunc,
	})

	slog.Info("leader-candidate proposal successful; promoted self to leader", "node", s.cfg.Node)

}
