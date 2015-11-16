package main

import (
	"fmt"
	"net"

	"github.com/starkandwayne/shield/agent"
	"github.com/voxelbrain/goptions"
)

type ShieldAgentOpts struct {
	AuthorizedKeysFile string `goptions:"-A, --authorized-keys, obligatory, description='Path to the authorized (public) keys file, for authenticating clients'"`
	HostKeyFile        string `goptions:"-k, --key, obligatory, description='Path to the server host key file'"`
	ListenAddress      string `goptions:"-l, --listen, obligatory, description='Network address and port to listen on'"`
}

func main() {
	fmt.Printf("starting up...\n")

	var opts ShieldAgentOpts
	if err := goptions.Parse(&opts); err != nil {
		fmt.Printf("%s\n", err)
		goptions.PrintHelp()
		return
	}

	authorizedKeys, err := agent.LoadAuthorizedKeys(opts.AuthorizedKeysFile)
	if err != nil {
		fmt.Printf("failed to load authorized keys: %s\n", err)
		return
	}

	config, err := agent.ConfigureSSHServer(opts.HostKeyFile, authorizedKeys)
	if err != nil {
		fmt.Printf("failed to configure SSH server: %s\n", err)
		return
	}

	listener, err := net.Listen("tcp", opts.ListenAddress)
	if err != nil {
		fmt.Printf("failed to bind %s: %s", opts.ListenAddress, err)
		return
	}

	fmt.Printf("listening on %s\n", opts.ListenAddress)

	agent := agent.NewAgent(config)
	agent.Serve(listener)
}
