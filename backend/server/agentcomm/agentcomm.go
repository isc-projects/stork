package agentcomm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"strconv"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"google.golang.org/grpc/security/advancedtls"

	agentapi "isc.org/stork/api"
	keactrl "isc.org/stork/appctrl/kea"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/eventcenter"
)

// Settings specific to communication with Agents.
type AgentsSettings struct{}

// Runtime information about the agent, e.g. connection, communication
// statistics.
type Agent struct {
	Address  string
	Client   agentapi.AgentClient
	GrpcConn *grpc.ClientConn
	Stats    *AgentCommStats
}

// The GRPC client callback to perform extra verification of the peer
// certificate.
// The callback is running at the end of server certificate verification.
func verifyPeer(params *advancedtls.VerificationFuncParams) (*advancedtls.VerificationResults, error) {
	// The peer must have the extended key usage set.
	if len(params.Leaf.ExtKeyUsage) == 0 {
		return nil, errors.New("peer certificate does not have the extended key usage set")
	}
	return &advancedtls.VerificationResults{}, nil
}

// Prepare TLS credentials with configured certs and verification options.
func prepareTLSCreds(caCertPEM, serverCertPEM, serverKeyPEM []byte) (credentials.TransportCredentials, error) {
	// Load the certificates from disk
	certificate, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		return nil, errors.Wrapf(err, "could not load client key pair")
	}

	// Create a certificate pool from the certificate authority
	certPool := x509.NewCertPool()

	// Append the client certificates from the CA
	if ok := certPool.AppendCertsFromPEM(caCertPEM); !ok {
		return nil, errors.New("failed to append CA certs")
	}

	// Prepare structure for advanced TLS with custom agent verification.
	// It sets root CA cert and stork server cert. It also defines hook function
	// that checks if stork agent cert is still valid.
	options := &advancedtls.ClientOptions{
		// set root CA cert to validate agent's certificate
		RootOptions: advancedtls.RootCertificateOptions{
			RootCACerts: certPool,
		},
		// set TLS client (stork server) cert to present its identity to server (stork agent)
		IdentityOptions: advancedtls.IdentityCertificateOptions{
			Certificates: []tls.Certificate{certificate},
		},
		// check cert and if it matches host IP
		VType: advancedtls.CertAndHostVerification,
		// additional verification hook function that checks if stork agent is not using old cert
		// VerifyPeer: func(params *advancedtls.VerificationFuncParams) (*advancedtls.VerificationResults, error) {
		// 	// TODO: add here check if agent cert is not old (compare it with
		// 	// CertFingerprint from Machine db record)
		// 	return &advancedtls.VerificationResults{}, nil
		// },
		// Only Stork server is allowed to connect to Stork agent over GRPC
		// and it always uses TLS 1.3.
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
		VerifyPeer: verifyPeer,
	}
	creds, err := advancedtls.NewClientCreds(options)
	if err != nil {
		log.Fatalf("advancedtls.NewClientCreds(%v) failed: %v", options, err)
	}

	return creds, nil
}

// Prepare gRPC connection to agent.
func (agent *Agent) MakeGrpcConnection(caCertPEM, serverCertPEM, serverKeyPEM []byte) error {
	// If there is any old connection then clean it up
	if agent.GrpcConn != nil {
		agent.GrpcConn.Close()
	}

	// Prepare TLS credentials
	creds, err := prepareTLSCreds(caCertPEM, serverCertPEM, serverKeyPEM)
	if err != nil {
		return errors.WithMessagef(err, "problem preparing TLS credentials")
	}

	// Setup new connection
	grpcConn, err := grpc.NewClient(
		agent.Address,
		grpc.WithTransportCredentials(creds),
	)
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
	GetConnectedAgentStatsWrapper(address string, port int64) *AgentCommStatsWrapper
	Ping(ctx context.Context, machine dbmodel.MachineTag) error
	GetState(ctx context.Context, machine dbmodel.MachineTag) (*State, error)
	ForwardRndcCommand(ctx context.Context, app ControlledApp, command string) (*RndcOutput, error)
	ForwardToNamedStats(ctx context.Context, app ControlledApp, statsAddress string, statsPort int64, path string, statsOutput interface{}) error
	ForwardToKeaOverHTTP(ctx context.Context, app ControlledApp, commands []keactrl.SerializableCommand, cmdResponses ...interface{}) (*KeaCmdsResult, error)
	TailTextFile(ctx context.Context, machine dbmodel.MachineTag, path string, offset int64) ([]string, error)
}

// Agents management map. It tracks Agents currently connected to the Server.
type connectedAgentsData struct {
	Settings      *AgentsSettings
	EventCenter   eventcenter.EventCenter
	AgentsMap     map[string]*Agent
	CommLoopReqs  chan *commLoopReq
	DoneCommLoop  chan bool
	Wg            *sync.WaitGroup
	serverCertPEM []byte
	serverKeyPEM  []byte
	caCertPEM     []byte
	mutex         sync.RWMutex
}

// Create new ConnectedAgents objects.
func NewConnectedAgents(settings *AgentsSettings, eventCenter eventcenter.EventCenter, caCertPEM, serverCertPEM, serverKeyPEM []byte) *connectedAgentsData {
	agents := connectedAgentsData{
		Settings:      settings,
		EventCenter:   eventCenter,
		AgentsMap:     make(map[string]*Agent),
		CommLoopReqs:  make(chan *commLoopReq),
		DoneCommLoop:  make(chan bool),
		Wg:            &sync.WaitGroup{},
		caCertPEM:     caCertPEM,
		serverCertPEM: serverCertPEM,
		serverKeyPEM:  serverKeyPEM,
		mutex:         sync.RWMutex{},
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
		return agent, nil
	}
	// Agent not found so allocate agent and prepare connection
	agent = &Agent{
		Address: address,
		Stats:   NewAgentStats(),
	}
	// Avoid a race with GetConnectedAgentStats().
	agents.mutex.Lock()
	agents.AgentsMap[address] = agent
	agents.mutex.Unlock()

	err := agent.MakeGrpcConnection(agents.caCertPEM, agents.serverCertPEM, agents.serverKeyPEM)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"address": address,
	}).Info("Connecting to new agent")

	return agent, nil
}

// Returns statistics for the connected agent. The statistics include number
// of errors to communicate with the agent and the number of errors to
// communicate with the apps behind the agent.
func (agents *connectedAgentsData) getConnectedAgentStats(address string, port int64) *AgentCommStats {
	if port != 0 {
		address = net.JoinHostPort(address, strconv.FormatInt(port, 10))
	}
	// Avoid a race with GetConnectedAgent().
	agents.mutex.RLock()
	defer agents.mutex.RUnlock()
	if agent, ok := agents.AgentsMap[address]; ok {
		return agent.Stats
	}
	return nil
}

// Returns a wrapper for statistics. The wrapper can be used to safely
// access the returned statistics for reading in other packages.
func (agents *connectedAgentsData) GetConnectedAgentStatsWrapper(address string, port int64) *AgentCommStatsWrapper {
	stats := agents.getConnectedAgentStats(address, port)
	if stats == nil {
		return nil
	}
	return NewAgentCommStatsWrapper(stats)
}
