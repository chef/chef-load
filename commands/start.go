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

var startCmd = &cobra.Command{
	Use:              "start",
	Short:            "Start the load of nodes, actions and/or reports.",
	TraverseChildren: true,
	Run: func(cmd *cobra.Command, args []string) {
		config, err := configFromViper()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Could not load chef-load config file")
		}

		chef_load.Start(config)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().Bool("profile-logs", false, "Generates API request profile from specified chef-load log files")
	viper.BindPFlags(startCmd.Flags())
}
