package openstack

import (
	gossh "code.google.com/p/go.crypto/ssh"
	"errors"
	"fmt"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/communicator/ssh"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/servers"
)

// SSHAddress returns a function that can be given to the SSH communicator
// for determining the SSH address based on the server AccessIPv4 setting..
func SSHAddress(csp gophercloud.CloudServersProvider, port int,
	allocated_floating_ip gophercloud.FloatingIp, client *servers.Client) func(multistep.StateBag) (string, error) {

	return func(state multistep.StateBag) (string, error) {
		s := state.Get("server").(*gophercloud.Server)

		//If the FloatingIp field isn't empty we'll associate the ip with our server
		//And then return the address we're handed.
		if allocated_floating_ip.Ip != "" {

			err := csp.AssociateFloatingIp(s.Id, allocated_floating_ip)

			if err != nil {
				return "", errors.New("Error Associating the new Floating Ip with the server")
			}

			return fmt.Sprintf("%s:%d", allocated_floating_ip.Ip, port), nil

		}
		
		listResults, err := servers.List(client)
		if err != nil {
			return "",errors.New("Could not get server list")
		}

		servers, err := servers.GetServers(listResults)
		if err != nil {
			return "", errors.New(fmt.Sprintf("Could not parse server list: %s", err))
		}

		for _, server := range servers {
			if server.Id == s.Id {
				for _, slices := range server.Addresses {
					sliceVar := slices.([]interface{})
					for _, address := range sliceVar {
						target := address.(map[string]interface{})
						if target["addr"] != "" {
							return fmt.Sprintf("%s:%d", target["addr"].(string), port), nil
						}
					}
				}
			}
		}

		serverState, err := csp.ServerById(s.Id)

		if err != nil {
			return "", err
		}

		state.Put("server", serverState)

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
