package aws

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	log "github.com/sirupsen/logrus"
)

var (
	ConfigManager *ConfigurationManager
	ec2Services   = make(map[string]map[string]*ec2.Client)
)

func getEC2ServiceForAccountAndRegion(account string, region string) *ec2.Client {
	log.Debugf("getEC2ServiceForAccountAndRegion: account %s, region %s", account, region)
	if ec2Services[account] == nil {
		ec2Services[account] = make(map[string]*ec2.Client)
	}

	if ec2Services[account][region] == nil {
		ec2Services[account][region] = ec2.New(ConfigManager.getConfigurationForAccountAndRegion(account, region))
	}
	return ec2Services[account][region]
}

type Ami struct {
	SourceAmiID   string
	SourceRegion  string
	SourceAmiName string
	SourceAmiTags *[]ec2.Tag
	AWSImage      *ec2.Image

	AmisPerRegion map[string]*Ami
}

func NewAmi(sourceAmiID string) *Ami {
	return &Ami{
		SourceAmiID: sourceAmiID,
	}
}

func NewAmiWithRegions(sourceAmiID string, sourceRegion string, regions []string) *Ami {
	ami := &Ami{
		SourceAmiID:   sourceAmiID,
		SourceRegion:  sourceRegion,
		AmisPerRegion: convertRegionSliceToAmi(regions),
	}

	return ami
}

func (ami *Ami) fetchMetadata() error {
	log.Debug("Fetching metadata about the AMI")
	ec2svc := getEC2ServiceForAccountAndRegion(*ConfigManager.defaultAccountID, ami.SourceRegion)

	var amiList []string
	amiList = append(amiList, ami.SourceAmiID)

	describeImagesInput := ec2.DescribeImagesInput{
		ImageIds: amiList,
	}
	request := ec2svc.DescribeImagesRequest(&describeImagesInput)
	result, err := request.Send(context.Background())

	if err != nil {
		return err
	}

	images := result.Images

	if len(images) < 1 {
		return errors.New(fmt.Sprintf("no ami found with id %s", ami.SourceAmiID))
	}

	ami.AWSImage = &images[0]

	// With newly copied AMI's name and tags can still be empty, causing nil references
	// And only set name and tags when not yet already set
	if ami.SourceAmiName == "" && ami.AWSImage.Name != nil {
		ami.SourceAmiName = *ami.AWSImage.Name
		log.Debugf("AMI name: %s", ami.SourceAmiName)
	}

	if ami.SourceAmiTags == nil && images[0].Tags != nil {
		ami.SourceAmiTags = &images[0].Tags
		log.Debugf("AMI tags: %s", ami.SourceAmiTags)
	}

	return nil
}

func (ami *Ami) Copy() {
	// Fetch name and tags for the source AMI
	err := ami.fetchMetadata()

	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup

	// in this loop region is the key
	for region := range ami.AmisPerRegion {
		log.Debugf("Region is %s", region)

		wg.Add(1)
		go func(amiF *Ami, region string) {
			var (
				relatedAmi *Ami
				err        error
			)

			// We obviously don't have to copy the AMI to a region where it already exists
			if region != amiF.SourceRegion {
				log.Debug("Starting copying")

				relatedAmi, err = amiF.copyToRegion(region)

				if err != nil {
					log.Fatal(err)
				}

				err = relatedAmi.setOwners(ConfigManager.accounts)

				if err != nil {
					log.Fatal(err)
				}
			} else {
				relatedAmi = amiF
			}

			for _, account := range ConfigManager.getAccounts() {
				// the original AMI already has the tags
				if account != *ConfigManager.defaultAccountID {
					err := relatedAmi.setTagsForAccount(account, *amiF.SourceAmiTags)

					if err != nil {
						log.Fatal(err)
					}
				}
			}

			wg.Done()
		}(ami, region)
	}

	wg.Wait()
}

func (ami *Ami) copyToRegion(region string) (*Ami, error) {
	relatedAmi := ami.AmisPerRegion[region]

	log.Infof("Copying AMI to region %s", relatedAmi.SourceRegion)
	copyImageInput := &ec2.CopyImageInput{
		Name:          aws.String(ami.SourceAmiName),
		SourceRegion:  aws.String(ami.SourceRegion),
		SourceImageId: aws.String(ami.SourceAmiID),
	}
	ec2Service := getEC2ServiceForAccountAndRegion(*ConfigManager.defaultAccountID, relatedAmi.SourceRegion)

	copyImageRequest := ec2Service.CopyImageRequest(copyImageInput)
	output, err := copyImageRequest.Send(context.Background())

	if err != nil {
		log.Debug(err)
		return nil, err
	}
	log.Infof("New AMI ID: %s", *output.ImageId)
	relatedAmi.SourceAmiID = *output.ImageId

	// Wait until AMI is `available`
	duration, _ := time.ParseDuration("5s")
	start := time.Now()

	for {
		_ = relatedAmi.fetchMetadata()

		if relatedAmi.isAvailable() == true {
			log.Infof("AMI %s is available.", relatedAmi.SourceAmiID)
			break
		}

		log.Infof("AMI %s is not available yet. Waiting %f seconds.", relatedAmi.SourceAmiID, duration.Seconds())
		time.Sleep(duration)
	}

	elapsed := time.Since(start)
	log.Infof("AMI took %s to become available", elapsed)

	return relatedAmi, nil
}

