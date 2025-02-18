package agentcomm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"iter"
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
	"isc.org/stork/appdata/bind9stats"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/eventcenter"
)

var _ agentConnector = (*agentConnectorImpl)(nil)

// Settings specific to communication with Agents.
type AgentsSettings struct{}

// Interface for interacting with Agents via gRPC.
type ConnectedAgents interface {
	Shutdown()
	GetConnectedAgentStatsWrapper(address string, port int64) *AgentCommStatsWrapper
	Ping(ctx context.Context, machine dbmodel.MachineTag) error
	GetState(ctx context.Context, machine dbmodel.MachineTag) (*State, error)
	ForwardRndcCommand(ctx context.Context, app ControlledApp, command string) (*RndcOutput, error)
	ForwardToNamedStats(ctx context.Context, app ControlledApp, statsAddress string, statsPort int64, path string, statsOutput interface{}) error
	ForwardToKeaOverHTTP(ctx context.Context, app ControlledApp, commands []keactrl.SerializableCommand, cmdResponses ...interface{}) (*KeaCmdsResult, error)
	TailTextFile(ctx context.Context, machine dbmodel.MachineTag, path string, offset int64) ([]string, error)
	ReceiveZones(ctx context.Context, app ControlledApp, filter *bind9stats.ZoneFilter) iter.Seq2[*bind9stats.ExtendedZone, error]
}

// Interface representing a connector to a selected agent over gRPC.
// The agentConnectorImpl is a default implementation for this interface
// used by the connectedAgentsImpl. The connector can be replaced
// with a mock in the unit tests to eliminate actual gRPC communication.
type agentConnector interface {
	// Attempts to establish connection with the agent. The connection
	// should be cached by the connector and reused when possible.
	// The implementation should close any existing connection before
	// establishing a new one.
	connect() error
	// Closes established connection with the agent. It should be called
	// on agent shutdown.
	close()
	// Creates and returns a gRPC client using the established connection.
	// The client typically is not cached for future use because it is
	// created using a lightweight call with an already established connection.
	// Several clients can use the same underlying connection.
	createClient() agentapi.AgentClient
}

// Default implementation of the connector.
type agentConnectorImpl struct {
	agentAddress  string
	serverCertPEM []byte
	serverKeyPEM  []byte
	caCertPEM     []byte
	mutex         sync.Mutex
	conn          *grpc.ClientConn
}

// Instantiates connector implementation.
func newAgentConnectorImpl(agentAddress string, serverCertPEM, serverKeyPEM, caCertPEM []byte) agentConnector {
	return &agentConnectorImpl{
		agentAddress:  agentAddress,
		serverCertPEM: serverCertPEM,
		serverKeyPEM:  serverKeyPEM,
		caCertPEM:     caCertPEM,
		mutex:         sync.Mutex{},
	}
}

// Connects or re-connects using specified agent address, certs and keys.
// It stores the established connection. It closes any existing connection.
func (impl *agentConnectorImpl) connect() error {
	impl.mutex.Lock()
	defer impl.mutex.Unlock()

	impl.closeUnsafe()

	// Prepare TLS credentials.
	creds, err := prepareTLSCreds(impl.caCertPEM, impl.serverCertPEM, impl.serverKeyPEM)
	if err != nil {
		return errors.WithMessage(err, "problem preparing TLS credentials")
	}

	// Setup new connection.
	conn, err := grpc.NewClient(
		impl.agentAddress,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return errors.Wrapf(err, "problem to dial to agent %s", impl.agentAddress)
	}
	impl.conn = conn
	return nil
}

// Closes an existing connection if it exists.
func (impl *agentConnectorImpl) close() {
	impl.mutex.Lock()
	defer impl.mutex.Unlock()
	impl.closeUnsafe()
}

// Closes an existing connection if it exists (non-safe for concurrent use).
func (impl *agentConnectorImpl) closeUnsafe() {
	if impl.conn != nil {
		impl.conn.Close()
		impl.conn = nil
	}
}

// Instantiates gRPC client using established connection.
func (impl *agentConnectorImpl) createClient() agentapi.AgentClient {
	return agentapi.NewAgentClient(impl.conn)
}

// Runtime information about the connected agent.
type agentState struct {
	address   string
	connector agentConnector
	stats     *AgentCommStats
}

// Agents management map. It tracks Agents currently connected to the Server.
type connectedAgentsImpl struct {
	settings           *AgentsSettings
	eventCenter        eventcenter.EventCenter
	agentsStates       map[string]*agentState
	commLoopReqs       chan *commLoopReq
	doneCommLoop       chan bool
	connectorFactoryFn func(string) agentConnector
	wg                 *sync.WaitGroup
	mutex              sync.RWMutex
}

// Returns an exported interface of ConnectedAgents with the underlying
// connectedAgentsImpl instance. This interface is to be used from other
// packages in the Stork server that need to communicate with the agents.
// This interface hides implementation details from external packages.
func NewConnectedAgents(settings *AgentsSettings, eventCenter eventcenter.EventCenter, caCertPEM, serverCertPEM, serverKeyPEM []byte) ConnectedAgents {
	return newConnectedAgentsImpl(settings, eventCenter, caCertPEM, serverCertPEM, serverKeyPEM)
}

