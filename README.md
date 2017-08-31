## Description

chef-load is a tool that simulates Chef Client API load on a [Chef Server](https://www.chef.io/chef/) and/or a [Chef Automate server](https://www.chef.io/chef://www.chef.io/automate/).

It is designed to be easy to use yet powerfully flexible and accurate in its simulation of the chef-client run.

It works well at this point but there is always room for improvement so please provide feedback.

## Considerations when applying high load

chef-load will periodically resolve hostnames for the API requests. It is recommended that the
hostnames and their IP addresses are put in the system's `/etc/hosts` file so chef-load does not
need to make DNS requests.

Make sure the maximum number of open file descriptors is set to `unlimited`. This can either be done in chef-load's systemd service file as shown below or you can follow instructions in [this link](https://www.cyberciti.biz/faq/linux-increase-the-maximum-number-of-open-files/).

## Installation & Setup

### Download prebuilt binary

Prebuilt chef-load binary files are available on chef-load's "Releases" page.

https://github.com/jeremiahsnapp/chef-load/releases

### Generate a chef-load configuration file.  

The configuration file uses [TOML syntax](https://github.com/toml-lang/toml) and documents a lot of the flexibility of chef-load so please read it.

```
chef-load --sample-config > chef-load.conf
```

chef-load logs all API requests in the file specified by the `log_file` setting in the config file. The default value is `/var/log/chef-load/chef-load.log`.

Make sure chef-load.conf has appropriate settings for applying load to your Chef Server,
Automate Server or both.

#### Chef API client

The client defined by "client_name" in the chef-load configuration file needs to be an admin user of the Chef Server organization.

## Various ways to run chef-load

Run chef-load using only the configuration file.

```
chef-load --config chef-load.conf
```

You can use the `--prefix` command line option to set the prefix for the node names. This
enables easily running multiple instances of chef-load without affecting each others' nodes.
For example, a value of "chef-load" will result in nodes named "chef-load-1", "chef-load-2", etc.

```
chef-load --config chef-load.conf --prefix chef-load-a

# in another terminal you can run the following to create another instance of chef-load
chef-load --config chef-load.conf --prefix chef-load-b
```

You can set the number of nodes using the `--nodes` command line option and the interval using the `--interval` command line option. The default value for both of these options is 30. This is useful for quickly adjusting the load without modifying the configuration file.

chef-load will evenly distribute the number of nodes across the desired interval (minutes).

Examples:

* 1800 nodes / 30 minute interval = 60 chef-client runs per minute
* 1800 nodes / 60 minute interval = 30 chef-client runs per minute

```
chef-load --config chef-load.conf --nodes 1800
```

```
chef-load --config chef-load.conf --nodes 1800 --interval 60
```

### Example chef-load systemd service file

Here is a working example of a systemd service file for chef-load. Notice that it is able to set `LimitNOFILE` to unlimited to avoid running out of file descriptors.

```
[Unit]
Description=Chef load testing tool
After=network.target

[Service]
ExecStart=/home/centos/chef-load -config /home/centos/chef_load.conf
Type=simple
PIDFile=/tmp/chef_load.pid
Restart=always
ExecReload=/bin/kill -HUP $MAINPID
KillMode=process
Restart=on-failure
LimitNOFILE=infinity

[Install]
WantedBy=default.target
```

## API Request Profile

chef-load prints an API request profile when it receives a `USR1` signal and when it is terminated.

```
root@ip-172-31-17-147:~# ./chef-load -config chef-load.conf -nodes 60 -interval 1 -prefix foo
2017-08-30T20:25:37Z Starting chef-load with 60 nodes distributed evenly across a 1 minute interval
2017-08-30T20:25:37Z All API requests will be logged in /var/log/chef-load/chef-load.log
2017-08-30T20:25:44Z Received Signal: USR1
2017-08-30T20:25:44Z Printing profile of API requests
Total API Requests: 616
% of Total | Subtotal | Status | Method | URL
2.27         14         204      POST     https://automate.lxc/data-collector/v0/
1.14         7          409      POST     https://chef.lxc/organizations/demo/clients
1.14         7          200      GET      https://chef.lxc/organizations/demo/environments/_default
1.14         7          200      POST     https://chef.lxc/organizations/demo/environments/_default/cookbook_versions
1.14         7          200      GET      https://chef.lxc/organizations/demo/nodes/foo-<N>
1.14         7          200      PUT      https://chef.lxc/organizations/demo/nodes/foo-<N>
1.14         7          404      GET      https://chef.lxc/organizations/demo/roles/chef-client
90.91        560        200      GET      https://chef.lxc:443/bookshelf/<...>
^C
2017-08-30T20:25:50Z Received Signal: INT
2017-08-30T20:25:50Z Printing profile of API requests
Total API Requests: 1149
% of Total | Subtotal | Status | Method | URL
2.35         27         204      POST     https://automate.lxc/data-collector/v0/
1.22         14         409      POST     https://chef.lxc/organizations/demo/clients
1.22         14         200      GET      https://chef.lxc/organizations/demo/environments/_default
1.13         13         200      POST     https://chef.lxc/organizations/demo/environments/_default/cookbook_versions
1.22         14         200      GET      https://chef.lxc/organizations/demo/nodes/foo-<N>
1.13         13         200      PUT      https://chef.lxc/organizations/demo/nodes/foo-<N>
1.22         14         404      GET      https://chef.lxc/organizations/demo/roles/chef-client
90.51        1040       200      GET      https://chef.lxc:443/bookshelf/<...>
2017-08-30T20:25:50Z Stopping chef-load
```

The `-profile-logs` option will read the specified chef-load log files and print an API request profile. If chef-load receives a `USR1` signal or is terminated before it finishes reading the log files then it will print an API request profile for the data that it read up to that point in time.

```
root@ip-172-31-17-147:~# ./chef-load -profile-logs  /var/log/chef-load/*
2017-08-31T15:04:42Z Reading log file /var/log/chef-load/chef-load.log
2017-08-31T15:04:43Z Reading log file /var/log/chef-load/chef-load.log.1
2017-08-31T15:04:43Z Reading log file /var/log/chef-load/chef-load.log.2
2017-08-31T15:04:43Z Reading log file /var/log/chef-load/chef-load.log.3
2017-08-31T15:04:43Z Received Signal: USR1
2017-08-31T15:04:43Z Printing profile of API requests
Total API Requests: 31478
% of Total | Subtotal | Status | Method | URL
2.55         804        204      POST     https://automate.lxc/data-collector/v0/
0.04         12         201      POST     https://chef.lxc/organizations/demo/clients
1.05         330        409      POST     https://chef.lxc/organizations/demo/clients
1.16         366        200      GET      https://chef.lxc/organizations/demo/environments/_default
1.16         365        200      POST     https://chef.lxc/organizations/demo/environments/_default/cookbook_versions
0.04         12         201      POST     https://chef.lxc/organizations/demo/nodes
1.12         354        200      GET      https://chef.lxc/organizations/demo/nodes/bar-12-<N>
0.04         12         404      GET      https://chef.lxc/organizations/demo/nodes/bar-12-<N>
1.11         348        200      PUT      https://chef.lxc/organizations/demo/nodes/bar-12-<N>
1.16         366        404      GET      https://chef.lxc/organizations/demo/roles/chef-client
90.57        28509      200      GET      https://chef.lxc:443/bookshelf/<...>
2017-08-31T15:04:43Z Reading log file /var/log/chef-load/chef-load.log.4
2017-08-31T15:04:44Z Reading log file /var/log/chef-load/chef-load.log.5
2017-08-31T15:04:44Z Reading log file /var/log/chef-load/chef-load.log.6
2017-08-31T15:04:44Z Printing profile of API requests
Total API Requests: 59128
% of Total | Subtotal | Status | Method | URL
2.56         1516       204      POST     https://automate.lxc/data-collector/v0/
0.03         20         201      POST     https://chef.lxc/organizations/demo/clients
1.08         641        409      POST     https://chef.lxc/organizations/demo/clients
1.16         685        200      GET      https://chef.lxc/organizations/demo/environments/_default
1.16         684        200      POST     https://chef.lxc/organizations/demo/environments/_default/cookbook_versions
0.03         20         201      POST     https://chef.lxc/organizations/demo/nodes
1.12         665        200      GET      https://chef.lxc/organizations/demo/nodes/bar-12-<N>
0.03         20         404      GET      https://chef.lxc/organizations/demo/nodes/bar-12-<N>
1.10         651        200      PUT      https://chef.lxc/organizations/demo/nodes/bar-12-<N>
1.16         685        404      GET      https://chef.lxc/organizations/demo/roles/chef-client
90.55        53541      200      GET      https://chef.lxc:443/bookshelf/<...>
```

## Using sample JSON data files

chef-load is able to use files containing ohai, converge status and compliance status data captured from real nodes. This helps by simulating more accurate API payloads.

chef-load's configuration file has the following settings to specify the path to sample data files.

* ohai_json_file
* converge_status_json_file
* compliance_status_json_file

The chef-load GitHub repo's ["sample-data" directory](https://github.com/jeremiahsnapp/chef-load/tree/master/sample-data) has a file for each type of data that can be used.

#### Create your own sample ohai JSON file

An ohai JSON file can be created by running "ohai > example-ohai.json".

#### Create sample compliance status JSON file

A compliance status JSON file can be created by adding the "json-file" reporter to a node's "audit" cookbook's attribute. The next Chef Client run will save the compliance status report in the "/var/chef/cache/cookbooks/audit/" directory.

Alternatively, you can simply run the "inspec exec" command along with the "--format json" option to execute inspec profiles against a node. The resultant JSON file can be used with the compliance_status_json_file chef-load option.

#### Create sample converge status JSON file

You can copy the following code to a file named capture-converge-status.rb in any cookbook's "libraries" directory and it will cause the next Chef Client run to save the converge status JSON data to "/tmp/converge-status.json" on the node. Once you capture the data you can remove the code from the cookbook.

```
class Chef
  class DataCollector
    class Reporter < EventDispatch::Base
      private
      def send_to_data_collector(message)
        return unless data_collector_accessible?
        IO.write('/tmp/converge-status.json', JSON.generate(message))
        http.post(nil, message, headers)
      end
    end
  end
end
```

## Build chef-load from source

To build chef-load you must have Go installed.  
Ref: https://golang.org/dl/

Setup a GOPATH if you haven't already.  
Ref: https://golang.org/doc/install#testing

```
mkdir ~/go-work
export GOPATH=~/go-work
```

Get and install chef-load

```
go get github.com/jeremiahsnapp/chef-load
```

It is easy to cross-compile chef-load for other platforms.  
Options for $GOOS and $GOARCH are listed in the following link.  
Ref: https://golang.org/doc/install/source#environment

The following command will create a chef-load executable file for linux amd64 in the current working directory.

```
env GOOS=linux GOARCH=amd64 go build github.com/jeremiahsnapp/chef-load
```

# License

chef-load - a tool that simulates Chef Client API load on a Chef Server and/or a Chef Automate server

|                      |                                          |
|:---------------------|:-----------------------------------------|
| **Author:**          | Jeremiah Snapp (<jeremiah@chef.io>)
| **Copyright:**       | Copyright 2016-2017, Chef Software, Inc.
| **License:**         | Apache License, Version 2.0

```
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```
