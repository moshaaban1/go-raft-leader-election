package server

import (
	"golang.org/x/exp/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

type heartbeatRequest struct {
	Leader string
	Term   int
}

type heartbeatResponse struct {
	Ack  bool
	Node string
}

type CandidateProposalReq struct {
	Candidate string
	Term      int
}

type CandidateProposalRes struct {
	Ack  bool
	Node string
}

func (s *Server) LeaderHeartbeat(c *fiber.Ctx) error {
	var req heartbeatRequest

	if err := c.BodyParser(&req); err != nil {
		return err
	}

	res := heartbeatResponse{
		Ack:  true,
		Node: s.cfg.Node,
	}

	s.mutex.Lock()
	if req.Term < s.term {
		res.Ack = false
	}
	s.leader = req.Leader
	s.term = req.Term

	s.scheduler.ResetTaskSchedule(LeaderHeartBeatTimeOut, s.cfg.HeartBeatTimeout)

	s.mutex.Unlock()

	if res.Ack {
		slog.Info("acknowledged heartbeat from leader", "leader", req.Leader)
		return c.JSON(&res)
	}

	slog.Warn("heartbeat rejected", "node", req.Leader)
	return c.Status(fasthttp.StatusPreconditionFailed).JSON(&res)
}

func (s *Server) CandidateElection(c *fiber.Ctx) error {
	var rq CandidateProposalReq
	if err := c.BodyParser(&rq); err != nil {
		return err
	}
	slog.Info("received a candidate proposal", "candidate", rq.Candidate)

	rs := CandidateProposalRes{
		Ack:  s.processCandidateProposal(rq.Candidate, rq.Term),
		Node: s.cfg.Node,
	}

	if rs.Ack {
		slog.Info("new leader candidate accepted", "leader", rq.Candidate)
		return c.JSON(&rs)
	}

	slog.Warn("candidate proposal rejected", "node", rq.Candidate)
	return c.Status(fasthttp.StatusPreconditionFailed).JSON(&rs)
}

// Each process can vote for at most one candidate in a term on a first-come-first-served basis
func (s *Server) processCandidateProposal(candidate string, term int) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// If candidate's term is less than the current term, it will be ignored means that the process votes for no candidate in the current term
	if term <= s.term {
		return false
	}

	s.nodeType = NodeFollower
	s.leader = candidate
	s.term = term

	s.scheduler.ResetTaskSchedule(LeaderHeartBeatTimeOut, s.cfg.HeartBeatTimeout)

	return true
}
