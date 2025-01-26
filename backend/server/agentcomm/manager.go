package agentcomm

import (
	"context"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"

	agentapi "isc.org/stork/api"
)

// Loop that receives requests to agents, sends to them, receives responses
// which are passed back to requestor. Requests and responses are passed
// via channels what guarantees that requests are forwarded to agents one
// by one.
func (agents *connectedAgentsImpl) communicationLoop() {
	defer agents.wg.Done()
	for {
		select {
		// wait for requests from parties that want to talk to agents
		case req := <-agents.commLoopReqs:
			if req != nil {
				agents.handleRequest(req)
			}
		// wait for done signal from shutdown function
		case <-agents.doneCommLoop:
			return
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
func (agents *connectedAgentsImpl) sendAndRecvViaQueue(agentAddr string, in interface{}) (interface{}, error) {
	respChan := make(chan *channelResp)
	req := &commLoopReq{AgentAddr: agentAddr, ReqData: in, RespChan: respChan}
	agents.commLoopReqs <- req
	respErr := <-respChan
	return respErr.Response, respErr.Err
}

// Pass given request directly to an agent.
func doCall(ctx context.Context, agent *agentState, in interface{}) (interface{}, error) {
	var response interface{}
	var err error
	// The options passed to the commands that can receive a big response (>4MiB).
	compressOption := grpc.UseCompressor(gzip.Name)
	// The biggest reported response had 5.4MB. Default limit is 4MiB.
	// The message limit applies to the decompressed message.
	// Set limit to 40MiB.
	increaseLimitOption := grpc.MaxCallRecvMsgSize(4 * 10 * 1024 * 1024)
	bigMessageOptions := []grpc.CallOption{compressOption, increaseLimitOption}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	client := agent.connector.createClient()
	switch inData := in.(type) {
	case *agentapi.PingReq:
		response, err = client.Ping(ctx, inData)
	case *agentapi.GetStateReq:
		response, err = client.GetState(ctx, inData, bigMessageOptions...)
	case *agentapi.ForwardRndcCommandReq:
		response, err = client.ForwardRndcCommand(ctx, inData, bigMessageOptions...)
	case *agentapi.ForwardToNamedStatsReq:
		response, err = client.ForwardToNamedStats(ctx, inData, bigMessageOptions...)
	case *agentapi.ForwardToKeaOverHTTPReq:
		response, err = client.ForwardToKeaOverHTTP(ctx, inData, bigMessageOptions...)
	case *agentapi.TailTextFileReq:
		response, err = client.TailTextFile(ctx, inData, bigMessageOptions...)
	default:
		err = errors.New("doCall: unsupported request type")
	}

	return response, err
}

// Forward request received from channel to given agent and send back response
// via channel to requestor.
func (agents *connectedAgentsImpl) handleRequest(req *commLoopReq) {
	// get agent and its grpc connection
	agent, err := agents.getConnectedAgent(req.AgentAddr)
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
		err2 := agent.connector.connect()
		if err2 != nil {
			log.WithFields(log.Fields{
				"agent": agent.address,
			}).Warn(err)
			req.RespChan <- &channelResp{
				Response: nil,
				Err:      errors.WithMessagef(err2, "grpc manager is unable to re-establish connection with the agent %s", agent.address),
			}
			return
		}

		// do call once again
		response, err2 = doCall(ctx, agent, req.ReqData)
		if err2 != nil {
			log.WithFields(log.Fields{
				"agent": agent.address,
			}).Warn(err)
			req.RespChan <- &channelResp{
				Response: nil,
				Err:      errors.WithMessagef(err2, "grpc manager is unable to re-establish connection with the agent %s", agent.address),
			}
			return
		}
	}

	req.RespChan <- &channelResp{Response: response, Err: nil}
}
