package chef_load

import (
	"fmt"
	"math/rand"
	"strings"

	"time"

	"github.com/icrowley/fake"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

const (
	compRecipes      = "compliance_recipes"
	compRoles        = "compliance_roles"
	compEnvironments = "compliance_environments"
)

type Node struct {
	name        string
	environment string
	roles       []string
	recipes     []string
	nodeUUID    uuid.UUID
	platform    string
}

func GenerateComplianceData(config *Config, requests chan *request) error {
	log.Infof("---> Load simulation config from matrix")
	platforms := config.Matrix.Samples.Platforms
	nodesCount := config.Matrix.Simulation.Nodes

	log.Infof("generating %d nodes for %d platforms", nodesCount, len(platforms))
	nodes := generateNodes(platforms, nodesCount)
	log.Infof("nodes %v", nodes)
	generateReports(config, nodes, requests)
	return nil
}

func generateNodeName() string {
	return strings.ToLower(fmt.Sprintf("%s-%s", fake.Color(), strings.Fields(fake.Street())[0]))
}

func generateNodes(platforms []Platform, nodesCount int) (nodes []Node) {
	// add missing nodes until we have enough
	for len(nodes) < nodesCount {
		node := Node{
			// TODO: we can have multiple nodes with the same node name
			name:        generateNodeName(),
			environment: getRandom(compEnvironments),
			roles:       getRandomStringArray(compRoles),
			recipes:     getRandomStringArray(compRecipes),

			platform: platforms[rand.Intn(len(platforms))].Name,
		}

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
	hours := minutes/60
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

//func generate_reports(nodes []Node, platforms []string, simulation, handler) {
func generateReports(config *Config, nodes []Node, requests chan *request) {
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
			complianceReportBody := dataCollectorComplianceReport(node.name, node.environment, node.roles, node.recipes, reportUUID, node.nodeUUID, reportEndTime, report)

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
