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
	"time"

	"github.com/cloudnatives/aws-ami-manager/aws"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	regions  []string
	accounts []string
)

// copyCmd represents the copy command
var copyCmd = &cobra.Command{
	Use:   "copy",
	Short: "Copies an AMI to a list of AWS regions and accounts",
	Long: `
		
	`,
	Run: func(cmd *cobra.Command, args []string) {
		runCopy()
	},
}

func runCopy() {
	log.Infof("Started copying AMI %s", amiID)
	start := time.Now()

	loadAWSConfigForProfiles()

	ami := aws.NewAmiWithRegions(&amiID, aws.ConfigManager.GetDefaultRegion(), &regions)
	ami.Copy()

	elapsed := time.Since(start)
	log.Infof("Finished copying AMI after %s", elapsed)
}

func init() {
	rootCmd.AddCommand(copyCmd)

	copyCmd.Flags().StringVar(&amiID, "amiID", "", "The source AMI ID, e.g. aws-0e38957fc6310ea8b")
	_ = copyCmd.MarkFlagRequired("amiID")

	copyCmd.Flags().StringSliceVar(&regions, "regions", []string{}, "The regions to copy this AMI to. Can be multiple flags, or a comma-separated value")
	_ = copyCmd.MarkFlagRequired("regions")

	copyCmd.Flags().StringSliceVar(&accounts, "accounts", []string{}, "The account ID's that will be authorized to use the Ami's. Can be multiple flags, or a comma-separated value")
	_ = copyCmd.MarkFlagRequired("regions")
}

func loadAWSConfigForProfiles() {
	aws.ConfigManager = aws.NewConfigurationManagerForRegionsAndAccounts(regions, accounts)
}
