//
// Copyright:: Copyright 2024 Chef Software, Inc.
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
	"os"

	chef_load "github.com/chef/chef-load/lib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var precreateCmd = &cobra.Command{
	Use:   "precreate",
	Short: "Pre-create all node client+node objects on the Chef Server.",
	Long: `Pre-create creates both client and node objects for every node in the
configured pool (num_nodes) on the Chef Server, then writes a JSON tracking
file (node_log_file) that records each node's name and deterministic UUID.

On subsequent runs of 'chef-load start' the log file is used to verify that
all nodes still exist (via the Chef Search API), and pre-creation is skipped
when all nodes are present.`,
	TraverseChildren: true,
	Run: func(cmd *cobra.Command, args []string) {
		config, err := configFromViper()
		if err != nil {
			log.WithField("error", err).Fatal("Could not load chef-load config file")
		}

		if runErr := chef_load.RunPrecreate(config); runErr != nil {
			log.WithField("error", runErr).Error("Pre-creation completed with errors")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(precreateCmd)
}
