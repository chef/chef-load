//
// Copyright:: Copyright 2017-2018 Chef Software, Inc.
// License:: Apache License, Version 2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package chef_load

import (
	"fmt"
)

type Platform struct {
	Name     string   `toml:"name"`
	Target   string   `toml:"target"`
	Profiles []string `toml:"profiles"`
}

type Samples struct {
	Platforms []Platform `mapstructure:"platforms"`
}

type Simulation struct {
	Days          int    `mapstructure:"days"`
	Nodes         int    `mapstructure:"nodes"`
	MaxScans      int    `mapstructure:"max_scans"`
	TotalMaxScans int    `mapstructure:"total_max_scans"`
	SampleFormat  string `mapstructure:"format"`
}

type Statistics struct {
	Sets []Set `mapstructure:"sets"`
}

type Set struct {
	Nodes      int `mapstructure:"nodes"`
	ScanPerDay int `mapstructure:"scan_per_day"`
}

type Matrix struct {
	Samples    Samples    `mapstructure:"samples"`
	Simulation Simulation `mapstructure:"simulation"`
	Statistics Statistics `mapstructure:"statistics"`
}

type Config struct {
	RunChefClient              bool
	LogFile                    string   `mapstructure:"log_file"`
	ChefServerURL              string   `mapstructure:"chef_server_url"`
	ClientName                 string   `mapstructure:"client_name"`
	ClientKey                  string   `mapstructure:"client_key"`
	DataCollectorURL           string   `mapstructure:"data_collector_url"`
	DataCollectorToken         string   `mapstructure:"data_collector_token"`
	OhaiJSONFile               string   `mapstructure:"ohai_json_file"`
	ConvergeStatusJSONFile     string   `mapstructure:"converge_status_json_file"`
	ComplianceStatusJSONFile   string   `mapstructure:"compliance_status_json_file"`
	ComplianceSampleReportsDir string   `mapstructure:"compliance_sample_reports_dir"`
	NumActions                 int      `mapstructure:"num_actions"`
	NumNodes                   int      `mapstructure:"num_nodes"`
	Interval                   int      `mapstructure:"interval"`
	NodeNamePrefix             string   `mapstructure:"node_name_prefix"`
	ChefEnvironment            string   `mapstructure:"chef_environment"`
	RunList                    []string `mapstructure:"run_list"`
	SleepDuration              int      `mapstructure:"sleep_duration"`
	DownloadCookbooks          string   `mapstructure:"download_cookbooks"`
	APIGetRequests             []string `mapstructure:"api_get_requests"`
	ChefVersion                string   `mapstructure:"chef_version"`
	ChefServerCreatesClientKey bool     `mapstructure:"chef_server_creates_client_key"`
	RandomData                 bool     `mapstructure:"random_data"`
	LivenessAgent              bool     `mapstructure:"liveness_agent"`
	EnableReporting            bool     `mapstructure:"enable_reporting"`
	DaysBack                   int      `mapstructure:"days_back"`
	Threads                    int      `mapstructure:"threads"`
	SleepTimeOnFailure         int      `mapstructure:"sleep_time_on_failure"`
	Matrix                     *Matrix  `mapstructure:"matrix"`
}

