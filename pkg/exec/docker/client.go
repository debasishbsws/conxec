package docker

import (
	"context"
	"fmt"

	"github.com/debasishbsws/conxec/pkg/exec"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
)

type DockerClient struct {
	client        *client.Client
	out           *streams.Out
	targetInspect *types.ContainerJSON
}

func NewClient(ctx context.Context, opts *exec.ExecOptions) (*DockerClient, error) {
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

func (c *DockerClient) inspect(ctx context.Context, container string) error {
	if c.targetInspect != nil {
		conInspect, err := c.client.ContainerInspect(ctx, container)
		if err != nil {
			return fmt.Errorf("Failed to inspect container: %w", err)
		}
		c.targetInspect = &conInspect
	}
	return nil
}

func (c *DockerClient) IsContainerRunning(ctx context.Context, container string) (bool, error) {
	if err := c.inspect(ctx, container); err != nil {
		return false, err
	}
	if c.targetInspect.State == nil {
		return false, nil
	}
	return c.targetInspect.State.Running, nil
}

func (c *DockerClient) GetContainerUserId(ctx context.Context, container string) (string, error) {
	if err := c.inspect(ctx, container); err != nil {
		return "", err
	}
	if c.targetInspect.State == nil {
		return "", nil
	}
	return c.targetInspect.Config.User, nil
}

func (c *DockerClient) pullImage(ctx context.Context, ref string, platform string) error {
	resp, err := c.client.ImagePull(ctx, ref, types.ImagePullOptions{
		Platform: platform,
	})
	defer resp.Close()
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	return jsonmessage.DisplayJSONMessagesToStream(resp, c.out, nil)
}

func (c *DockerClient) PullTargetImage(ctx context.Context, image string, platform string) error {
	if platform == "" {
		if c.targetInspect == nil {
			return fmt.Errorf("platform is not specified use --runtime to specify")
		}
		platform = c.targetInspect.Platform
	}
	return c.pullImage(ctx, image, platform)
}
