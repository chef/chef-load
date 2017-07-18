package main

import (
	"fmt"
	"os"

	"github.com/naoina/toml"
)

type chefLoadConfig struct {
	RunChefClient            bool
	ChefServerURL            string `toml:"chef_server_url"`
	ClientName               string
	ClientKey                string
	DataCollectorURL         string `toml:"data_collector_url"`
	DataCollectorToken       string
	OhaiJSONFile             string `toml:"ohai_json_file"`
	ConvergeStatusJSONFile   string `toml:"converge_status_json_file"`
	ComplianceStatusJSONFile string `toml:"compliance_status_json_file"`
	NumNodes                 int
	Interval                 int
	NodeNamePrefix           string
	ChefEnvironment          string
	RunList                  []string
	SleepDuration            int
	DownloadCookbooks        string
	APIGetRequests           []string `toml:"api_get_requests"`
	EnableReporting          bool
}

func printSampleConfig() {
	sampleConfig := `# The chef_server_url, client_name and client_key parameters must be set if you want
# to make API requests to a Chef Server.
#
# Be sure to include the organization name
# For example: chef_server_url = "https://chef.example.com/organizations/demo/"
# chef_server_url = ""
#
# The client defined by client_name needs to be an admin user of the Chef Server org.
# client_name = "CLIENT_NAME"
# client_key = "/path/to/CLIENT_NAME.pem"

# The data_collector_url must be set if you want to make API requests to an Automate server.
# For example: data_collector_url = "https://automate.example.org/data-collector/v0/"
# data_collector_url = ""

# The Authorization token for the Automate server.
# The following default value is sufficient unless you set your own token in your Automate server.
# data_collector_token = "93a49a4f2482c64126f7b6015e6b0f30284287ee4054ff8807fb63d9cbd1c506"

# Ohai data will be loaded from this file and used for the nodes' automatic attributes.
# See the chef-load README for instructions for creating an ohai JSON file.
# ohai_json_file = "/path/to/example-ohai.json"

# Data from a converge status report will be loaded from this file and used
# for each node's converge status report that is sent to the Automate server.
# See the chef-load README for instructions for creating a converge status JSON file.
# converge_status_json_file = "/path/to/example-converge-status.json"

# Data from a compliance status report will be loaded from this file and used
# for each node's compliance status report that is sent to the Automate server.
# See the chef-load README for instructions for creating a compliance status JSON file.
# compliance_status_json_file = "/path/to/example-compliance-status.json"

# chef-load will evenly distribute the number of nodes across the desired interval (minutes)
# Examples:
#   30 nodes / 30 minute interval =  1 chef-client run per minute
# 1800 nodes / 30 minute interval = 60 chef-client runs per minute
# num_nodes = 30
# interval = 30

# This prefix will go at the beginning of each node name.
# This enables running multiple instances of chef-load without affecting each others' nodes
# For example, a value of "chef-load" will result in nodes named "chef-load-1", "chef-load-2", ...
# node_name_prefix = "chef-load"

# Chef environment used for each node
# chef_environment = "_default"

# run_list is the run list used for each node. It should be a list of strings.
# For example: run_list = [ "role[role_name]", "recipe_name", "recipe[different_recipe_name@1.0.0]" ]
# The default value is an empty run_list.
# run_list = [ ]

# sleep_duration is an optional setting that is available to provide a delay to simulate
# the amount of time a Chef Client takes actually converging all of the run list's resources.
# sleep_duration is measured in seconds
# sleep_duration = 0

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
# ephemeral ports or reducing load by reducing the requests_per_minute setting.
# Ref: http://www.cyberciti.biz/tips/linux-increase-outgoing-network-sockets-range.html
#
# download_cookbooks = "never"

# api_get_requests is an optional list of API GET requests that are made during the chef-client run.
# This is used to simulate the API requests that the cookbooks would make.
# For example, it can make Chef Search or data bag item requests.
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
		RunChefClient: false,

		ChefServerURL: "",

		DataCollectorURL:   "",
		DataCollectorToken: "93a49a4f2482c64126f7b6015e6b0f30284287ee4054ff8807fb63d9cbd1c506",

		OhaiJSONFile: "",

		ConvergeStatusJSONFile: "",

		ComplianceStatusJSONFile: "",

		NumNodes: 30,

		Interval: 30,

		NodeNamePrefix: "chef-load",

		ChefEnvironment: "_default",
		RunList:         make([]string, 0),

		SleepDuration: 0,

		DownloadCookbooks: "never",
		EnableReporting:   false,
	}

	if err = toml.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
