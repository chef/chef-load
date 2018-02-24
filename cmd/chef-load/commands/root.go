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
	Long: `A tool for simulating load on a Chef Server and/or a Chef Automate server.
                Complete documentation is available at https://github.com/chef/chef-load`,
	Run: func(cmd *cobra.Command, args []string) {
	},
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
	rootCmd.PersistentFlags().StringVarP(&num, "number", "", "The number of nodes or actions to simulate") //TODO: Note change from "nodes", update docs
	rootCmd.PersistentFlags().StringVarP(&interval, "interval", "", "Interval between a node's chef-client runs, in minutes")
	rootCmd.PersistentFlags().StringVarP(&sampleConfig, "sample-config", false, "Print out full sample configuration")
	rootCmd.PersistentFlags().Bool("profile-logs", false, "Generates API request profile from specified chef-load log files")
	rootCmd.PersistentFlags().Bool("version", false, "Print chef-load version")
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
