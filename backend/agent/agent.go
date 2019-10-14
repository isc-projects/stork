package agent

import (
	"net"
	"fmt"
	"context"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"isc.org/stork/api"
)


// Stork Agent settings.
type AgentSettings struct {
	Host         string        `long:"host" description:"the IP to listen on" env:"STORK_AGENT_ADDRESS"`
	Port         int           `long:"port" description:"the port to listen on for connections" default:"8080" env:"STORK_AGENT_PORT"`
}

// Global Stork Agent state
type StorkAgent struct {
	Settings AgentSettings
}


// API exposed to Stork Server

// Get version of Agent
func (s *StorkAgent) GetVersion(ctx context.Context, in *agentapi.Empty) (*agentapi.AgentVersion, error) {
	log.Printf("Received: GetVersion %v", in)
	return &agentapi.AgentVersion{Version: "1.0.9a"}, nil
}

// Detect services (Kea, Bind)
func (s *StorkAgent) DetectServices(ctx context.Context, in *agentapi.DetectServicesReq) (*agentapi.DetectServicesRsp, error) {
	log.Printf("Received: DetectServices %v", in)
	return &agentapi.DetectServicesRsp{Abc: "321"}, nil
}

// Restart Kea service.
func (s *StorkAgent) RestartKea(ctx context.Context, in *agentapi.RestartKeaReq) (*agentapi.RestartKeaRsp, error) {
	log.Printf("Received: RestartKea %v", in)
	return &agentapi.RestartKeaRsp{Xyz: "321"}, nil
}

func (sa *StorkAgent) Serve() {
	// Install gRPC API handlers.
	server := grpc.NewServer()
	agentapi.RegisterAgentServer(server, sa)

	// Prepare listener on configured address.
	addr := fmt.Sprintf("%s:%d", sa.Settings.Host, sa.Settings.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on port: %+v", err)
	}

	// Start serving gRPC
	log.WithFields(log.Fields{
		"address": lis.Addr(),
	}).Infof("Started serving Stork Agent")
	if err := server.Serve(lis); err != nil {
		log.Fatalf("Failed to listen on port: %+v", err)
	}
}
