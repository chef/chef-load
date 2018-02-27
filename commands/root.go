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
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "chef-load",
	Short: "`A tool for simulating loading chef data",
	Long: `A tool for simulating load on a Chef Server and/or a Chef Automate Server.
         Complete documentation is available at https://github.com/chef/chef-load`,
	Run: func(cmd *cobra.Command, args []string) {},
}

// Execute adds all child commands to the root command sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	//rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.chef-load.toml)")
	//rootCmd.PersistentFlags().StringVarP(&nodes, "nodes", "", "The number of nodes to simulate")
	//rootCmd.PersistentFlags().StringVarP(&interval, "interval", "", "Interval between a node's chef-client runs, in minutes")
	rootCmd.PersistentFlags().Bool("profile-logs", false, "Generates API request profile from specified chef-load log files")
	rootCmd.PersistentFlags().Bool("random-data", false, "Generates random data")
}

func initConfig() {
	viper.SetConfigName(".chef-load")
	viper.AddConfigPath("$HOME")
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}
}
