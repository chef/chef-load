package main

import (
	"fmt"
	"os"

	"github.com/naoina/toml"
)

type chefLoadConfig struct {
	Mode                          string
	DataCollectorURL              string `toml:"data_collector_url"`
	DataCollectorToken            string
	EnableChefClientDataCollector bool
	SleepDuration                 int
	ChefServerURL                 string `toml:"chef_server_url"`
	ClientName                    string
	ClientKey                     string
	BootstrapNodesConcurrency     int
	Nodes                         int
	NodeNamePrefix                string
	OhaiJSONFile                  string `toml:"ohai_json_file"`
	Interval                      int
	Splay                         int
	Runs                          int
	ChefEnvironment               string
	RunList                       []string
	DownloadCookbooks             string
	APIGetRequests                []string `toml:"api_get_requests"`
	EnableReporting               bool
}

func printSampleConfig() {
	sampleConfig := `# Select the mode chef-load should operate in.
#
# Available modes are:
#
# chef-client - simulate a chef-client run's API requests optionally including Chef Reporting
#               and Automate's Visibility API requests
#
# data-collector - simulate only the API requests that a chef-client sends to
#                  an Automate server's data-collector endpoint. The benefit of
#                  this mode is it applies load to an Automate server without requiring
#                  a Chef Server.
mode = "chef-client"

# The URL to the Chef Automate Visibility Data Collector URL
# data_collector_url = "https://automate.example.org/data-collector/v0/"
#
# The Authorization token for Chef Automate Visibility
# data_collector_token = "93a49a4f2482c64126f7b6015e6b0f30284287ee4054ff8807fb63d9cbd1c506"
#
# Send data to the Chef Automate Visibility Data Collector when the mode is "chef-client"
# enable_chef_client_data_collector = false

# When the mode is "chef-client" the sleep_duration happens between the chef-client
# getting its cookbooks and it making the final API requests to report it has finished its run.
# When the mode is "data-collector" the sleep_duration happens between the data-collector's run_start
# and its run_converge messages.
# In both cases the intent is to enable a more accurate simulation of API requests.
# sleep_duration is measured in seconds
# sleep_duration = 0

# The URL of the Chef Server including the organization name
chef_server_url = "https://chef.example.com/organizations/demo/"

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
# client_name = "CLIENT_NAME"
# client_key = "/path/to/CLIENT_NAME.pem"

# chef-load only uses the API client defined above when bootstrapping all nodes.
# The bootstrap process makes the following API requests for each node.
# 1. Delete the node
# 2. Delete the node's client
# 3. Create a new client for the node
#
# It is important for the bootstrap process to complete successfully for each node otherwise there
# is no chance for the node's chef-client run to make successful API requests.
# To ensure success the first thing chef-load does is bootstrap all nodes.
#
# bootstrap_nodes_concurrency = 20

# Number of nodes making chef-client runs
# nodes = 10

# This prefix will go at the beginning of each node name.
# This enables running multiple instances of the chef-load tool without affecting each others' nodes
# For example, a value of "chef-load" will result in nodes named "chef-load-0", "chef-load-1", ...
# node_name_prefix = "chef-load"

# Ohai data will be loaded from this file and used for the nodes' automatic attributes.
# Leave this unset to leave automatic attributes empty.
# An ohai JSON file can be created by running "ohai > ohai.json".
# ohai_json_file = "/path/to/ohai.json"

# interval = 1800     # Interval between a node's chef-client runs, in seconds
# splay = 300         # A random number between zero and splay that is added to interval, in seconds

# runs = 0            # Number of chef-client runs each node should make, 0 value will make infinite runs

# chef_environment = "_default"     # Chef environment used by each node

# run_list is the run list used by each node. It should be a list of strings.
# For example: run_list = [ "role[role_name]", "recipe_name", "recipe[different_recipe_name@1.0.0]" ]
# The default value is an empty run_list.
# run_list = [ ]

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

# Send data to the Chef server's Reporting service
# enable_reporting = false
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
		Mode: "chef-client",

		DataCollectorURL:              "http://automate.example.org/data-collector/v0",
		DataCollectorToken:            "93a49a4f2482c64126f7b6015e6b0f30284287ee4054ff8807fb63d9cbd1c506",
		EnableChefClientDataCollector: false,

		SleepDuration: 0,

		ChefServerURL: "https://chef.example.com/organizations/demo/",

		BootstrapNodesConcurrency: 20,

		Nodes:          10,
		NodeNamePrefix: "chef-load",

		OhaiJSONFile: "",

		Interval: 1800,
		Splay:    300,

		Runs: 0,

		ChefEnvironment:   "_default",
		RunList:           make([]string, 0),
		DownloadCookbooks: "never",
		EnableReporting:   false,
	}

	if err = toml.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
