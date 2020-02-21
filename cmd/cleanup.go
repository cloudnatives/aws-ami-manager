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

var (
	tagsToMatch    []string
	versionsToKeep int
)

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Cleanup earlier versions of the AMI",
	Long: `Cleanup earlier versions in the different regions. 

It keeps the most recent version with the same tags and AMI's that are currently in use.		
	`,
	Run: func(cmd *cobra.Command, args []string) {
		runCleanup()
	},
}

func runCleanup() {
	cm := aws.NewConfigurationManager()

	ami := aws.NewAmi(amiID)
	ami.SourceRegion = cm.GetDefaultRegion()

	aws.ConfigManager = cm

	err := ami.Cleanup(regions, tagsToMatch, versionsToKeep)

	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Older AMI's related to %s has been cleaned up successfully", ami.SourceAmiID)
}

func init() {
	rootCmd.AddCommand(cleanupCmd)

	cleanupCmd.Flags().StringVar(&amiID, "amiID", "", "The source AMI ID, e.g. aws-0e38957fc6310ea8b")
	_ = cleanupCmd.MarkFlagRequired("amiID")

	cleanupCmd.Flags().StringSliceVar(&regions, "regions", []string{}, "The regions to copy this AMI to. Can be multiple flags, or a comma-separated value")
	_ = cleanupCmd.MarkFlagRequired("regions")

	cleanupCmd.Flags().StringSliceVar(&tagsToMatch, "tags", []string{}, "The tags to filter the AMI's on. Can be multiple flags, or a comma-separated value")
	_ = cleanupCmd.MarkFlagRequired("regions")

	cleanupCmd.Flags().IntVar(&versionsToKeep, "versions-to-keep", 5, "The number of AMI's you would like to keep. Defaults to 5.")
}
