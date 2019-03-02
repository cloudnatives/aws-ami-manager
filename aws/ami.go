package aws

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go/aws"
	log "github.com/sirupsen/logrus"
)

var (
	AWSConfigurationManager *ConfigurationManager
	ec2Services             = make(map[string]map[string]*ec2.EC2)
)

func getEC2ServiceForAccountAndRegion(account string, region string) *ec2.EC2 {
	if ec2Services[account] == nil {
		ec2Services[account] = make(map[string]*ec2.EC2)
	}

	if ec2Services[account][region] == nil {
		ec2Services[account][region] = ec2.New(AWSConfigurationManager.getConfigurationForAccountAndRegion(account, region))
	}
	return ec2Services[account][region]
}

type Ami struct {
	SourceAmiID   *string
	SourceRegion  *string
	SourceAmiName *string
	SourceAmiTags *[]ec2.Tag
	AWSImage      *ec2.Image

	AmisPerRegion map[string]*Ami
}

func NewAmi(sourceAmiID *string, sourceRegion *string, regions *[]string) Ami {
	ami := Ami{
		SourceAmiID:   sourceAmiID,
		SourceRegion:  sourceRegion,
		AmisPerRegion: convertRegionSliceToAmi(*regions),
	}

	return ami
}

func (ami *Ami) fetchMetadata() {
	log.Debug("Fetching metadata about the AMI")
	ec2svc := getEC2ServiceForAccountAndRegion(*AWSConfigurationManager.defaultAccountID, *ami.SourceRegion)

	var amiList []string
	amiList = append(amiList, *ami.SourceAmiID)

	describeImagesInput := ec2.DescribeImagesInput{
		ImageIds: amiList,
	}
	request := ec2svc.DescribeImagesRequest(&describeImagesInput)
	result, err := request.Send()

	if err != nil {
		log.Fatal(err)
	}

	images := result.Images

	if len(images) != 1 {
		log.Fatal("Invalid number of AMI ID's returned for AMI: %s", *ami.SourceAmiID)
	}

	ami.AWSImage = &images[0]

	// With newly copied AMI's name and tags can still be empty, causing nil references
	// And only set name and tags when not yet already set
	if ami.SourceAmiName == nil && ami.AWSImage.Name != nil {
		ami.SourceAmiName = ami.AWSImage.Name
		log.Debugf("AMI name: %s", *ami.SourceAmiName)
	}

	if ami.SourceAmiTags == nil && images[0].Tags != nil {
		ami.SourceAmiTags = &images[0].Tags
		log.Debugf("AMI tags: %s", ami.SourceAmiTags)
	}
}

func (ami *Ami) Copy() {
	// Fetch name and tags for the source AMI
	ami.fetchMetadata()

	// in this loop region is the key
	for region := range ami.AmisPerRegion {
		var (
			relatedAmi *Ami
			err        error
		)

		// We obviously don't have to copy the AMI to a region where it already exists
		if region != *ami.SourceRegion {
			relatedAmi, err = ami.copyToRegion(region)

			if err != nil {
				log.Fatal(err)
			}

			err = relatedAmi.setOwners(AWSConfigurationManager.accounts)

			if err != nil {
				log.Fatal(err)
			}
		} else {
			relatedAmi = ami
		}

		for _, account := range AWSConfigurationManager.getAccounts() {
			if account != *AWSConfigurationManager.defaultAccountID {
				err := relatedAmi.setTagsForAccount(account, *ami.SourceAmiTags)

				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}

func (ami *Ami) copyToRegion(region string) (*Ami, error) {
	relatedAmi := ami.AmisPerRegion[region]

	log.Infof("Copying AMI to region %s", *relatedAmi.SourceRegion)
	copyImageInput := &ec2.CopyImageInput{
		Name:          ami.SourceAmiName,
		SourceRegion:  ami.SourceRegion,
		SourceImageId: ami.SourceAmiID,
	}
	ec2Service := getEC2ServiceForAccountAndRegion(*AWSConfigurationManager.defaultAccountID, *relatedAmi.SourceRegion)

	copyImageRequest := ec2Service.CopyImageRequest(copyImageInput)

	output, err := copyImageRequest.Send()

	if err != nil {
		return nil, err
	}
	log.Infof("New AMI ID: %s", *output.ImageId)
	relatedAmi.SourceAmiID = output.ImageId

	// Wait until AMI is `available`
	duration, _ := time.ParseDuration("5s")
	start := time.Now()

	for {
		relatedAmi.fetchMetadata()

		if relatedAmi.isAvailable() == true {
			log.Infof("AMI %s is available.", *relatedAmi.SourceAmiID)
			break
		}

		log.Infof("AMI %s is not available yet. Waiting %f seconds.", *relatedAmi.SourceAmiID, duration.Seconds())
		time.Sleep(duration)
	}

	elapsed := time.Since(start)
	log.Infof("AMI took %s to become available", elapsed)

	return relatedAmi, nil
}

func (ami *Ami) setOwners(owners []string) error {
	log.Infof("Setting owners to AMI %s", *ami.SourceAmiID)
	ec2Service := getEC2ServiceForAccountAndRegion(*AWSConfigurationManager.defaultAccountID, *ami.SourceRegion)

	modifyImageAttributeInput := &ec2.ModifyImageAttributeInput{
		ImageId: ami.SourceAmiID,
		LaunchPermission: &ec2.LaunchPermissionModifications{
			Add: createLaunchPermissionsForOwners(owners),
		},
	}

	modifyImageAttributeRequest := ec2Service.ModifyImageAttributeRequest(modifyImageAttributeInput)
	_, err := modifyImageAttributeRequest.Send()

	return err
}

func (ami *Ami) isAvailable() bool {
	if ami.AWSImage == nil {
		ami.fetchMetadata()
	}

	log.Debugf("Current AMI state is %s", ami.AWSImage.State)
	return ami.AWSImage.State == ec2.ImageStateAvailable
}

func (ami *Ami) setTagsForAccount(account string, tags []ec2.Tag) error {
	log.Infof("Setting tags for account %s", account)
	ec2service := getEC2ServiceForAccountAndRegion(account, *ami.SourceRegion)

	input := &ec2.CreateTagsInput{
		Resources: []string{*ami.SourceAmiID},
		Tags:      tags,
	}

	request := ec2service.CreateTagsRequest(input)
	_, err := request.Send()

	return err
}

func convertRegionSliceToAmi(slice []string) map[string]*Ami {
	amis := make(map[string]*Ami)

	for _, region := range slice {
		amis[region] = &Ami{SourceRegion: &region}
	}

	return amis
}

func createLaunchPermissionsForOwners(owners []string) []ec2.LaunchPermission {
	launchPermissions := make([]ec2.LaunchPermission, len(owners))
	for _, owner := range owners {
		launchPermissions = append(launchPermissions, ec2.LaunchPermission{
			UserId: aws.String(owner),
		})
	}
	return launchPermissions
}
