package agentcomm

import (
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"github.com/pkg/errors"

	"isc.org/stork/api"
)

// Settings specific to communication with Agents
type AgentsSettings struct {
	AgentListenAddress string  `long:"agents-host" description:"the IP to listen on for communication from Stork Agent" default:"" env:"STORK_AGENTS_HOST"`
	AgentListenPort    int     `long:"agents-port" description:"the port to listen on for communication from Stork Agent" default:"8080" env:"STORK_AGENTS_PORT"`
}

// Runtime information about the agent, e.g. connection.
type Agent struct {
	Address string
	Client agentapi.AgentClient
	GrpcConn *grpc.ClientConn
}

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

// Agents management map. It tracks Agents currently connected to the Server.
type ConnectedAgents struct {
	Settings AgentsSettings
	AgentsMap map[string]*Agent
}

// Initialize Agents map
func (agents *ConnectedAgents) Init() error {
	agents.AgentsMap = make(map[string]*Agent)
	return nil
}

// Shutdown agents in agents map
func (agents *ConnectedAgents) Shutdown() {
	for _, agent := range agents.AgentsMap {
		agent.GrpcConn.Close()
	}
}

func (agents *ConnectedAgents) GetConnectedAgent(address string) (*Agent, error) {
	// Look for agent in Agents map and if found then return it
	agent, ok := agents.AgentsMap[address]
	if ok {
		log.Printf("connecting to existing agent from %v", address)
		return agent, nil
	}

	// Agent not found so allocate agent and prepare connection
	agent = new(Agent)
	agent.Address = address
	agent.MakeGrpcConnection()

	// Store it in Agents map
	agents.AgentsMap[address] = agent
	log.Printf("connecting to new agent from %v", address)

	return agent, nil
}
