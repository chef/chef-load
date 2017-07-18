## Description

chef-load is a tool that simulates Chef Client API load on a [Chef Server](https://www.chef.io/chef/) and/or a [Chef Automate server](https://www.chef.io/chef://www.chef.io/automate/).

It is designed to be easy to use yet powerfully flexible and accurate in its simulation of the chef-client run.

It works well at this point but there is always room for improvement so please provide feedback.

## Requirements

#### Download prebuilt binary

Prebuilt chef-load binary files are available on chef-load's "Releases" page.

https://github.com/jeremiahsnapp/chef-load/releases

#### OR build the chef-load executable

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

#### Chef API client

The client defined by "client_name" in the chef-load configuration file needs to be an admin user of the org.

## Upgrading chef-load

Sometimes a new version of chef-load has breaking changes in the configuration file. The easiest way to handle this might
be to create a new config file and copy/paste modified settings from the old file.

## Usage

Print help.

```
chef-load --help
```

### Generate a chef-load configuration file.  

The configuration file uses [TOML syntax](https://github.com/toml-lang/toml) and documents a lot of the flexibility of chef-load so please read it.

```
chef-load --sample-config > chef-load.conf
```

Make sure chef-load.conf has appropriate settings for applying load to your Chef Server,
Automate Server or both.

### Various ways to run chef-load

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

### Using sample JSON data files

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

### Run as a systemd service

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
LimitNOFILE=unlimited:unlimited

[Install]
WantedBy=default.target
```

### Considerations when applying high load

Make sure the system has `nscd` or something similar in place to cache DNS requests. This can significantly improve chef-load's performance when applying high load.

Make sure the maximum number of open file descriptors is set to `unlimited`. This can either be done in chef-load's systemd service file as shown above or you can follow instructions in [this link](https://www.cyberciti.biz/faq/linux-increase-the-maximum-number-of-open-files/).

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
