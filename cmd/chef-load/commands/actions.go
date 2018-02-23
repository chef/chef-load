package commands

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/chef/a2/components/config-mgmt-service/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var nodes = &cobra.Command{
	Use:   "actions",
	Short: fmt.Sprintf("Generates chef actions data", config.Default().Host, config.Default().Port),
	Run: func(cmd *cobra.Command, args []string) {
		conf, err := config.ConfigFromViper()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Failed to configure chef-load service")
		}

		log.Info("Generating chef-server actions")
		rand.Seed(time.Now().UTC().UnixNano())
		for i := 1; i <= 10; i++ {
			// TODO: Check the errors
			chefAction(randomActionType())
		}

	},
}

func init() {
	rootCmd.AddCommand(nodes)
}
