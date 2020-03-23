package chef_load

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"strings"

	"time"

	"github.com/icrowley/fake"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

const (
	compRecipes      = "compliance_recipes"
	compRoles        = "compliance_roles"
	compEnvironments = "compliance_environments"
)

type NodeDetails struct {
	name        string
	ipAddr      string
	environment string
	roles       []string
	recipes     []string
	nodeUUID    uuid.UUID
	platform    string
	sourceFqdn  string // fqdn of chef server that ran this
	fqdn        string // fqdn of the box that we're scanning
	policyName  string
	policyGroup string
	orgName     string
	chefTags    []string

	//// "source_fqdn": "localhost",
	//  "organization_name": "",
	//  "policy_group": "",
	//  "policy_name": "",
	//  "chef_tags": [],
	////  "fqdn": "myapache.example.com"
}

func GenerateComplianceData(config *Config, requests chan *request) error {
	log.Infof("---> Load simulation config from matrix")
	platforms := config.Matrix.Samples.Platforms
	nodesCount := config.Matrix.Simulation.Nodes

	log.Infof("generating %d nodes for %d platforms", nodesCount, len(platforms))
	nodes := generateNodes(config.NodeNamePrefix, platforms, nodesCount)
	log.Infof("nodes %v", nodes)
	generateReports(config, nodes, requests)
	return nil
}

func ip2int(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func int2ip(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}

func generateNodeName(nodeNamePrefix string) string {
	return strings.ToLower(fmt.Sprintf("%s-%s-%s-%s", nodeNamePrefix, fake.Color(), strings.Fields(fake.Street())[0], fake.Color()))
}

func generateIpAddress() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%d.%d.%d.%d", r.Intn(255), r.Intn(255), r.Intn(255), r.Intn(255))
}

func generateSourcFqdn() string {
	rand.Seed(time.Now().UnixNano())
	data := []string{
		"chefserver1.foo.bar",
		"alex.kung.foo.arm.bar",
		"rick.kung.foo.arm.bar",
	}
	return data[rand.Intn(len(data))]
}

func generateChefOrgs() string {
	rand.Seed(time.Now().UnixNano())
	data := []string{
		"org1",
		"org2",
		"org3",
		"org4",
		"org5",
		"org6",
		"org7",
		"org8",
		"org9",
		"org10",
	}
	return data[rand.Intn(len(data))]
}

func generateChefTags() []string {
	rand.Seed(time.Now().UnixNano())
	data := []string{
		"tag1",
		"tag2",
		"tag3",
		"tag4",
		"tag5",
		"tag6",
		"tag7",
		"tag8",
		"tag9",
		"tag10",
	}
	//shuffle the list of tags and then return a random number of them
	rand.Shuffle(len(data), func(i, j int) { data[i], data[j] = data[j], data[i] })
	return data[0:rand.Intn(len(data))]
}

func generatePolicyGroup() string {
	rand.Seed(time.Now().UnixNano())
	data := []string{
		"policy.group1",
		"policy.group2",
		"policy.group3",
		"policy.group4",
	}
	return data[rand.Intn(len(data))]
}

func generatePolicyName() string {
	rand.Seed(time.Now().UnixNano())
	data := []string{
		"policy.name1",
		"policy.name2",
		"policy.name3",
		"policy.name4",
	}
	return data[rand.Intn(len(data))]
}

func generateNodes(nodeNamePrefix string, platforms []Platform, nodesCount int) (nodes []NodeDetails) {

	// add missing nodes until we have enough
	for len(nodes) < nodesCount {
		node := NodeDetails{
			// TODO: we can have multiple nodes with the same node name
			name:        generateNodeName(nodeNamePrefix),
			ipAddr:      generateIpAddress(),
			sourceFqdn:  generateSourcFqdn(),
			environment: getRandom(compEnvironments),
			roles:       getRandomStringArray(compRoles),
			recipes:     getRandomStringArray(compRecipes),
			orgName:     generateChefOrgs(),
			chefTags:    generateChefTags(),
			policyGroup: generatePolicyGroup(),
			policyName:  generatePolicyName(),
			platform:    platforms[rand.Intn(len(platforms))].Name,
		}
		node.fqdn = node.name
		node.nodeUUID = uuid.NewV3(uuid.NamespaceDNS, node.name)
		nodes = append(nodes, node)
	}
	return nodes
}

func intervalMinutes(nodesCount int, index int, maxScansPerDay int) int {
	divisor := float32(index) / float32(nodesCount)

	if divisor <= .1 { //10%
		return 1440 / maxScansPerDay
	} else if divisor <= .4 { //30%
		return 1440
	} else if divisor <= .8 { //40%
		return 10080
	} else {
		return 43200
	}
}

func intervalToString(minutes int) string {
	hours := minutes / 60
	if hours < 24 {
		return fmt.Sprintf("%d hour(s)", hours)
	} else {
		return fmt.Sprintf("%d day(s)", hours/24)
	}
}

func loadSampleReport(config *Config, platform string, format string) map[string]interface{} {
	complianceJSON := parseJSONFile(fmt.Sprintf("%s/%s-%s.json", config.ComplianceSampleReportsDir, platform, format))
	return complianceJSON
}

//func generate_reports(nodes []NodeDetails, platforms []string, simulation, handler) {
func generateReports(config *Config, nodes []NodeDetails, requests chan *request) {
	endTime := time.Now().UTC()

	dataCollectorClient, _ := NewDataCollectorClient(&DataCollectorConfig{
		Token:   config.DataCollectorToken,
		URL:     config.DataCollectorURL,
		SkipSSL: true,
	}, requests)

	var nodesCount = config.Matrix.Simulation.Nodes
	if nodesCount < len(nodes) {
		nodesCount = len(nodes)
	}

	log.Infof("Generating Inspec reports over a period of %d day(s)", config.Matrix.Simulation.Days)
	totalScans := 0
	nodeIndex := 0
	totalMaxScans := config.Matrix.Simulation.TotalMaxScans
	for nodeIndex < nodesCount && totalScans < totalMaxScans {
		node := nodes[nodeIndex]
		sampleReport := loadSampleReport(config, node.platform, config.Matrix.Simulation.SampleFormat)
		interval := intervalMinutes(nodesCount, nodeIndex+1, config.Matrix.Simulation.MaxScans)
		log.Infof("Generating Inspec reports for node %s (%d/%d) with interval of %s , scans so far: %d", node.name, nodeIndex+1, nodesCount, intervalToString(interval), totalScans)
		maxScansNode := (config.Matrix.Simulation.Days*24*60)/interval + 1
		scanIndex := maxScansNode
		for scanIndex > 0 && totalScans < totalMaxScans {
			scanIndex -= 1
			report := sampleReport
			reportUUID, _ := uuid.NewV4()
			reportEndTime := endTime.Add(time.Duration(-interval*scanIndex) * time.Minute)
			complianceReportBody := dataCollectorComplianceReport(node, reportUUID, reportEndTime, report)

			if config.DataCollectorURL != "" {
				chefAutomateSendMessage(dataCollectorClient, node.name, complianceReportBody)
			}

			if scanIndex > 0 && scanIndex%500 == 0 {
				log.Info(scanIndex)
			}
		}
		totalScans += maxScansNode
		nodeIndex += 1
	}
}
