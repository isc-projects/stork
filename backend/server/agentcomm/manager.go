package agentcomm

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"

	agentapi "isc.org/stork/api"
	"isc.org/stork/appdata/bind9stats"
)

// Loop that receives requests to agents, sends to them, receives responses
// which are passed back to requestor. Requests and responses are passed
// via channels what guarantees that requests are forwarded to agents one
// by one.
func (agents *connectedAgentsData) communicationLoop() {
	defer agents.Wg.Done()
	for {
		select {
		// wait for requests from parties that want to talk to agents
		case req := <-agents.CommLoopReqs:
			if req != nil {
				agents.handleRequest(req)
			}
		// wait for done signal from shutdown function
		case <-agents.DoneCommLoop:
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

func (agents *connectedAgentsData) receiveZones(ctx context.Context, agentAddress string, agentPort int64, controlAddress string, controlPort int64, filter *bind9stats.ZoneFilter) (grpc.ServerStreamingClient[agentapi.Zone], func(), error) {
	agentAddressPort := net.JoinHostPort(agentAddress, strconv.FormatInt(agentPort, 10))

	client, conn, err := makeGrpcConnection(agentAddressPort, agents.caCertPEM, agents.serverCertPEM, agents.serverKeyPEM)
	if err != nil {
		return nil, nil, err
	}
	defer conn.Close()

	request := &agentapi.ReceiveZonesReq{
		ControlAddress: controlAddress,
		ControlPort:    controlPort,
	}
	if filter != nil {
		if filter.View != nil {
			request.ViewName = *filter.View
		}
		if filter.Limit != nil {
			request.Limit = int64(*filter.Limit)
		}
	}
	stream, err := client.ReceiveZones(ctx, request)
	if err != nil {
		return nil, nil, err
	}
	return stream, func() {
		conn.Close()
	}, nil
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
	// The options passed to the commands that can receive a big response (>4MiB).
	compressOption := grpc.UseCompressor(gzip.Name)
	// The biggest reported response had 5.4MB. Default limit is 4MiB.
	// The message limit applies to the decompressed message.
	// Set limit to 40MiB.
	increaseLimitOption := grpc.MaxCallRecvMsgSize(4 * 10 * 1024 * 1024)
	bigMessageOptions := []grpc.CallOption{compressOption, increaseLimitOption}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	switch inData := in.(type) {
	case *agentapi.PingReq:
		response, err = agent.Client.Ping(ctx, inData)
	case *agentapi.GetStateReq:
		response, err = agent.Client.GetState(ctx, inData, bigMessageOptions...)
	case *agentapi.ForwardRndcCommandReq:
		response, err = agent.Client.ForwardRndcCommand(ctx, inData, bigMessageOptions...)
	case *agentapi.ForwardToNamedStatsReq:
		response, err = agent.Client.ForwardToNamedStats(ctx, inData, bigMessageOptions...)
	case *agentapi.ForwardToKeaOverHTTPReq:
		response, err = agent.Client.ForwardToKeaOverHTTP(ctx, inData, bigMessageOptions...)
	case *agentapi.TailTextFileReq:
		response, err = agent.Client.TailTextFile(ctx, inData, bigMessageOptions...)
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
		err2 := agent.MakeGrpcConnection(agents.caCertPEM, agents.serverCertPEM, agents.serverKeyPEM)
		if err2 != nil {
			log.WithFields(log.Fields{
				"agent": agent.Address,
			}).Warn(err)
			req.RespChan <- &channelResp{
				Response: nil,
				Err:      errors.WithMessagef(err2, "grpc manager is unable to re-establish connection with the agent %s", agent.Address),
			}
			return
		}

		// do call once again
		response, err2 = doCall(ctx, agent, req.ReqData)
		if err2 != nil {
			log.WithFields(log.Fields{
				"agent": agent.Address,
			}).Warn(err)
			req.RespChan <- &channelResp{
				Response: nil,
				Err:      errors.WithMessagef(err2, "grpc manager is unable to re-establish connection with the agent %s", agent.Address),
			}
			return
		}
	}

	req.RespChan <- &channelResp{Response: response, Err: nil}
}
