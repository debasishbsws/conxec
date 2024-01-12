package exec

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

const (
	defaultDebuggerImage = "ghcr.io/debasishbsws/conxec-debugger:latest"
)

func New(opt []Option) (*ExecOptions, error) {
	exec := &ExecOptions{}
	for _, o := range opt {
		if err := o(exec); err != nil {
			return nil, err
		}
	}

	return exec, nil
}

type ExecOptions struct {
	Target      string   // target is the container id or name
	Command     []string // cmd is the command to execute
	DbgImg      string   // dbgImg is the debugger image
	Runtime     string   // runtime is the docker runtime
	Schema      string   // schema is the schema of the target
	UserN       string   // user-name is the user name of the target
	UserID      string   // user-id is the user id of the target
	GroupN      string   // group-name is the group name of the target
	GroupID     string   // group-id is the group id of the target
	Tty         bool     // tty is the flag to enable tty
	Interactive bool     // interactive is the flag to enable interactive

}

type Option func(*ExecOptions) error

func WithTarget(target string) Option {
	return func(opt *ExecOptions) error {
		opt.Target = target
		return nil
	}
}

func WithCommand(command []string) Option {
	return func(opt *ExecOptions) error {
		opt.Command = command
		return nil
	}
}

func WithDebuggerImage(dbgImg string) Option {
	if dbgImg == "" {
		dbgImg = defaultDebuggerImage
	}
	return func(opt *ExecOptions) error {
		opt.DbgImg = dbgImg
		return nil
	}
}

func WithRuntime(runtime string) Option {
	return func(opt *ExecOptions) error {
		opt.Runtime = runtime
		return nil
	}
}

func WithUser(user string) Option {
	reg, err := regexp.Compile(`^[a-z_][a-z0-9_-]*:[0-9]+::[a-z_][a-z0-9_-]*:[0-9]+$`)
	if err != nil {
		panic(err)
	}
	return func(opt *ExecOptions) error {
		if !reg.MatchString(user) {
			return fmt.Errorf("invalid user format: %q. Use: <user-name>:<user-id>::<group-name>:<group-id>", user)
		}
		usergroupInfo := strings.Split(user, "::")
		opt.UserN = strings.Split(usergroupInfo[0], ":")[0]
		opt.UserID = strings.Split(usergroupInfo[0], ":")[1]
		opt.GroupN = strings.Split(usergroupInfo[1], ":")[0]
		opt.GroupID = strings.Split(usergroupInfo[1], ":")[1]
		return nil
	}
}

type DebuggerClient interface {
	IsContainerRunning(context.Context, string) (bool, error)
	GetContainerUserId(context.Context, string) (string, error)
	PullTargetImage(context.Context, string, string) error
	CreateDebuggerContainer(context.Context, *ExecOptions, string) error
}

func RunDebugger(ctx context.Context, client DebuggerClient, opts *ExecOptions) error {
	if isRunning, err := client.IsContainerRunning(ctx, opts.Target); err != nil {
		return err
	} else if !isRunning {
		return fmt.Errorf("target container: %q is not running", opts.Target)
	}

	if targetContainerUserId, err := client.GetContainerUserId(ctx, opts.Target); err != nil {
		return err
	} else if targetContainerUserId != "root" && targetContainerUserId != "0" {
		// TODO: support non-root user
		/* Look for the user and group in the target container by look at /proc/1/status (somewhere around there)
		uid, gid, err := getUidGid(target)
		if not found send error of specifying user and group
		*/
		return fmt.Errorf("User of target container: %q is not root user -u to specify user and group", opts.Target)
	}

	fmt.Printf("Pulling debugger image: %q\n", opts.DbgImg)
	if err := client.PullTargetImage(ctx, opts.DbgImg, opts.Runtime); err != nil {
		return fmt.Errorf("failed to pull debugger image: %w", err)
	}

	return nil
}
