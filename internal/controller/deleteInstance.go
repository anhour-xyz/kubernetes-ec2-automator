package controller

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"time"
)

func deleteEC2Instance(ctx context.Context,
	ec2Client *ec2.Client, instanceID string) error {
	if instanceID == "" {
		return nil
	}
	_, err := ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})

	if err != nil {
		return fmt.Errorf("terminate EC2 instance %s:%w", instanceID, err)
	}

	waiter := ec2.NewInstanceTerminatedWaiter(ec2Client)
	err = waiter.Wait(
		ctx,
		&ec2.DescribeInstancesInput{
			InstanceIds: []string{instanceID},
		}, 5*
			time.Minute,
	)

	if err != nil {
		return fmt.Errorf("wait for EC2 instance %s to terminate: %w", instanceID, err)
	}

	return nil
}
