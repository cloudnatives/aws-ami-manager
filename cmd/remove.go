// Copyright © 2019 Jeroen Schepens <jeroen@cloudnatives.be>
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

package cmd

import (
	"github.com/cloudnatives/aws-ami-manager/aws"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		runRemove()
	},
}

func runRemove() {
	cm := aws.NewConfigurationManager()

	ami := aws.NewAmi(amiID)
	ami.SourceRegion = cm.GetDefaultRegion()

	aws.ConfigManager = cm

	err := ami.RemoveAmi()

	if err != nil {
		log.Fatal(err)
	}

	log.Infof("AMI %s has been removed successfully", ami.SourceAmiID)
}

func init() {
	rootCmd.AddCommand(removeCmd)

	removeCmd.Flags().StringVar(&amiID, "amiID", "", "The source AMI ID, e.g. aws-0e38957fc6310ea8b")
	_ = removeCmd.MarkFlagRequired("amiID")

}
