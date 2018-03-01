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
	"errors"
	"fmt"
	"os"
	"strings"

	chef_load "github.com/chef/chef-load/lib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "chef-load",
	Short: "`A tool for simulating loading chef data",
	Long: `A tool for simulating load on a Chef Server and/or a Chef Automate Server.
Complete documentation is available at https://github.com/chef/chef-load`,
	TraverseChildren: true,
	Run:              func(cmd *cobra.Command, args []string) {},
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.chef-load.toml)")
	rootCmd.PersistentFlags().StringP("num_nodes", "n", "", "The number of nodes to simulate")
	rootCmd.PersistentFlags().StringP("num_actions", "a", "", "The number of actions to generate")
	rootCmd.PersistentFlags().BoolP("random_data", "r", false, "Generates random data")
	viper.BindPFlags(rootCmd.PersistentFlags())
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			fmt.Println("Unable read config:", err)
			os.Exit(1)
		}
	} else {
		// TODO: @afiune if the user doesn't provide a config file
		// should we load it from the home directory or somewhere else
		// also, should we instead as them to run `chef-load init`?
		viper.SetConfigName(".chef-load")
		viper.SetConfigType("toml")
		viper.AddConfigPath("$HOME")
	}
}

func configFromViper() (*chef_load.Config, error) {
	cfg := chef_load.Default()
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if cfg.ChefServerURL == "" && cfg.DataCollectorURL == "" {
		return nil, errors.New("You must set chef_server_url or data_collector_url or both")
	}

	if cfg.ChefServerURL != "" {
		cfg.RunChefClient = true
		if !strings.HasSuffix(cfg.ChefServerURL, "/") {
			cfg.ChefServerURL = cfg.ChefServerURL + "/"
		}
		if cfg.ClientName == "" || cfg.ClientKey == "" {
			return nil, errors.New("You must set client_name and client_key if chef_server_url is set")
		}
	}

	if cfg.DataCollectorURL != "" && cfg.ChefServerURL == "" {
		// make sure cfg.ChefServerURL is set to something because it is used
		// even when only in data-collector mode
		cfg.ChefServerURL = "https://chef.example.com/organizations/demo/"
	}

	return &cfg, nil
}
