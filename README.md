## Description

chef-load is a tool written in Go that simulates load on a [Chef Server](https://www.chef.io/chef/) and/or [Chef Automate](https://www.chef.io/chef://www.chef.io/automate/) from a configured number of nodes.

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

Generate a chef-load configuration file.  
The configuration file uses [TOML syntax](https://github.com/toml-lang/toml) and documents a lot of the flexibility of chef-load so please read it.

```
chef-load --sample-config > chef-load.conf
```

Make sure chef-load.conf is configured properly.

Select the "mode" you want chef-load to operate in.

You will need to make sure appropriate settings are made depending on the chosen mode. For example, "chef-client" mode
will require at least "chef_server_url", "client_name" and "client_key" to have working values in chef-load.conf. However, the "data-collector" mode will require at least the "data_collector_url" and "data_collector_token" to have working values.

Run chef-load using only the configuration file.

```
chef-load --config chef-load.conf
```

You can set the desired Chef Client runs per minute rate using the `--rpm` command line option. This is useful for quickly adjusting the rate without modifying the configuration file.

```
chef-load --config chef-load.conf --rpm 30
```

You can also set the desired interval between each node's Chef Client run using the `--interval` option. The default is 30 minutes.

```
chef-load --config chef-load.conf --rpm 30 --interval 1
```
