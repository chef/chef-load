//
// Copyright:: Copyright 2018 Chef Software, Inc.
// License:: Apache License, Version 2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package commands

import (
	chef_load "github.com/chef/chef-load/lib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var generateCmd = &cobra.Command{
	Use:              "generate",
	Short:            "Generates specific number of chef nodes, actions and/or compliance reports",
	TraverseChildren: true,
	Run: func(cmd *cobra.Command, args []string) {
		config, err := configFromViper()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Could not load chef-load config file")
		}

		chef_load.GenerateData(config)
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().Int("days_back", 0, "The number days back for historical data")
	generateCmd.Flags().Int("threads", 3000, "Number of simultaneous goroutines to spawn for historical data")
	generateCmd.Flags().Int("sleep_time_on_failure", 5, "Time in seconds to sleep when a failure is detected for historical data")
	viper.BindPFlags(generateCmd.Flags())
}
