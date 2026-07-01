package mapper

import (
	"context"
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var subnetIDPattern = regexp.MustCompile(`^subnet-[0-9a-f]{8,17}$`)

// MapSubnetToCloudConfig queries AWS to get VPC and availability zone
// from the first subnet ID in the subnet_ids array
func MapSubnetToCloudConfig(
	ctx context.Context,
	ec2Client *ec2.Client,
	subnetIDs []string,
) (*CloudProviderConfig, error) {

	// Validate input
	if len(subnetIDs) == 0 {
		return nil, fmt.Errorf("subnet_ids is required and cannot be empty")
	}

	subnetID := subnetIDs[0]

	// Validate subnet ID format
	if !isValidSubnetID(subnetID) {
		return nil, fmt.Errorf("invalid subnet ID format: %s", subnetID)
	}

	// Query AWS for subnet details
	input := &ec2.DescribeSubnetsInput{
		SubnetIds: []string{subnetID},
	}

	result, err := ec2Client.DescribeSubnets(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe subnet %s: %w", subnetID, err)
	}

	if len(result.Subnets) == 0 {
		return nil, fmt.Errorf("subnet %s not found", subnetID)
	}

	subnet := result.Subnets[0]

	// Validate required fields are present
	if subnet.VpcId == nil || *subnet.VpcId == "" {
		return nil, fmt.Errorf("subnet %s has no VPC ID", subnetID)
	}
	if subnet.AvailabilityZone == nil || *subnet.AvailabilityZone == "" {
		return nil, fmt.Errorf("subnet %s has no availability zone", subnetID)
	}

	return &CloudProviderConfig{
		SubnetID: subnetID,
		VpcID:    *subnet.VpcId,
		Zone:     *subnet.AvailabilityZone,
	}, nil
}

// isValidSubnetID validates AWS subnet ID format
func isValidSubnetID(id string) bool {
	return subnetIDPattern.MatchString(id)
}
