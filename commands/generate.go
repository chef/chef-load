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

	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: fmt.Sprintf("Generates data"),
	Run: func(cmd *cobra.Command, args []string) {
		//conf, err := config.ConfigFromViper()
		//if err != nil {
		//log.WithFields(log.Fields{
		//"error": err,
		//}).Fatal("Failed to configure config-mgmt service")
		//}

		//fmt.Printf("%s Starting chef-load with %d nodes distributed evenly across a %d minute interval\n", time.Now().UTC().Format(iso8601DateTime), config.NumNodes, config.Interval)
		//fmt.Printf("%s All API requests will be logged in %s\n", time.Now().UTC().Format(iso8601DateTime), config.LogFile)
		//delayBetweenNodes := time.Duration(math.Ceil(float64(time.Duration(config.Interval)*(time.Minute/time.Nanosecond))/float64(config.NumNodes))) * time.Nanosecond
		//firstRun := true
		//for {
		//for i := 1; i <= config.NumNodes; i++ {
		//nodeName := config.NodeNamePrefix + "-" + strconv.Itoa(i)
		//go chefClientRun(nodeClient, nodeName, firstRun, ohaiJSON, convergeJSON, complianceJSON)
		//time.Sleep(delayBetweenNodes)
		//}
		//firstRun = false
		//}

	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
