package docker

import (
	"context"
	"fmt"

	"github.com/debasishbsws/conxec/pkg/exec"
	"github.com/docker/docker/client"
)

type DockerClient struct {
	client *client.Client
}

func NewClient(ctx context.Context, opts exec.ExecOptions) (*DockerClient, error) {
	dockerOpts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}
	if opts.Runtime != "" {
		dockerOpts = append(dockerOpts, client.WithHost(opts.Runtime))
	}
	client, err := client.NewClientWithOpts(dockerOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return &DockerClient{
		client: client,
	}, nil
}

func (c *DockerClient) IsContainerRunning(ctx context.Context, container string) (bool, error) {
	conInspect, err := c.client.ContainerInspect(ctx, container)
	if err != nil {
		return false, fmt.Errorf("Failed to inspect container: %w", err)
	}
	if conInspect.State == nil {
		return false, nil
	}
	return conInspect.State.Running, nil
}

func (c *DockerClient) GetContainerUserId(ctx context.Context, container string) (string, error) {
	conInspect, err := c.client.ContainerInspect(ctx, container)
	if err != nil {
		return "", fmt.Errorf("Failed to inspect container: %w", err)
	}
	if conInspect.State == nil {
		return "", nil
	}
	return conInspect.Config.User, nil
}

func (c *DockerClient) PullImage(ctx context.Context, image string) error {
	return nil
}
