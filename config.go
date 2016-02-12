package main

import (
	"fmt"
	"os"

	"github.com/naoina/toml"
)

type chefLoadConfig struct {
	ChefServerUrl     string
	ClientName        string
	ClientKey         string
	Nodes             int
	NodeNamePrefix    string
	NodeJsonFile      string
	Interval          int
	Splay             int
	Runs              int
	RunList           []string
	DownloadCookbooks string
	ApiGetRequests    []string
	SleepDuration     int
}

func printSampleConfig() {
	sampleConfig := `# The URL of the Chef Server including the organization name
chef_server_url = "https://HOSTNAME_or_IP/organizations/ORG_NAME/"

# Before a node's first chef-client run chef-load uses the API client defined by client_name
# and client_key to delete the node and its corresponding client.
#
# Then chef-load creates a new client for the node and puts the public key that corresponds with the
# private key defined by client_key into the new client.
# This avoids the need to manage individual private keys for each node.
#
# The new client is used for all remaining API requests.
#
# The client defined by client_name needs to be able to create clients. By default only admin users
# can create clients so the recommendation is to set client_name and client_key to an admin user
# of the org.
#
# If you want to use a regular user or even a regular client instead then you will need
# to use the knife-acl plugin to create a new group in the Chef Server, add the regular user or
# client to that group and then give the group the create permission on the clients container.
# Ref: https://github.com/chef/knife-acl
#
client_name = "CLIENT_NAME"
client_key = "/path/to/CLIENT_NAME.pem"

# Number of nodes making chef-client runs
# nodes = 10

# This prefix will go at the beginning of each node name.
# This enables running multiple instances of the chef-load tool without affecting each others' nodes
# For example, a value of "chef-load" will result in nodes named "chef-load-0", "chef-load-1", ...
# node_name_prefix = "chef-load"

# Node data will be loaded from this file. Set this to test uploading large amount of data.
# Leave this unset to use empty objects
# node_json_file = "/path/to/node.json"

# interval = 1800     # Interval between a node's chef-client runs, in seconds

# runs = 0            # Number of chef-client runs each node should make, 0 value will make infinite runs

# run_list = [ ]      # run_list used by each node, a list of strings

# download_cookbooks controls which chef-client run downloads cookbook files.
# Options are: "never", "first" (first chef-client run only), "always"
#
# Downloading cookbooks can significantly increase the number of API requests that chef-load
# makes depending on the run_list.
#
# Normal TCP protocol requires the connections to be in TIME-WAIT for about two minutes and it is
# recommended that the system's TIME-WAIT parameter's do not get changed.
# Ref: http://vincent.bernat.im/en/blog/2014-tcp-time-wait-state-linux.html
#
# If chef-load makes enough API requests then the number of connections can increase until
# the system runs out of ephemeral ports resulting in connect(2) error EADDRNOTAVAIL.
# Ref: http://manpages.ubuntu.com/manpages/trusty/en/man2/connect.2freebsd.html
# Ref: http://manpages.ubuntu.com/manpages/trusty/en/man7/ip.7.html
#
# If you aren't concerned with simulating the download of cookbook files then the recommendation is
# to use "never" or "first".
#
# If you want to use "always" and you run out of ephemeral ports then consider increasing the range of
# ephemeral ports or reducing load by changing chef-load settings such as "nodes" or "interval".
# Ref: http://www.cyberciti.biz/tips/linux-increase-outgoing-network-sockets-range.html
#
# download_cookbooks = "never"

# api_get_requests is an optional list of API GET requests that are made during the chef-client run.
# This is used to simulate the API requests that the cookbooks would make.
# For example, it can make Chef Search requests or requests to get data bag items.
# The values can be either full URLs that include the chef_server_url portion or just the portion of
# the URL that comes after the chef_server_url.
# For example, to make a Chef Search API request that searches for all nodes you can use either of the
# following values:
#
# "https://chef.example.com/organizations/orgname/search/node?q=*%253A*&sort=X_CHEF_id_CHEF_X%20asc&start=0"
# or
# "search/node?q=*%253A*&sort=X_CHEF_id_CHEF_X%20asc&start=0"
# 
# api_get_requests = [ ]

# sleep_duration happens between the chef-client getting its cookbooks and it making the final API requests
# to report it has finished its run. The intent is to enable a more accurate simulation of API requests.
# sleep_duration = 0
`
	fmt.Print(sampleConfig)
}

func loadConfig(file string) (*chefLoadConfig, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Initialize default configuration values
	config := chefLoadConfig{
		Nodes:          10,
		NodeNamePrefix: "chef-load",
		NodeJsonFile: "",

		Interval: 1800,
		Splay:    300,

		Runs: 0,

		RunList:           make([]string, 0),
		DownloadCookbooks: "never",
		SleepDuration:     0,
	}

	if err = toml.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
