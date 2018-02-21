package commands

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/chef/a2/components/config-mgmt-service/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var nodes = &cobra.Command{
	Use:   "nodes",
	Short: fmt.Sprintf("Generates node data", config.Default().Host, config.Default().Port),
	Run: func(cmd *cobra.Command, args []string) {
		conf, err := config.ConfigFromViper()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Failed to configure config-mgmt service")
		}

		fmt.Printf("%s Starting chef-load with %d nodes distributed evenly across a %d minute interval\n", time.Now().UTC().Format(iso8601DateTime), config.NumNodes, config.Interval)
		fmt.Printf("%s All API requests will be logged in %s\n", time.Now().UTC().Format(iso8601DateTime), config.LogFile)
		delayBetweenNodes := time.Duration(math.Ceil(float64(time.Duration(config.Interval)*(time.Minute/time.Nanosecond))/float64(config.NumNodes))) * time.Nanosecond
		firstRun := true
		for {
			for i := 1; i <= config.NumNodes; i++ {
				nodeName := config.NodeNamePrefix + "-" + strconv.Itoa(i)
				go chefClientRun(nodeClient, nodeName, firstRun, ohaiJSON, convergeJSON, complianceJSON)
				time.Sleep(delayBetweenNodes)
			}
			firstRun = false
		}

		log.Info("Starting config mgmt API service")

	},
}

func init() {
	RootCmd.AddCommand(nodes)
}