func Default() Config {
	return Config{
		RunChefClient:              false,
		LogFile:                    "/var/log/chef-load/chef-load.log",
		ChefServerURL:              "",
		DataCollectorURL:           "",
		DataCollectorToken:         "93a49a4f2482c64126f7b6015e6b0f30284287ee4054ff8807fb63d9cbd1c506",
		OhaiJSONFile:               "",
		ConvergeStatusJSONFile:     "",
		ComplianceStatusJSONFile:   "",
		ComplianceSampleReportsDir: "",
		NumNodes:                   30,
		Interval:                   30,
		NodeNamePrefix:             "chef-load",
		ChefEnvironment:            "_default",
		RunList:                    make([]string, 0),
		SleepDuration:              0,
		DownloadCookbooks:          "never",
		ChefVersion:                "13.2.20",
		ChefServerCreatesClientKey: false,
		EnableReporting:            false,
		RandomData:                 false,
		LivenessAgent:              false,
		NumActions:                 30,
		DaysBack:                   0,
		Threads:                    3000,
		SleepTimeOnFailure:         5,
		Matrix: &Matrix{
			Simulation: Simulation{
				Days:          1,
				Nodes:         0,
				MaxScans:      2,
				TotalMaxScans: 2,
				SampleFormat:  "full",
			},
			Samples: Samples{
				Platforms: []Platform{
					{Name: "c5-with-skip-message-depends", Profiles: []string{"mylinux-success-1.8.9","myrapper-child-0.6.2"}},
					{Name: "c5-with-skip-message", Profiles: []string{"mylinux-success-1.8.9","myprofile1-1.0.0"}},
					{Name: "c5", Profiles: []string{"mylinux-success-1.8.9"}},
					{Name: "c6", Profiles: []string{"cis-centos6-level1-1.1.0-1.4", "ssh-baseline-2.2.0"}},
					{Name: "c7", Profiles: []string{"mylinux-success-1.8.9"}},
					{Name: "d7", Profiles: []string{"apache-baseline-2.0.2"}},
					{Name: "d8", Profiles: []string{"mylinux-failure-minor-5.2.0"}},
					{Name: "d8-2", Profiles: []string{"mylinux-failure-major-5.4.4"}},
					{Name: "f22", Profiles: []string{"linux-baseline-2.2.0", "ssh-baseline-2.2.0",
						"apache-baseline-2.0.2", "mysql-baseline-2.1.0"}},
					{Name: "u12", Profiles: []string{"cis-ubuntu12_04lts-level1-1.1.0-2"}},
					{Name: "u14", Profiles: []string{"mylinux-success-1.8.9"}},
					{Name: "u18", Profiles: []string{"linux-baseline-2.2.0", "ssh-baseline1-2.2.0"}},
				},
			},
			Statistics: Statistics{
				Sets: []Set{
					{Nodes: 10, ScanPerDay: 1}, {Nodes: 10, ScanPerDay: 24},
					{Nodes: 100, ScanPerDay: 1}, {Nodes: 100, ScanPerDay: 24},
					{Nodes: 1000, ScanPerDay: 1}, {Nodes: 1000, ScanPerDay: 24},
					{Nodes: 10000, ScanPerDay: 1}, {Nodes: 10000, ScanPerDay: 24}, {Nodes: 10000, ScanPerDay: 96},
				}},
		},
	}
}

