package cmd

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
	var name string
	var userGroup string
	var runtime string
	var tty bool
	var interactive bool

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
				exec.WithName(name),
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

	return cmd
}

func ExecuteCmd(ctx context.Context, execOpts *exec.ExecOptions) error {
	if sep := strings.Index(execOpts.Target, "://"); sep != -1 {
		execOpts.Schema = execOpts.Target[:sep+3]
		execOpts.Target = execOpts.Target[sep+3:]
	} else {
		execOpts.Schema = schemaDocker
	}

	switch execOpts.Schema {
	case schemaDocker:
		dockerClient, err := docker.NewClient(ctx, execOpts)
		if err != nil {
			return err
		}
		return exec.RunDebugger(ctx, dockerClient, execOpts)

	case schemaContainerd:
		return errors.New("coming soon...")

	default:
		return fmt.Errorf("unknown schema %q", execOpts.Schema)
	}
}

// Still need to build a image of my own to use it as a debugger
// docker run --rm -ti --name debuger --pid container:pig --network container:pig busybox sh -c '
// ln -fs /proc/$$/root/bin/ /proc/1/root/.conxec
// cat > /.conxec-entrypoint.sh <<EOF
// #!/bin/sh
// export PATH=$PATH:/.conxec

// chroot /proc/1/root /.conxec/sh
// EOF
// sh /.conxec-entrypoint.sh'
