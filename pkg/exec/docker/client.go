package docker

import (
	"context"
	"fmt"

	"github.com/debasishbsws/conxec/pkg/exec"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
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
	dockerClient, err := client.NewClientWithOpts(dockerOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return &DockerClient{
		client: dockerClient,
	}, nil
}

func (c *DockerClient) inspect(ctx context.Context, container string) error {
	if c.targetInspect == nil {
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

func pullImage(client *client.Client, out *streams.Out, ctx context.Context, ref string, platform string) error {
	resp, err := client.ImagePull(ctx, ref, types.ImagePullOptions{
		Platform: platform,
	})
	defer resp.Close()
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	return jsonmessage.DisplayJSONMessagesToStream(resp, out, nil)
}

func (c *DockerClient) PullTargetImage(ctx context.Context, image string, platform string) error {
	if platform == "" {
		if c.targetInspect == nil {
			return fmt.Errorf("platform is not specified use --runtime to specify")
		}
		platform = c.targetInspect.Platform
	}
	return pullImage(c.client, c.out, ctx, image, platform)
}

func createContainer(client *client.Client, ctx context.Context, targetInspect *types.ContainerJSON,
	image, entrypoint, user, containerName string,
	tty, stdin bool,
) (string, error) {
	resp, err := client.ContainerCreate(ctx, &container.Config{
		Image:      image,
		Entrypoint: []string{"sh"},
		Cmd:        []string{"-c", entrypoint},
		// User:         user, now just use image default user
		Tty:          tty,
		OpenStdin:    stdin,
		AttachStdin:  stdin,
		AttachStdout: true,
		AttachStderr: true,
	},
		&container.HostConfig{
			Privileged: targetInspect.HostConfig.Privileged,
			CapAdd:     targetInspect.HostConfig.CapAdd,
			CapDrop:    targetInspect.HostConfig.CapDrop,

			AutoRemove:  true, // remove the container when it exits TODO: make it configurable '--rm' flag
			PidMode:     container.PidMode("container:" + targetInspect.ID),
			NetworkMode: container.NetworkMode("container:" + targetInspect.ID),
		},
		&network.NetworkingConfig{
			EndpointsConfig: targetInspect.NetworkSettings.Networks,
		},
		nil,
		containerName,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	fmt.Printf("Created container: %q\n", resp.ID)
	return resp.ID, nil
}

func (c *DockerClient) CreateDebuggerContainer(ctx context.Context, opts *exec.ExecOptions, entrypoint string) error {
	_, err := createContainer(c.client, ctx, c.targetInspect, opts.DbgImg, entrypoint,
		opts.UserN, opts.Target, opts.Tty, opts.Interactive)
	if err != nil {
		return err
	}

	return nil
}
