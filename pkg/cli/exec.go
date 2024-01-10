package cli

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/debasishbsws/conxec/pkg/exec"
	"github.com/debasishbsws/conxec/pkg/exec/docker"
	"github.com/spf13/cobra"
)

const (
	schemaContainerd = "containerd://"
	schemaDocker     = "docker://"
	// schemaKubeCRI    = "cri://"
	// schemaKubeLong   = "kubernetes://"
	// schemaKubeShort  = "k8s://"
	// schemaNerdctl    = "nerdctl://"
)

func ExecCmd() *cobra.Command {
	var target string
	var command []string
	var dbgImage string
	var userGroup string

	cmd := &cobra.Command{
		Use:   "exec [container-id/name] [command]",
		Short: "Execute a command in a running container. default is /bin/sh",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target = args[0]
			if len(args) > 1 {
				command = args[1:]
			} else {
				command = []string{"/bin/sh"}
			}
			log.Printf("target: %s, cmd: %s", target, command)
			opt := []exec.Option{
				exec.WithTarget(target),
				exec.WithCommand(command),
				exec.WithDebuggerImage(dbgImage),
				exec.WithUser(userGroup),
			}
			exec, err := exec.New(opt)
			if err != nil {
				return err
			}
			return ExecuteCmd(cmd.Context(), exec)
		},
	}

	cmd.Flags().StringVar(&dbgImage, "dbg-img", "",
		"debugger image to use (e.g: cgr.dev/chainguard/busybox:latest or busybox:musl)",
	)
	cmd.Flags().StringVarP(&userGroup, "user", "-u", "root:0::root:0",
		"user and group to use format: <user-name>:<user-id>::<group-name>:<group-id> (e.g: root:0::root:0)",
	)

	return cmd
}

func ExecuteCmd(ctx context.Context, execOpts *exec.ExecOptions) error {
	if sep := strings.Index(execOpts.Target, "://"); sep != 1 {
		execOpts.Schema = execOpts.Target[:sep+3]
		execOpts.Target = execOpts.Target[sep+3:]
	} else {
		execOpts.Schema = schemaDocker
	}

	switch execOpts.Schema {
	case schemaDocker:
		dockerClient := &docker.DockerClient{}
		return exec.RunDebugger(ctx, dockerClient, execOpts)

	case schemaContainerd:
		return errors.New("coming soon...")

	default:
		return fmt.Errorf("unknown schema %q", execOpts.Schema)
	}
}
