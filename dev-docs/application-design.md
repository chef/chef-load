# chef-load Solution Description

## Purpose

`chef-load` is a Go-based load-generation tool for Chef Server and Chef Automate. It simulates Chef Infra Client activity for configurable node populations and can produce:

- Chef Client Runs (CCRs), including node/client lifecycle, run-list expansion, cookbook resolution, optional cookbook file downloads, node saves, and optional Chef Reporting messages.
- Chef Automate Data Collector messages for run start, run converge, actions, liveness pings, and compliance reports.
- Historical or synthetic InSpec compliance scans.
- Random Chef Server action traffic.

It generates traffic and payloads; it does not provision virtual machines, install Chef Server, or execute recipes on real operating-system nodes.

## Application Structure

The application has two layers:

1. `main.go` is the process entry point. It calls `commands.Execute()` and contains no load-generation logic.
2. `commands/` provides the Cobra command-line interface and Viper configuration loading. It translates command-line flags or a TOML configuration into `lib.Config`, validates the required endpoints and credentials, and invokes the library.

The primary commands are:

| Command | Responsibility |
| --- | --- |
| `chef-load start` | Runs continuous node/CCR load, with optional actions and liveness traffic. |
| `chef-load generate` | Generates a finite batch of CCRs, actions, compliance reports, and optional liveness data; supports historical data with `days_back`. |
| `chef-load init` | Prints a sample configuration. |
| `chef-load version` | Prints the application version. |

## Runtime Startup

The continuous load test is started as follows:

```text
main.go
  -> commands.Execute()
  -> Cobra start command
  -> configFromViper()
  -> lib.Start(config)
```

`configFromViper` starts from `lib.Default()`, overlays TOML and flags, and requires at least one of `chef_server_url` or `data_collector_url`. When a Chef Server URL is configured, it enables `RunChefClient` and requires `client_name` and `client_key`. The URL must include the Chef organization, for example `/organizations/demo/`.

`lib.Start` then:

1. Creates a JSON log file and an API request aggregator.
2. Calculates the delay between simulated converges from `interval / num_nodes`, which spreads node activity across the requested interval.
3. Starts optional goroutines for:
	- liveness-agent pings, when `liveness_agent` is enabled;
	- random Chef actions, when a Data Collector URL and `num_actions` are configured.
4. Creates the configured node slots and schedules their first CCR.
5. Runs each CCR in a goroutine through `ChefClientRun`, limiting active work to the configured node slots. Each completed run signals the scheduler so that the slot can run again.
6. Optionally replaces a node slot with a newly named node according to `node_replacement_rate`.
7. Continues until interrupted. SIGINT, SIGTERM, or SIGUSR1 causes the request profile to be printed before the process exits.

The separate `generate` command uses `lib.GenerateData`. It runs finite workloads concurrently through `GenerateCCRs`, `GenerateChefActions`, `GenerateComplianceData`, and, when enabled, `GenerateLivenessData`, then prints an aggregate API request profile.

## Simulated Chef Client Run

`lib/chef_client_run.go` contains the main CCR state machine. For each node it:

1. Builds node identity, UUIDs, timestamps, Ohai data, and optional converge/compliance payloads.
2. Selects the configured load mode:
	- `legacy`: run lists, roles, and environments;
	- `policyfile`: Policyfile API for every node;
	- `mixed`: a configured fraction of node slots use Policyfiles.
3. When Chef Server traffic is enabled, creates or fetches the client and node, then resolves the configured run list or fetches the Policyfile.
4. Resolves cookbook versions and optionally downloads cookbook files. Policyfile nodes use cookbook artifact manifests; legacy nodes use the environment cookbook-versions endpoint.
5. Waits for the configured synthetic converge duration.
6. Optionally saves the node and sends Chef Reporting start/end messages.
7. Sends Data Collector `run_start`, `run_converge`, and CCR update-action messages.
8. Optionally sends an `inspec_report` compliance message, either from a configured JSON file or a synthetic minimal report.

The run is a protocol-level simulation. No Chef client process is launched and no cookbook recipe is executed locally.

## Infrastructure Connectivity

