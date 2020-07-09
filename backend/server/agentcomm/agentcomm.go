package agentcomm

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	agentapi "isc.org/stork/api"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/eventcenter"
)

// Settings specific to communication with Agents
type AgentsSettings struct {
}

// Holds runtime communication statistics with Kea daemons via
// a given agent.
type AgentKeaCommStats struct {
	CurrentErrorsCA      int64            // generall errors in communication to/via CA
	CurrentErrorsDaemons map[string]int64 // errors returned by particular daemon (including CA)
}

// Holds runtime communication statistics with Bind9 daemon for
// a given agent.
type AgentBind9CommStats struct {
	CurrentErrorsRNDC  int64
	CurrentErrorsStats int64
}

type AppCommStatsKey struct {
	Address string
	Port    int64
}

// Holds runtime statistics of communication with a given agent and
// with the apps behind this agent.
type AgentStats struct {
	CurrentErrors int64
	AppCommStats  map[AppCommStatsKey]interface{}
	mutex         *sync.Mutex
}

// Runtime information about the agent, e.g. connection, communication
// statistics.
type Agent struct {
	Address  string
	Client   agentapi.AgentClient
	GrpcConn *grpc.ClientConn
	Stats    AgentStats
}

// Prepare gRPC connection to agent.
func (agent *Agent) MakeGrpcConnection() error {
	// If there is any old connection then clean it up
	if agent.GrpcConn != nil {
		agent.GrpcConn.Close()
	}

	// Setup new connection
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())

	grpcConn, err := grpc.Dial(agent.Address, opts...)
	if err != nil {
		return errors.Wrapf(err, "problem with dial to agent %s", agent.Address)
	}

	agent.Client = agentapi.NewAgentClient(grpcConn)
	agent.GrpcConn = grpcConn

	return nil
}

// Interface for interacting with Agents via gRPC.
type ConnectedAgents interface {
	Shutdown()
	GetConnectedAgent(address string) (*Agent, error)
	GetConnectedAgentStats(adddress string, port int64) *AgentStats
	GetState(ctx context.Context, address string, agentPort int64) (*State, error)
	ForwardRndcCommand(ctx context.Context, dbApp *dbmodel.App, command string) (*RndcOutput, error)
	ForwardToNamedStats(ctx context.Context, agentAddress string, agentPort int64, statsAddress string, statsPort int64, path string, statsOutput interface{}) error
	ForwardToKeaOverHTTP(ctx context.Context, dbApp *dbmodel.App, commands []*KeaCommand, cmdResponses ...interface{}) (*KeaCmdsResult, error)
	TailTextFile(ctx context.Context, agentAddress string, agentPort int64, path string, offset int64) ([]string, error)
}

// Agents management map. It tracks Agents currently connected to the Server.
type connectedAgentsData struct {
	Settings     *AgentsSettings
	EventCenter  eventcenter.EventCenter
	AgentsMap    map[string]*Agent
	CommLoopReqs chan *commLoopReq
	DoneCommLoop chan bool
	Wg           *sync.WaitGroup
}

// Create new ConnectedAgents objects.
func NewConnectedAgents(settings *AgentsSettings, eventCenter eventcenter.EventCenter) ConnectedAgents {
	agents := connectedAgentsData{
		Settings:     settings,
		EventCenter:  eventCenter,
		AgentsMap:    make(map[string]*Agent),
		CommLoopReqs: make(chan *commLoopReq),
		DoneCommLoop: make(chan bool),
		Wg:           &sync.WaitGroup{},
	}

	agents.Wg.Add(1)
	go agents.communicationLoop()

	return &agents
}

// Shutdown agents in agents map.
func (agents *connectedAgentsData) Shutdown() {
	log.Printf("Stopping communication with agents")
	for _, agent := range agents.AgentsMap {
		agent.GrpcConn.Close()
	}

	close(agents.CommLoopReqs)
	agents.DoneCommLoop <- true
	agents.Wg.Wait()
	log.Printf("Stopped communication with agents")
}

// Get Agent object by its address.
func (agents *connectedAgentsData) GetConnectedAgent(address string) (*Agent, error) {
	// Look for agent in Agents map and if found then return it
	agent, ok := agents.AgentsMap[address]
	if ok {
		log.WithFields(log.Fields{
			"address": address,
		}).Info("connecting to existing agent")
		return agent, nil
	}

	// Agent not found so allocate agent and prepare connection
	agent = new(Agent)
	agent.Address = address
	agent.Stats.AppCommStats = make(map[AppCommStatsKey]interface{})
	agent.Stats.mutex = new(sync.Mutex)
	err := agent.MakeGrpcConnection()
	if err != nil {
		return nil, err
	}

	// Store it in Agents map
	agents.AgentsMap[address] = agent
	log.WithFields(log.Fields{
		"address": address,
	}).Info("connecting to new agent")

	return agent, nil
}

// Returns statistics for the connected agent. The statistics include number
// of errors to communicate with the agent and the number of errors to
// communicate with the apps behind the agent.
func (agents *connectedAgentsData) GetConnectedAgentStats(address string, port int64) *AgentStats {
	if port != 0 {
		address = fmt.Sprintf("%s:%d", address, port)
	}
	if agent, ok := agents.AgentsMap[address]; ok {
		return &agent.Stats
	}
	return nil
}
