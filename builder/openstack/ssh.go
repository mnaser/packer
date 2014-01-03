package openstack

import (
	gossh "code.google.com/p/go.crypto/ssh"
	"errors"
	"fmt"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/communicator/ssh"
	"github.com/rackspace/gophercloud"
	"time"
)

// SSHAddress returns a function that can be given to the SSH communicator
// for determining the SSH address based on the server AccessIPv4 setting..
func SSHAddress(csp gophercloud.CloudServersProvider, port int) func(multistep.StateBag) (string, error) {
	return func(state multistep.StateBag) (string, error) {

		// #TODO(benbp): this should be replaced with user specified
		// values in the packer builder template
		pools := []string{"public", "private", "nebula"}
		// #TODO(benbp): This for loop makes an assumption about the number of
		// possible addresses there will be. There needs to be a more flexible
		// conditional that loops through all of N available addresses.
		for j := 0; j < 2; j++ {
			s := state.Get("server").(*gophercloud.Server)
			for i := 0; i < len(pools); i++ {
				if val, ok := s.Addresses[pools[i]]; ok {
					addr := val.([]interface{})[j].(map[string]interface{})["addr"]
					if addr != "" {
						return fmt.Sprintf("%s:%d", addr, port), nil
					}
				}
			}
			serverState, err := csp.ServerById(s.Id)

			if err != nil {
				return "", err
			}

			state.Put("server", serverState)
			time.Sleep(1 * time.Second)
		}

		return "", errors.New("couldn't determine IP address for server")
	}
}

// SSHConfig returns a function that can be used for the SSH communicator
// config for connecting to the instance created over SSH using the generated
// private key.
func SSHConfig(username string) func(multistep.StateBag) (*gossh.ClientConfig, error) {
	return func(state multistep.StateBag) (*gossh.ClientConfig, error) {
		privateKey := state.Get("privateKey").(string)

		keyring := new(ssh.SimpleKeychain)
		if err := keyring.AddPEMKey(privateKey); err != nil {
			return nil, fmt.Errorf("Error setting up SSH config: %s", err)
		}

		return &gossh.ClientConfig{
			User: username,
			Auth: []gossh.ClientAuth{
				gossh.ClientAuthKeyring(keyring),
			},
		}, nil
	}
}
