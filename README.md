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

Make sure chef-load.conf has appropriate settings for applying load to your Chef Server,
Automate Server or both.

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

You can set the desired Chef Client runs per minute rate using the `--rpm` command line option. This is useful for quickly adjusting the rate without modifying the configuration file.

```
chef-load --config chef-load.conf --rpm 30
```

You can also set the desired interval between each node's Chef Client run using the `--interval` option. The default is 30 minutes.

```
chef-load --config chef-load.conf --rpm 30 --interval 1
```
