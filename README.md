# AWS Ami Manager

Aws-ami-manager offers a simple way to perform copy, remove and cleanup operations on your AMI's. 

## Usage

This application uses the typical ways of authenticating with AWS.

### Copy
```
./aws-ami-manager \
copy \
--amiID=ami-0e94877fc6310ea8b \
--regions=eu-west-1,eu-central-1 \
--accounts=123456789,987654321
```

Make sure the accounts you want to copy are accessible through an Assume Role. 

### Remove

### Cleanup

## Licence

Apache License, version 2.0
