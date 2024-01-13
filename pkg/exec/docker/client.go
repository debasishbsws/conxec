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

func (c *DockerClient) GetContainerInfo(ctx context.Context, container string) (*exec.ContainerInspectInfo, error) {
	conInspect, err := c.client.ContainerInspect(ctx, container)
	if err != nil {
		return nil, fmt.Errorf("Failed to inspect container: %w", err)
	}
	info := &exec.ContainerInspectInfo{
		ID:            conInspect.ID,
		Isrunning:     conInspect.State.Running,
		IsPrivileged:  conInspect.HostConfig.Privileged,
		IsPidModeHost: conInspect.HostConfig.PidMode.IsHost(),
		Pid:           conInspect.State.Pid,
		User:          conInspect.Config.User,
		Platform:      conInspect.Platform,
	}
	c.targetInspect = &conInspect
	return info, nil
}

func (c *DockerClient) PullImage(ctx context.Context, image string, platform string) error {
	resp, err := c.client.ImagePull(ctx, image, types.ImagePullOptions{
		Platform: platform,
	})
	defer resp.Close()
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	return jsonmessage.DisplayJSONMessagesToStream(resp, c.out, nil)
}

func (c *DockerClient) CreateContainer(ctx context.Context, targetInspect *exec.ContainerInspectInfo,
	image, entrypoint, user, containerName string,
	tty, stdin bool,
) (string, error) {
	resp, err := c.client.ContainerCreate(ctx, &container.Config{
		Image:        image,
		Entrypoint:   []string{"sh"},
		Cmd:          []string{"-c", entrypoint},
		User:         user, //now just use image default user
		Tty:          tty,
		OpenStdin:    stdin,
		AttachStdin:  stdin,
		AttachStdout: true,
		AttachStderr: true,
	},
		&container.HostConfig{
			Privileged: targetInspect.IsPrivileged,
			CapAdd:     c.targetInspect.HostConfig.CapAdd,
			CapDrop:    c.targetInspect.HostConfig.CapDrop,

			AutoRemove:  true, // remove the container when it exits TODO: make it configurable '--rm' flag
			PidMode:     container.PidMode("container:" + targetInspect.ID),
			NetworkMode: container.NetworkMode("container:" + targetInspect.ID),
		},
		&network.NetworkingConfig{
			EndpointsConfig: c.targetInspect.NetworkSettings.Networks,
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
