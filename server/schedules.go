package server

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dyweb/gommon/errors"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
	"golang.org/x/exp/slog"
	"shaaban.com/raft-leader-election/internal/scheduler"
)

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

func (s *Server) sendHeartbeatToPeer(peer string, term int) (err error) {
	// TODO
	return nil
}

func (s *Server) heartbeatErrorFunc(err error) {
	// TODO
}

func (s *Server) LeaderHeartbeatTimeOut() error {
	var (
		wg    sync.WaitGroup
		err   []error
		peers = s.cfg.Peers
		term  = s.incrementTerm()
	)

	wg.Add(len(peers))

	for _, peer := range peers {
		go func(peer string) {
			defer wg.Done()

			e := s.sendLeaderElectionToCandidates(peer, term)

			if e != nil {
				err = append(err, e)
			}

		}(peer)
	}

	wg.Wait()

	if len(err) > len(peers)/2 {
		return errors.New("error encountered when sending candidate proposal to peers")
	}

	s.upgradeSelfNodeToLeader()

	return nil
}

func (s *Server) sendLeaderElectionToCandidates(peer string, term int) (err error) {
	client := fiber.AcquireClient()
	defer fiber.ReleaseClient(client)

	agent := client.Post(fmt.Sprintf("http://%s/candidate/proposal", peer)).JSON(&CandidateProposalReq{
		Candidate: s.cfg.Node,
		Term:      term,
	})

	rs := fiber.AcquireResponse()
	defer fiber.ReleaseResponse(rs)
	if err = fasthttp.Do(agent.Request(), rs); err != nil {
		return err
	}
	if rs.StatusCode() != fasthttp.StatusOK {
		err = fmt.Errorf("status code %d", rs.StatusCode())
		return err
	}

	var res CandidateProposalRes
	if err = json.Unmarshal(rs.Body(), &res); err != nil {
		return err
	}
	if !res.Ack {
		err = fmt.Errorf("failed to accept proposal from peer %s for term %d", peer, term)
		return err
	}

	return nil
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

func (s *Server) leaderHeartbeatTimeoutErrorFunc(err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.nodeType = NodeFollower

	slog.Error("error encountered on task leader_heartbeat_timeout", "error", err)
}
