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
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: fmt.Sprintf("Initialize chef-load configuration file"),
	Run: func(cmd *cobra.Command, args []string) {
		// TODO @afiune Instead of printing the config, write it to disk
		// with the default config and tell the user that they can modify it
		chef_load.PrintSampleConfig()
	},
}

func init() {
	// TODO: Output file param?
	rootCmd.AddCommand(initCmd)
}
