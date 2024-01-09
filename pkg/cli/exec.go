package cli

import (
	"log"

	"github.com/debasishbsws/conxec/pkg/exec"
	"github.com/spf13/cobra"
)

func ExecCmd() *cobra.Command {
	var target string
	var command []string
	var dbgImage string

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
			}
			if err := exec.New(opt); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dbgImage, "dbg-img", "", "debugger image to use (e.g: cgr.dev/chainguard/busybox:latest or busybox:musl)")

	return cmd
}
