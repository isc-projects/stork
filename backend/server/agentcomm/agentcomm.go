package agentcomm

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	agentapi "isc.org/stork/api"
)

// Settings specific to communication with Agents
type AgentsSettings struct {
	AgentListenAddress string `long:"agents-host" description:"the IP to listen on for communication from Stork Agent" default:"" env:"STORK_AGENTS_HOST"`
	AgentListenPort    int    `long:"agents-port" description:"the port to listen on for communication from Stork Agent" default:"8080" env:"STORK_AGENTS_PORT"`
}

// Runtime information about the agent, e.g. connection.
type Agent struct {
	Address  string
	Client   agentapi.AgentClient
	GrpcConn *grpc.ClientConn
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
	GetState(ctx context.Context, address string, agentPort int64) (*State, error)
	GetBind9State(ctx context.Context, agentAddress string, agentPort int64) (*Bind9State, error)
	ForwardToKeaOverHTTP(ctx context.Context, agentAddress string, agentPort int64, caURL string, commands []*KeaCommand, cmdResponses ...interface{}) (*KeaCmdsResult, error)
}

// Agents management map. It tracks Agents currently connected to the Server.
type connectedAgentsData struct {
	Settings  *AgentsSettings
	AgentsMap map[string]*Agent
}

// Create new ConnectedAgents objects.
func NewConnectedAgents(settings *AgentsSettings) ConnectedAgents {
	agents := connectedAgentsData{
		Settings:  settings,
		AgentsMap: make(map[string]*Agent),
	}
	return &agents
}

// Shutdown agents in agents map.
func (agents *connectedAgentsData) Shutdown() {
	for _, agent := range agents.AgentsMap {
		agent.GrpcConn.Close()
	}
}

// Get Agent object by its address.
func (agents *connectedAgentsData) GetConnectedAgent(address string) (*Agent, error) {
	// Look for agent in Agents map and if found then return it
	agent, ok := agents.AgentsMap[address]
	if ok {
		log.Printf("connecting to existing agent on %v", address)
		return agent, nil
	}

	// Agent not found so allocate agent and prepare connection
	agent = new(Agent)
	agent.Address = address
	err := agent.MakeGrpcConnection()
	if err != nil {
		log.Warn(err)
	}

	// Store it in Agents map
	agents.AgentsMap[address] = agent
	log.Printf("connecting to new agent on %v", address)

	return agent, nil
}