func PrintSampleConfig() {
	sampleConfig := `# log_file specifies the location to log API requests
# log_file = "/var/log/chef-load/chef-load.log"

# The chef_server_url, client_name and client_key parameters must be set if you want
# to make API requests to a Chef Server.
#
# chef-load will also automatically attempt to connect to the Chef Server authenticated data collector proxy.
# If you enabled this feature on the Chef Server, Chef Client run data will automatically be forwarded to Automate.
# If you do not have Automate or the feature is disabled on the Chef Server, chef-load will detect this and
# disable data collection.
#
# Be sure to include the organization name
# For example: chef_server_url = "https://chef.example.com/organizations/demo/"
# chef_server_url = ""
#
# The client defined by client_name needs to be an admin user of the Chef Server org.
# client_name = "CLIENT_NAME"
# client_key = "/path/to/CLIENT_NAME.pem"

# The data_collector_url must be set if you want to make API requests directly to an Automate server.
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

# Directory where compliance sample inspec reports live.  Compliance load requires these sample reports
# Use these as templates for creating the loaded reports for the generate action.
# See the chef-load README for instructions for obtaining the samples.
# compliance_sample_reports_dir = "/path/to/sample-data/inspec-reports"

# chef-load will evenly distribute the number of nodes across the desired interval (minutes)
# Examples:
#   30 nodes / 30 minute interval =  1 chef-client run per minute
# 1800 nodes / 30 minute interval = 60 chef-client runs per minute
# num_nodes = 30
# interval = 30

# During the same interval of time, it is also possible to load a number of Chef actions
# num_actions = 30

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

# days_back is an optional setting that allows the load of historical data. When provided, the tool
# will use this value to load the data from today to the provided day back.
# days_back = 30

# download_cookbooks controls which chef-client run downloads cookbook files.
# Options are: "never", "first" (first chef-client run only), "always"
#
# Downloading cookbooks can significantly increase the number of API requests that chef-load
# makes depending on the run_list. If you aren't concerned with simulating the download of cookbook files
# then the recommendation is to use "never" or "first".
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

# chef_version sets the value of the X-Chef-Version HTTP header in API requests sent to the Chef Server.
# This value represents the version of the Chef Client making the API requests. The default is "13.2.20"
# chef_version = "13.2.20"

# Ever since Chef Client 12.x was released the default behavior has been for the Chef Client to create its
# own client key locally and then upload the public side to the Chef Server when it creates the client object.
# chef-load simulates this behavior. However, if you want chef-load to ask the Chef Server to create a client key
# when the client object is created then set chef_server_creates_client_key to true.
# chef_server_creates_client_key = false

# Send data to the Chef server's Reporting service
# enable_reporting = false

# Generate Random Data
# random_data = true

# Generate Liveness Agent Data
# liveness_agent = true

# Matrix settings for Compliance Generation.  This is to ensure a diversity of nodes/scan/profiles
# for compliance data. This only applied when running in "this day back" or "generate" mode.
# In the future, it would be great if we could harmonize this with the converge nodes so that
# we will have consistancy between that which is ingested between the two
[matrix]
   #The samples listed in this section (matrix.samples), specify the profiles that each individual node will have included in their respective scans.
  [matrix.samples]
    [[matrix.samples.platforms]]
    name = "c5"
    target = "ssh://root@0.0.0.0:11032"
    profiles = [
      "mylinux-success-1.8.9"
    ]

    [[matrix.samples.platforms]]
    name = "c6"
    target = "ssh://root@0.0.0.0:11024"
    profiles = [
      "cis-centos6-level1-1.1.0-1.4",
      "ssh-baseline-2.2.0"
    ]

    [[matrix.samples.platforms]]
    name = "c7"
    target = "ssh://root@0.0.0.0:11025"
    profiles = [
      "mylinux-success-1.8.9"
    ]

    [[matrix.samples.platforms]]
    name = "d7"
    target = "ssh://root@0.0.0.0:11029"
    profiles = [
      "apache-baseline-1.0.2"
    ]

    [[matrix.samples.platforms]]
    name = "d8"
    target = "ssh://root@0.0.0.0:11028"
    profiles = [
      "mylinux-failure-minor-5.2.0"
    ]

    [[matrix.samples.platforms]]
    name = "d8-2"
    target = "ssh://root@0.0.0.0:11028"
    profiles = [
      "mylinux-failure-major-5.4.4"
    ]

    [[matrix.samples.platforms]]
    name = "f22"
    target = "ssh://root@0.0.0.0:11026"
    profiles = [
      "linux-baseline-2.2.0",
      "ssh-baseline-2.2.0",
      "apache-baseline-2.0.2",
      "mysql-baseline-2.1.0"
    ]

    [[matrix.samples.platforms]]
    name = "u12"
    target = "ssh://root@0.0.0.0:11022"
    profiles = [
      "cis-ubuntu12_04lts-level1-1.1.0-2"
    ]

    [[matrix.samples.platforms]]
    name = "u14"
    target = "ssh://root@0.0.0.0:11031"
    profiles = [
      "mylinux-success-1.8.9"
    ]

    [[matrix.samples.platforms]]
    name = "u18"
    target = "ssh://root@0.0.0.0:11033"
    profiles = [
      "linux-baseline-2.2.0",
      "ssh-baseline-2.2.0"
    ]

  [matrix.simulation]
  days = 10
  nodes = 50
  max_scans = 2
  total_max_scans = 1000000
  sample_format = "full"

  [matrix.statistics]

    [[matrix.statistics.sets]]
    nodes = 10
    scan_per_day = 1

    [[matrix.statistics.sets]]
    nodes = 10
    scan_per_day = 24

    [[matrix.statistics.sets]]
    nodes = 100
    scan_per_day = 1

    [[matrix.statistics.sets]]
    nodes = 100
    scan_per_day = 24

    [[matrix.statistics.sets]]
    nodes = 1000
    scan_per_day = 1

    [[matrix.statistics.sets]]
    nodes = 1000
    scan_per_day = 24

    [[matrix.statistics.sets]]
    nodes = 10000
    scan_per_day = 1

    [[matrix.statistics.sets]]
    nodes = 10000
    scan_per_day = 24

    [[matrix.statistics.sets]]
    nodes = 10000
    scan_per_day = 96
`
	fmt.Print(sampleConfig)
}
