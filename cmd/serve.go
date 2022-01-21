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
	"os"

	"github.com/BetssonGroup/fluenthose/pkg/firehose"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve a HTTP endpoint for Kineses Data Firehose",
	Run: func(cmd *cobra.Command, args []string) {
		setupLogging()
		accessKey := os.Getenv("ACCESS_KEY")
		if accessKey == "" {
			cobra.CheckErr("ACCESS_KEY environment variable is required")
		}
		log.Info("log-level: %s", log.GetLevel())
		firehose.RunFirehoseServer(
			cmd.Flag("listen").Value.String(),
			accessKey,
			cmd.Flag("forward").Value.String(),
			cmd.Flag("event-type-header-name").Value.String(),
		)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	serveCmd.Flags().StringP("listen", "l", ":8080", "Listen address")
	serveCmd.Flags().StringP("forward", "f", "127.0.0.1:24224", "Forward address")
	// Set event type header name
	serveCmd.Flags().StringP("event-type-header-name", "e", "X-EVENT-TYPE", "Event type header name")
}
