/*
Copyright © 2021 Betsson Group AB

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fluenthose",
	Short: "Receive Kinesis Data Firehose events over HTTP and forward to FluentBit",
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentFlags().StringP("log-level", "", "info", "Log level")

}

func setupLogging() {
	log.SetFormatter(&log.JSONFormatter{})
	lvl, err := log.ParseLevel(rootCmd.PersistentFlags().Lookup("log-level").Value.String())
	if err != nil {
		cobra.CheckErr(err)
	}
	log.SetLevel(lvl)
}