func (ami *Ami) setOwners(owners []string) error {
	log.Infof("Setting owners to AMI %s", ami.SourceAmiID)
	log.Debugf("Fetching EC2 service for region: %s", ami.SourceRegion)
	ec2Service := getEC2ServiceForAccountAndRegion(*ConfigManager.defaultAccountID, ami.SourceRegion)

	modifyImageAttributeInput := &ec2.ModifyImageAttributeInput{
		ImageId: aws.String(ami.SourceAmiID),
		LaunchPermission: &ec2.LaunchPermissionModifications{
			Add: createLaunchPermissionsForOwners(owners),
		},
	}

	modifyImageAttributeRequest := ec2Service.ModifyImageAttributeRequest(modifyImageAttributeInput)
	_, err := modifyImageAttributeRequest.Send(context.Background())

	log.Debugf("Owners set for AMI %s", ami.SourceAmiID)

	return err
}

func (ami *Ami) isAvailable() bool {
	if ami.AWSImage == nil {
		_ = ami.fetchMetadata()
	}

	log.Debugf("Current AMI state is %s", ami.AWSImage.State)
	return ami.AWSImage.State == ec2.ImageStateAvailable
}

func (ami *Ami) setTagsForAccount(account string, tags []ec2.Tag) error {
	log.Infof("Setting tags for account %s", account)
	log.Debug(ami)
	ec2service := getEC2ServiceForAccountAndRegion(account, ami.SourceRegion)

	input := &ec2.CreateTagsInput{
		Resources: []string{ami.SourceAmiID},
		Tags:      tags,
	}

	request := ec2service.CreateTagsRequest(input)
	_, err := request.Send(context.Background())

	return err
}

func convertRegionSliceToAmi(slice []string) map[string]*Ami {
	amis := make(map[string]*Ami)

	for _, region := range slice {
		ami := &Ami{SourceRegion: region}
		amis[region] = ami
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

func (ami *Ami) Cleanup(regions []string, tagsToMatch []string, versionsToKeep int) error {
	// describe ami
	err := ami.fetchMetadata()

	if err != nil {
		log.Fatal(err)
	}

	// convert Tag slice to map for easier lookup
	tags := convertTagSliceToMap(ami.AWSImage.Tags)

	// get the tags we need to match with
	var matchedTags []ec2.Tag
	for _, tagToMatch := range tagsToMatch {
		if match, ok := tags[tagToMatch]; ok {
			matchedTags = append(matchedTags, match)
		}
	}

	for _, region := range regions {
		ec2svc := getEC2ServiceForAccountAndRegion(*ConfigManager.defaultAccountID, region)

		describeImagesInput := ec2.DescribeImagesInput{
			Filters: convertTagSliceToFilter(matchedTags),
		}
		request := ec2svc.DescribeImagesRequest(&describeImagesInput)
		result, err := request.Send(context.Background())

		if err != nil {
			log.Fatal(err)
		}

		images := result.Images

		// sort the returned images
		sort.Slice(images, func(i, j int) bool {
			firstDate, err := time.Parse(time.RFC3339, *images[i].CreationDate)

			if err != nil {
				log.Fatal(err)
			}

			secondDate, err := time.Parse(time.RFC3339, *images[j].CreationDate)

			if err != nil {
				log.Fatal(err)
			}

			return firstDate.After(secondDate)
		})

		var i = 0
		for _, image := range images {
			// keep the first (i.e. most recent) AMI
			if i >= versionsToKeep {
				log.Debugf("Deleting image %s", *image.ImageId)
				err = removeAwsAmi(&image, ec2svc)
				log.Infof("Image %s deleted", *image.ImageId)

				if err != nil {
					log.Fatal(err)
				}
			}
			i++
		}

	}
	return err
}

func (ami *Ami) RemoveAmi() error {
	// describe ami
	err := ami.fetchMetadata()

	if err != nil {
		log.Fatal(err)
	}

	ec2Service := getEC2ServiceForAccountAndRegion(*ConfigManager.defaultAccountID, ConfigManager.GetDefaultRegion())
	err = removeAwsAmi(ami.AWSImage, ec2Service)

	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func removeAwsAmi(image *ec2.Image, ec2Service *ec2.Client) error {
	// deregister ami
	deregisterAmiInput := &ec2.DeregisterImageInput{
		ImageId: image.ImageId,
	}

	_, err := ec2Service.DeregisterImageRequest(deregisterAmiInput).Send(context.Background())

	if err != nil {
		log.Fatal(err)
	}

	log.Debug("AMI is de-registered.")

	// delete snapshot
	for _, mapping := range image.BlockDeviceMappings {
		deleteSnapshotInput := &ec2.DeleteSnapshotInput{
			SnapshotId: mapping.Ebs.SnapshotId,
		}

		_, err := ec2Service.DeleteSnapshotRequest(deleteSnapshotInput).Send(context.Background())

		if err != nil {
			return err
		}
	}

	log.Debug("Snapshots have been deleted.")

	return nil
}

func convertTagSliceToMap(tagSlice []ec2.Tag) map[string]ec2.Tag {
	tagMap := make(map[string]ec2.Tag)
	if len(tagSlice) > 0 {
		for _, tag := range tagSlice {
			tagMap[*tag.Key] = tag
		}
	}
	return tagMap
}

func convertTagSliceToFilter(tags []ec2.Tag) []ec2.Filter {
	tagsFilter := make([]ec2.Filter, len(tags))

	for key, tag := range tags {
		tagsFilter[key] = convertTagToFilter(tag)
	}

	return tagsFilter
}

func convertTagToFilter(tag ec2.Tag) ec2.Filter {
	name := "tag:" + *tag.Key
	values := make([]string, 1)

	values = append(values, *tag.Value)

	return ec2.Filter{
		Name:   &name,
		Values: values,
	}
}