### Chef Server

Chef Server connectivity is implemented by `lib/util.go` and the `github.com/go-chef/chef` dependency:

- `getAPIClient` reads the configured client private key and creates a signed Chef API client.
- `apiRequest` builds and executes authenticated Chef API requests, adding `X-Ops-Server-Api-Version` and `X-Chef-Version` headers.
- `ChefClientRun`, `generator.go`, `run_list.go`, `role.go`, `cookbooks.go`, and `policyfile.go` use that client for Chef Server resources.

The exercised Chef Server APIs include clients, nodes, roles, environments, cookbook versions, cookbook files, Policyfiles, optional reports, and the authenticated `/data-collector` endpoint. SSL verification is disabled for the Chef client and Data Collector HTTP transport, so deployments should provide network isolation and appropriate endpoint controls.

### Chef Automate Data Collector

`lib/data_collector.go` defines `DataCollectorClient`. It is a separate HTTP client that:

- posts JSON messages to the configured `data_collector_url`;
- authenticates with `x-data-collector-auth` and `x-data-collector-token`, or `Authorization: Bearer dev` for the development token;
- records request method, URL, status, and duration for profiling.

When `data_collector_url` is set, CCR lifecycle messages, actions, liveness events, and compliance reports are sent directly to that endpoint. When it is absent but `chef_server_url` is set, CCR Data Collector messages are sent through Chef Server using `POST data-collector`; the Chef Server may forward them to Automate if its authenticated Data Collector integration is enabled.

### Server Automation Boundary

The application automates server-facing API traffic rather than server management:

- Chef Server is driven through the signed API client in `go-chef/chef`.
- Chef Automate is driven through the custom authenticated HTTP client in `data_collector.go`.
- There is no SSH client, cloud SDK, VM lifecycle controller, or remote command executor in `main.go` or `lib/`.
- `Vagrantfile` and Habitat files package or run the tool; they are not part of the CCR request engine.

## Module Map

| Module | Responsibility |
| --- | --- |
| `config.go` | Configuration schema, defaults, and sample TOML output. |
| `service.go` | Continuous `start` scheduler, goroutines, throttling, replacement, signals, and request profiling. |
| `chef_client_run.go` | One simulated Chef Client Run from node setup through converge and reporting. |
| `generator.go` | Finite CCR generation, concurrency batching, random node/run payloads, and historical timestamps. |
| `data_collector.go` | Data Collector HTTP transport and run/compliance message construction. |
| `chef_actions.go` | Random Chef action event types and action payloads. |
| `compliance_generator.go` | Synthetic node identities and sample InSpec report generation over a time range. |
| `liveness_agent.go` | Liveness `node_ping` payloads and continuous/finite liveness traffic. |
| `policyfile.go` | Policyfile retrieval, run-list extraction, and cookbook artifact resolution. |
| `run_list.go` | Parsing run lists, recursively expanding roles, and resolving cookbook dependencies. |
| `role.go` | Role retrieval and environment-specific role run lists. |
| `cookbooks.go` | Cookbook artifact/file models and probabilistic file downloads. |
| `reporting.go` | Optional Chef Reporting run start/end requests. |
| `util.go` | Chef API client creation, HTTP request wrapper, JSON input, logging, and request aggregation. |
| `random_data.go` | Static pools used to build realistic synthetic nodes, actions, resources, and compliance data. |
| `version.go` | Build/application version value. |

## End-to-End Flow

```text
CLI flags or TOML
		  |
		  v
lib.Config validation
		  |
		  +--> Start: continuous scheduler --> ChefClientRun goroutines
		  |                                      |
		  |                                      +--> Chef Server API (signed go-chef client)
		  |                                      +--> Data Collector directly (custom HTTP client)
		  |                                      +--> Chef Server /data-collector proxy
		  |                                      +--> optional Reporting and compliance messages
		  |
		  +--> Generate: finite concurrent generators
															  |
															  +--> CCRs, actions, InSpec, liveness
															  +--> request profile
```

The result is a controllable, repeatable source of Chef platform traffic for performance, capacity, ingestion, and integration testing against Chef Server and Chef Automate.
