package server

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

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

func (s *Server) sendHeartbeatToPeer(peer string, term int) (err error) {
	// TODO
	return nil
}
