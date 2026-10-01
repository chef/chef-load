# Running a realistic Chef Load test against Chef 360 DSM

> **Article outline:** A practical guide for operators who want to download
> `chef-load`, generate realistic Chef Infra Client or InSpec traffic, and
> compare the results with an on-premises Chef 360 Deployment Service Manager
> (DSM) deployment.

## 1. Purpose and scope

- Explain what the sample load test measures and what it does not measure.
- Define the target audience: operators sizing or validating an on-premises
  Chef 360 DSM deployment.
- Link to the [Chef 360 DSM documentation](https://docs.chef.io/360/1.7/dsm/).
- State the assumptions:
  - The operator has access to a Chef 360 DSM service.
  - The Automate data collector endpoint is available.
  - The operator has already used the DSM sizing tool to determine an initial
    Kubernetes cluster shape.
  - The test is run in a non-production or explicitly approved environment.

## 2. Prerequisites and test inputs

- Confirm the required `chef-load` release, operating system, network access,
  credentials, and TLS certificates.
- Gather the DSM Automate data collector URL, authentication details, and any
  tenant or organization identifiers required by the service.
- Choose the workload family and version:
  - Chef Infra Client 18
  - Chef Infra Client 19
  - InSpec 5
  - InSpec 6
  - InSpec 7
- Explain how to select a version that matches the operator's current fleet.
- Describe the test data needed:
  - Current Chef Infra Client node and run history data.
  - Runlists downloaded from representative nodes or environments.
  - InSpec profiles and inputs downloaded from representative compliance
    scans.
  - A representative mix of node counts, run durations, cookbook sizes,
    profile sizes, and reporting frequency.
- Warn against using synthetic data that is substantially smaller or simpler
  than the production workload.

## 3. Download and install `chef-load`

- Identify the release artifact or repository download location.
- Show how to download and verify the archive or binary.
- Show how to install or unpack it without overwriting an existing test
  installation.
- Verify the installation with the version/help command.
- Explain where the sample configuration and default TOML file are located.

## 4. Configure the sample load

### 4.1 Select the client or InSpec version

- Show the version-selection option or TOML setting.
- Provide a short example for each supported choice:
  - `chef-client` 18
  - `chef-client` 19
  - `inspec` 5
  - `inspec` 6
  - `inspec` 7
- Explain that the selected version should reflect the data shape and event
  behavior of the operator's current estate.

### 4.2 Configure the Automate data collector endpoint

- Show where to set the DSM Automate data collector endpoint.
- Show how to provide credentials and TLS settings securely.
- Explain how to confirm that the endpoint is the intended DSM service before
  starting a sustained test.
- Call out any required proxy, certificate-authority, timeout, retry, or
  concurrency settings.

  http://chef-ingest-service:3001/organizations/*/data-collector - ask Yadav (do I need the data_collector token? in request body or as header?)

will also hit regular client endpoints: node/, runlist/

### 4.3 Default TOML file parameters

Document the default TOML file in a reference table. For every parameter,
include its exact key, type, default, valid range or values, and operational
effect.

| Parameter group | Parameters to document | Why it matters |
|---|---|---|
| Endpoint and authentication | Automate data collector URL, credentials, TLS/CA settings | Determines the DSM target and connection security |
| Workload selection | Chef Infra Client/InSpec product and version | Controls the generated request and payload shape |
| Test duration | Start time, duration, steady-state period, ramp-up, ramp-down | Makes runs comparable |
| Load rate | Nodes, runs, scans, events per second, or equivalent rate controls | Defines the offered workload |
| Concurrency | Workers, batches, queues, and connection limits | Controls parallel pressure on DSM |
| Realistic data | Node data, runlists, cookbooks, profiles, inputs, and fixtures | Prevents underrepresenting payload size and variability |
| Reliability | Timeout, retry, backoff, and failure thresholds | Separates transient failures from capacity limits |
| Output | Log level, metrics, report path, and run identifiers | Supports analysis and repeatability |

- Include a complete, commented default TOML example.
- Distinguish settings that should remain at defaults from settings the
  operator must change.
- Explain how command-line overrides interact with TOML values.
- Note which values must not be committed to source control, such as tokens
  and private keys.

## 5. Prepare realistic test data

### 5.1 Capture current Chef Infra Client data

- Explain how to select a representative sample of current nodes rather than
  only the smallest or healthiest nodes.
- Download or export current node data and run history using the supported
  Chef tooling.
- Preserve realistic cookbook, environment, role, policy, and runlist
  relationships.
- Remove secrets and unrelated personally identifiable information while
  preserving payload sizes and field distributions.
- Record the source population, sample size, and timestamp.

### 5.2 Download runlists and related artifacts

- Show the supported workflow for downloading runlists from the selected
  nodes or environments.
- Include cookbook and policy dependencies needed to reproduce payload size.
- Explain how to avoid accidentally testing an empty or default runlist.
- Verify that downloaded fixtures can be parsed before starting the load.

### 5.3 Download InSpec profiles

- Select profiles and inputs that represent current compliance usage.
- Download profile dependencies and preserve realistic control counts and
  metadata.
- Verify each profile with the selected InSpec major version.
- Include a mix of small, medium, and large profiles where that reflects the
  fleet.

### 5.4 Build a workload matrix

Define repeatable scenarios, for example:

| Scenario | Product/version | Data set | Offered load | Goal |
|---|---|---|---|---|
| Baseline | Current fleet version | Small representative sample | Low | Establish healthy latency and error baselines |
| Expected load | Current fleet version | Production-shaped sample | Expected peak | Validate the sizing-tool recommendation |
| Peak | Current fleet version | Production-shaped sample | Peak plus headroom | Find saturation behavior |
| Version comparison | 18/19 or InSpec 5/6/7 | Same data where compatible | Same rate | Identify version-related differences |

## 6. Run the sample load

- Start with a connectivity and low-rate smoke test.
- Show the command that uses the TOML file and selected version.
- Show how to ramp gradually to the expected and peak rates.
- Record the exact configuration, fixture revision, cluster shape, DSM image
  version, and test timestamps for every run.
- Explain what successful requests, retries, timeouts, and rejected requests
  mean.
- Stop the test safely and preserve logs, metrics, and the generated summary.

## 7. Monitor DSM while the test runs

- Correlate `chef-load` timestamps with DSM and Kubernetes metrics.
- Monitor request rate, latency percentiles, error rate, queue depth, and
  retries at the data collector boundary.
- Monitor DSM pod CPU, memory, garbage collection, restarts, throttling, and
  network/disk pressure.
- Pay particular attention to `dsm_erchef` replicas and the point at which
  additional replicas no longer reduce saturation.
- Capture Kubernetes replica counts, node utilization, scheduling events, and
  autoscaling decisions.
- Explain how to distinguish application saturation from node, network,
  storage, or downstream-service limits.

## 8. Read and compare the results

- Define the key numbers:
  - Sustained requests or events per second.
  - Successful and failed request counts.
  - p50, p95, and p99 latency.
  - Timeout and retry rates.
  - Work completed during the steady-state window.
  - CPU and memory utilization per DSM pod and per Kubernetes node.
  - Replica count and queue/backlog behavior.
- Compare runs only when the workload matrix, data set, DSM version, and test
  window are equivalent.
- Explain how to compare observed capacity with the sizing-tool result:
  - Treat the sizing-tool output as the initial cluster recommendation.
  - Use the load results and pod counters to validate the recommendation
    under the expected and peak workload.
  - Leave operational headroom; do not size to the first visible saturation
    point.
- Include a sample results table and a short interpretation example.
- Explain why throughput alone is insufficient when latency or errors rise.

## 9. What to do when DSM pods are saturated

### 9.1 Confirm the saturation signal

- Verify that pod CPU or memory, queue depth, latency, or error rate is the
  limiting factor.
- Check whether pods are already distributed across available nodes.
- Rule out a constrained node, storage, network, or downstream dependency.

### 9.2 Scale out the Kubernetes cluster

- Add Kubernetes worker nodes when existing nodes cannot provide the required
  CPU or memory headroom.
- Confirm that new nodes are schedulable and that DSM pods can be placed on
  them.
- Re-run the same workload scenario and compare the before/after metrics.

### 9.3 Adjust DSM pod scalability

- Increase the replica count or autoscaling limits for the affected DSM
  components, specifically `dsm_erchef`, using the supported Chef 360
  deployment configuration.
- Confirm that resource requests and limits, scheduling constraints, and
  pod-disruption settings permit the additional replicas to run.
- Re-test to determine whether scaling `dsm_erchef` improves throughput and
  latency or simply moves the bottleneck elsewhere.
- Document the final node count, replica settings, observed headroom, and
  rollback values.

## 10. Troubleshooting and safety

- Endpoint authentication or TLS failures.
- Invalid version, profile, runlist, or fixture errors.
- Excessive retries caused by an overly aggressive ramp.
- Client-side bottlenecks that make DSM appear underutilized.
- Pods restarting, pending, throttled, or evicted.
- How to stop a test without leaving background workers running.
- Data-handling guidance for credentials, node data, profiles, and logs.

## 11. Reproducibility checklist

- [ ] `chef-load` version and checksum recorded.
- [ ] Chef Infra Client or InSpec version recorded.
- [ ] TOML file and command-line overrides archived without secrets.
- [ ] Automate data collector endpoint and DSM version recorded.
- [ ] Fixture source, sample size, and download timestamp recorded.
- [ ] Kubernetes node count, pod replicas, and resource settings recorded.
- [ ] Expected and peak load rates recorded.
- [ ] Latency, error, saturation, and scaling results retained.
- [ ] Follow-up sizing or deployment changes documented.

## 12. Reference material

- [Chef 360 DSM documentation](https://docs.chef.io/360/1.7/dsm/)
- `chef-load` installation and command reference.
- The generated/default TOML configuration reference.
- Kubernetes and DSM monitoring dashboards used during the test.