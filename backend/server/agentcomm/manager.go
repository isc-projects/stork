package agentcomm

import (
	"context"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	agentapi "isc.org/stork/api"
)

// Loop that receives requests to agents, sends to them, receives responses
// which are passed back to requestor. Requests and responses are passed
// via channels what guarantees that requests are forwarded to agents one
// by one.
func (agents *connectedAgentsData) communicationLoop() {
	for {
		select {
		case req, ok := <-agents.CommLoopReqs:
			if !ok {
				return
			}
			agents.handleRequest(req)
		case <-time.After(10 * time.Second):
			// To be implemented: gathering stats periodically
		}
	}
}

type channelResp struct {
	Response interface{}
	Err      error
}

type commLoopReq struct {
	AgentAddr string
	ReqData   interface{}
	RespChan  chan *channelResp
}

// Send a request to agent and receive response using channel to communication loop.
func (agents *connectedAgentsData) sendAndRecvViaQueue(agentAddr string, in interface{}) (interface{}, error) {
	respChan := make(chan *channelResp)
	req := &commLoopReq{AgentAddr: agentAddr, ReqData: in, RespChan: respChan}
	agents.CommLoopReqs <- req
	respErr := <-respChan
	return respErr.Response, respErr.Err
}

// Pass given request directly to an agent.
func doCall(ctx context.Context, agent *Agent, in interface{}) (interface{}, error) {
	var response interface{}
	var err error
	switch inData := in.(type) {
	case *agentapi.GetStateReq:
		response, err = agent.Client.GetState(ctx, inData)
	case *agentapi.ForwardRndcCommandReq:
		response, err = agent.Client.ForwardRndcCommand(ctx, inData)
	case *agentapi.ForwardToKeaOverHTTPReq:
		response, err = agent.Client.ForwardToKeaOverHTTP(ctx, inData)
	default:
		err = errors.New("doCall: unsupported request type")
	}
	return response, err
}

// Forward request received from channel to given agent and send back response
// via channel to requestor.
func (agents *connectedAgentsData) handleRequest(req *commLoopReq) {
	// get agent and its grpc connection
	agent, err := agents.GetConnectedAgent(req.AgentAddr)
	if err != nil {
		req.RespChan <- &channelResp{Response: nil, Err: err}
		return
	}

	// do call
	ctx := context.Background()
	response, err := doCall(ctx, agent, req.ReqData)

	if err != nil {
		// GetConnectedAgent remembers the grpc connection so it might
		// return an already existing connection.  This connection may
		// be broken so we should retry at least once.
		err2 := agent.MakeGrpcConnection()
		if err2 != nil {
			log.Warn(err)
			req.RespChan <- &channelResp{Response: nil, Err: errors.Wrap(err2, "problem with connection to agent")}
			return
		}

		// do call once again
		response, err = doCall(ctx, agent, req.ReqData)
		if err != nil {
			req.RespChan <- &channelResp{Response: nil, Err: errors.Wrap(err, "problem with connection to agent")}
			return
		}
	}

	req.RespChan <- &channelResp{Response: response, Err: err}
}
