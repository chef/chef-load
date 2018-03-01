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
	"fmt"

	chef_load "github.com/chef/chef-load/lib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:              "generate",
	Short:            fmt.Sprintf("Generates data"),
	TraverseChildren: true,
	Run: func(cmd *cobra.Command, args []string) {
		config, err := configFromViper()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Could not load chef-load config file")
		}

		go chef_load.GenerateCCRs(config)
		chef_load.GenerateChefActions(config)
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
