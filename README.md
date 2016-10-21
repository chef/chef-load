## Description

chef-load is a tool written in Go that simulates load on a [Chef Server](https://www.chef.io/chef/) and/or [Chef Automate](https://www.chef.io/chef://www.chef.io/automate/) from a configured number of nodes.

It is designed to be easy to use yet powerfully flexible and accurate in its simulation of the chef-client run.

It works well at this point but there is always room for improvement so please provide feedback.

## Requirements

#### Build the chef-load executable

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

The client defined by "client_name" in the chef-load configuration file needs to be able to create clients.

By default only admin users can create clients so the recommendation is to set "client_name" and
"client_key" to an admin user of the org.

If you want to use a regular user or even a regular client instead then you will need
to use the [knife-acl plugin](https://github.com/chef/knife-acl) to create a new group in the
Chef Server, add the regular user or client to that group and then give the group the create
permission on the clients container.

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

Configure at least "chef_server_url", "client_name" and "client_key" in chef-load.conf.

You can set the number of nodes and chef-client runs as command line options.  
This is useful for quickly testing the chef-load configuration.  

The following will make one node perform two chef-client runs.

```
chef-load --config chef-load.conf --nodes 1 --runs 2
```

Run chef-load using only the configuration file.

```
chef-load --config chef-load.conf
```