// Instantiates connectedAgentsImpl. It is used by the NewConnectedAgents
// function and by the unit tests of the agentcomm package.
func newConnectedAgentsImpl(settings *AgentsSettings, eventCenter eventcenter.EventCenter, caCertPEM, serverCertPEM, serverKeyPEM []byte) *connectedAgentsImpl {
	agents := connectedAgentsImpl{
		settings:     settings,
		eventCenter:  eventCenter,
		agentsStates: make(map[string]*agentState),
		commLoopReqs: make(chan *commLoopReq),
		doneCommLoop: make(chan bool),
		connectorFactoryFn: func(agentAddress string) agentConnector {
			return newAgentConnectorImpl(agentAddress, serverCertPEM, serverKeyPEM, caCertPEM)
		},
		wg:    &sync.WaitGroup{},
		mutex: sync.RWMutex{},
	}

	agents.wg.Add(1)
	go agents.communicationLoop()

	return &agents
}

// Replaces the default connector factory with a custom one. It can be
// used to mock gRPC calls.
func (agents *connectedAgentsImpl) setConnectorFactory(factory func(string) agentConnector) {
	agents.connectorFactoryFn = factory
}

// Stops communication with all agents.
func (agents *connectedAgentsImpl) Shutdown() {
	log.Printf("Stopping communication with agents")
	for _, agent := range agents.agentsStates {
		agent.connector.close()
	}

	close(agents.commLoopReqs)
	agents.doneCommLoop <- true
	agents.wg.Wait()
	log.Printf("Stopped communication with agents")
}

// Get Agent object by its address.
func (agents *connectedAgentsImpl) getConnectedAgent(address string) (*agentState, error) {
	// Look for agent in Agents map and if found then return it
	agent, ok := agents.agentsStates[address]
	if ok {
		return agent, nil
	}
	// Agent not found so allocate agent and prepare connection
	agent = &agentState{
		address:   address,
		stats:     NewAgentStats(),
		connector: agents.connectorFactoryFn(address),
	}
	// Avoid a race with GetConnectedAgentStats().
	agents.mutex.Lock()
	agents.agentsStates[address] = agent
	agents.mutex.Unlock()

	err := agent.connector.connect()
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
func (agents *connectedAgentsImpl) getConnectedAgentStats(address string, port int64) *AgentCommStats {
	if port != 0 {
		address = net.JoinHostPort(address, strconv.FormatInt(port, 10))
	}
	// Avoid a race with GetConnectedAgent().
	agents.mutex.RLock()
	defer agents.mutex.RUnlock()
	if agent, ok := agents.agentsStates[address]; ok {
		return agent.stats
	}
	return nil
}

// Returns a wrapper for statistics. The wrapper can be used to safely
// access the returned statistics for reading in other packages.
func (agents *connectedAgentsImpl) GetConnectedAgentStatsWrapper(address string, port int64) *AgentCommStatsWrapper {
	stats := agents.getConnectedAgentStats(address, port)
	if stats == nil {
		return nil
	}
	return NewAgentCommStatsWrapper(stats)
}

// The GRPC client callback to perform extra verification of the peer
// certificate.
// The callback is running at the end of server certificate verification.
func verifyPeer(params *advancedtls.HandshakeVerificationInfo) (*advancedtls.PostHandshakeVerificationResults, error) {
	// The peer must have the extended key usage set.
	if len(params.Leaf.ExtKeyUsage) == 0 {
		return nil, errors.New("peer certificate does not have the extended key usage set")
	}
	return &advancedtls.PostHandshakeVerificationResults{}, nil
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
	options := &advancedtls.Options{
		// set root CA cert to validate agent's certificate
		RootOptions: advancedtls.RootCertificateOptions{
			RootCertificates: certPool,
		},
		// set TLS client (stork server) cert to present its identity to server (stork agent)
		IdentityOptions: advancedtls.IdentityCertificateOptions{
			Certificates: []tls.Certificate{certificate},
		},
		// check cert and if it matches host IP
		VerificationType: advancedtls.CertAndHostVerification,
		// additional verification hook function that checks if stork agent is not using old cert
		// VerifyPeer: func(params *advancedtls.VerificationFuncParams) (*advancedtls.VerificationResults, error) {
		// 	// TODO: add here check if agent cert is not old (compare it with
		// 	// CertFingerprint from Machine db record)
		// 	return &advancedtls.VerificationResults{}, nil
		// },
		// Only Stork server is allowed to connect to Stork agent over GRPC
		// and it always uses TLS 1.3.
		MinTLSVersion:              tls.VersionTLS13,
		MaxTLSVersion:              tls.VersionTLS13,
		AdditionalPeerVerification: verifyPeer,
	}
	creds, err := advancedtls.NewClientCreds(options)
	if err != nil {
		log.Fatalf("advancedtls.NewClientCreds(%v) failed: %v", options, err)
	}

	return creds, nil
}
