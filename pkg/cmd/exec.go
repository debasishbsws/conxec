package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/debasishbsws/conxec/pkg/exec"
	"github.com/debasishbsws/conxec/pkg/exec/docker"
	"github.com/debasishbsws/conxec/pkg/iocli"
	"github.com/spf13/cobra"
)

const (
	schemaContainerd = "containerd://"
	schemaDocker     = "docker://"
)

func ExecCmd() *cobra.Command {
	var target string
	var command []string
	var dbgImage string
	var name string
	var userGroup string
	var runtime string
	var tty bool
	var interactive bool
	var mountDir string

	cmd := &cobra.Command{
		Use:   "exec [container-id/name] [command]",
		Short: "Execute a command in a running container. default is /bin/sh",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target = args[0]
			if len(args) > 1 {
				command = args[1:]
			} else {
				command = []string{"sh"}
			}
			aditionalPackages, err := cmd.Flags().GetStringSlice("application")
			if err != nil {
				return err
			}
			opt := []exec.Option{
				exec.WithTarget(target),
				exec.WithCommand(command),
				exec.WithDebuggerImage(dbgImage),
				exec.WithUser(userGroup),
				exec.WithName(name),
				exec.WithRuntime(runtime),
				exec.WithTty(tty),
				exec.WithStdin(interactive),
				exec.WithAditionalPackages(aditionalPackages),
				exec.WithMountDir(mountDir),
			}
			exec, err := exec.New(opt)
			if err != nil {
				return err
			}
			return ExecuteCmd(cmd.Context(), exec)
		},
	}

	cmd.Flags().StringVar(&dbgImage, "dbg-img", "",
		"debugger image to use (e.g: ghcr.io/debasishbsws/conxec-debugger:latest or busybox:musl)",
	)
	cmd.Flags().StringVarP(&name, "name", "n", "", "name of the container")
	cmd.Flags().StringVarP(&userGroup, "user", "u", "root:0::root:0",
		"user and group to use format: <user-name>:<user-id>::<group-name>:<group-id> (e.g: root:0::root:0)",
	)
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, `Keep the STDIN open (as in "docker exec -i")`)
	cmd.Flags().BoolVarP(&tty, "tty", "t", false, `Allocate a pseudo-TTY (as in "docker exec -t")`)
	cmd.Flags().StringVar(&runtime, "runtime", "",
		`Runtime address ("/var/run/docker.sock" | "/run/containerd/containerd.sock" | "https://<kube-api-addr>:8433/...)`,
	)
	cmd.Flags().StringSliceP("application", "a", []string{}, "additional application to install in the debugger image works only with root user")
	cmd.Flags().StringVarP(&mountDir, "mount", "m", "", "mount directory in the target container can be access by $MNTD")
	return cmd
}

func ExecuteCmd(ctx context.Context, execOpts *exec.ExecOptions) error {
	if sep := strings.Index(execOpts.Target, "://"); sep != -1 {
		execOpts.Schema = execOpts.Target[:sep+3]
		execOpts.Target = execOpts.Target[sep+3:]
	} else {
		execOpts.Schema = schemaDocker
	}

	clistream := iocli.NewCliStream(os.Stdin, os.Stdout, os.Stderr)

	switch execOpts.Schema {
	case schemaDocker:
		dockerClient, err := docker.NewClient(ctx, execOpts, clistream)
		if err != nil {
			return err
		}
		return exec.RunDebugger(ctx, dockerClient, execOpts, clistream)

	case schemaContainerd:
		return errors.New("coming soon...")

	default:
		return fmt.Errorf("unknown schema %q", execOpts.Schema)
	}
}
