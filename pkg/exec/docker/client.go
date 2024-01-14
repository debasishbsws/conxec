package docker

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/debasishbsws/conxec/pkg/exec"
	"github.com/debasishbsws/conxec/pkg/iocli"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/moby/moby/pkg/jsonmessage"
	"github.com/moby/moby/pkg/stdcopy"
)

type DockerClient struct {
	client        *client.Client
	out           *streams.Out
	targetInspect *types.ContainerJSON
}

func NewClient(ctx context.Context, opts *exec.ExecOptions, clistream *iocli.CliStream) (*DockerClient, error) {
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
		out:    clistream.AuxStream(),
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

func (c *DockerClient) AttachContainer(ctx context.Context, containerID string, tty, stdin bool, cliStream *iocli.CliStream) error {
	resp, err := c.client.ContainerAttach(ctx, containerID, types.ContainerAttachOptions{
		Stream: true,
		Stdin:  stdin,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return fmt.Errorf("failed to attach container: %w", err)
	}
	log.Printf("Attached container: %q\n", containerID)
	defer resp.Close()

	var cin io.ReadCloser
	if stdin {
		cin = cliStream.InputStream()
	}

	var cout io.Writer = cliStream.OutputStream()
	var cerr io.Writer = cliStream.ErrorStream()
	if tty {
		cerr = cliStream.OutputStream()
	}

	go func() {
		s := ioStreamer{
			streams:      cliStream,
			inputStream:  cin,
			outputStream: cout,
			errorStream:  cerr,
			resp:         resp,
			tty:          tty,
			stdin:        stdin,
		}

		if err := s.stream(ctx); err != nil {
			log.Printf("ioStreamer.stream() failed: %s", err)
		}
	}()

	if err := c.client.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("cannot start debugger container: %w", err)
	}

	if tty && cliStream.OutputStream().IsTerminal() {
		iocli.StartResizing(ctx, cliStream.OutputStream(), c.client, containerID)
	}

	statusCh, errCh := c.client.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("waiting debugger container failed: %w", err)
		}
	case <-statusCh:
	}

	return nil

}

type ioStreamer struct {
	streams *iocli.CliStream

	inputStream  io.ReadCloser
	outputStream io.Writer
	errorStream  io.Writer

	resp types.HijackedResponse

	stdin bool
	tty   bool
}

func (s *ioStreamer) stream(ctx context.Context) error {
	if s.tty {
		s.streams.InputStream().SetRawTerminal()
		s.streams.OutputStream().SetRawTerminal()
		defer func() {
			s.streams.InputStream().RestoreTerminal()
			s.streams.OutputStream().RestoreTerminal()
		}()
	}

	inDone := make(chan error)
	go func() {
		if s.stdin {
			if _, err := io.Copy(s.resp.Conn, s.inputStream); err != nil {
				log.Printf("Error forwarding stdin: %s", err)
			}
		}
		close(inDone)
	}()

	outDone := make(chan error)
	go func() {
		var err error
		if s.tty {
			_, err = io.Copy(s.outputStream, s.resp.Reader)
		} else {
			_, err = stdcopy.StdCopy(s.outputStream, s.errorStream, s.resp.Reader)
		}
		if err != nil {
			log.Printf("Error forwarding stdout/stderr: %s", err)
		}
		close(outDone)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-inDone:
		<-outDone
		return nil
	case <-outDone:
		return nil
	}
}
