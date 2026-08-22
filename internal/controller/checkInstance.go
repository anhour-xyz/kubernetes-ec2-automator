package controller

import (
	"context"
	"fmt"
)

func checkEC2InstanceExists(ctx context.Context, instanceID string) {
	fmt.Println("Checking instance ", instanceID)
}
