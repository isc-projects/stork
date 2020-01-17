package agent

import (
	"os/exec"
	"regexp"

	log "github.com/sirupsen/logrus"
)

type Bind9Daemon struct {
	Pid     int32
	Name    string
	Active  bool
	Version string
}

type Bind9State struct {
	Version string
	Active  bool
	Daemon  Bind9Daemon
}

func detectBind9App() *App {
	// TODO: address, control port

	bind9App := &App{
		Type: "bind9",
	}

	return bind9App
}

func getBind9State(app *App) (*Bind9State, error) { //nolint:unparam
	state := &Bind9State{
		Active: false,
	}

	// version
	cmd := exec.Command("rndc", "-k", "/etc/bind/rndc.key", "status")
	out, err := cmd.Output()
	if err != nil {
		log.Warnf("cannot get BIND 9 status: %+v", err)
	} else {
		versionPtrn := regexp.MustCompile(`version:\s(.+)\n`)
		match := versionPtrn.FindStringSubmatch(string(out))
		if match != nil {
			state.Version = match[1]
		} else {
			log.Warnf("cannot get BIND 9 version: unable to find version in rndc output")
		}

		state.Active = true
	}

	// TODO: pid

	namedDaemon := Bind9Daemon{
		Name:    "named",
		Active:  state.Active,
		Version: state.Version,
	}

	state.Daemon = namedDaemon

	return state, err
}
