package aws

import (
	"errors"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go/aws"
)

type CredentialsProvider struct {
	*sts.Credentials
}

func (s CredentialsProvider) Retrieve() (awsv2.Credentials, error) {

	if s.Credentials == nil {
		return awsv2.Credentials{}, errors.New("sts credentials are nil")
	}

	return awsv2.Credentials{
		AccessKeyID:     aws.StringValue(s.AccessKeyId),
		SecretAccessKey: aws.StringValue(s.SecretAccessKey),
		SessionToken:    aws.StringValue(s.SessionToken),
		Expires:         aws.TimeValue(s.Expiration),
	}, nil
}
