package exec

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/debasishbsws/conxec/pkg/iocli"
	"github.com/google/uuid"
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
	Target  string   // target is the container id or name
	Command []string // cmd is the command to execute
	DbgImg  string   // dbgImg is the debugger image
	Name    string   // name is the name of the container
	Runtime string   // runtime is the docker runtime
	Schema  string   // schema is the schema of the target
	UserN   string   // user-name is the user name of the target
	UserID  string   // user-id is the user id of the target
	GroupN  string   // group-name is the group name of the target
	GroupID string   // group-id is the group id of the target
	Tty     bool     // tty is the flag to enable tty
	Stdin   bool     // interactive is the flag to enable interactive

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

func WithName(name string) Option {
	return func(opt *ExecOptions) error {
		opt.Name = name
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

func WithTty(tty bool) Option {
	return func(opt *ExecOptions) error {
		opt.Tty = tty
		return nil
	}
}

func WithStdin(stdin bool) Option {
	return func(opt *ExecOptions) error {
		opt.Stdin = stdin
		return nil
	}
}

type DebuggerClient interface {
	// GetContainerInfo returns the container info
	GetContainerInfo(ctx context.Context, containerName string) (*ContainerInspectInfo, error)
	// Pull an iamge from the registry if not present
	PullImage(ctx context.Context, iamgeName string, patform string) error
	// Create a Container and return the container id
	CreateContainer(ctx context.Context, targetInspect *ContainerInspectInfo,
		image, entrypoint, user, containerName string,
		tty, stdin bool) (containerID string, err error)
	AttachContainer(ctx context.Context, containerID string, tty, stdin bool, cliStream *iocli.CliStream) error
}

func shellescape(args []string) []string {
	escaped := []string{}
	for _, a := range args {
		// check if the string has any special characters or escape charecters
		if strings.ContainsAny(a, " \t\n\r") {
			escaped = append(escaped, strconv.Quote(a))
		} else {
			escaped = append(escaped, a)
		}
	}
	return escaped
}

//go:embed conxec-entrypoint.templ
var entrypointTemplate string

func generateEntrypoint(runID string, targetPID int, cmd []string) string {
	entrypointTemplae := template.Must(template.New("entrypoint").Parse(entrypointTemplate))
	var command string
	if len(cmd) == 0 {
		command = "sh"
	} else {
		command = "sh -c '" + strings.Join(shellescape(cmd), " ") + "'"
	}
	log.Printf("cmd: %s\n", cmd)
	data := map[string]string{
		"ID":  runID,
		"PID": fmt.Sprintf("%d", targetPID),
		"CMD": command,
	}
	var entrypoint strings.Builder
	if err := entrypointTemplae.Execute(&entrypoint, data); err != nil {
		panic(err)
	}
	return entrypoint.String()
}

type ContainerInspectInfo struct {
	ID            string
	Isrunning     bool
	IsPrivileged  bool
	IsPidModeHost bool
	Pid           int
	User          string
	Platform      string
}

func RunDebugger(ctx context.Context, client DebuggerClient, opts *ExecOptions, cliStream *iocli.CliStream) error {
	targetContainerInfo, err := client.GetContainerInfo(ctx, opts.Target)
	if err != nil {
		return err
	}
	if !targetContainerInfo.Isrunning {
		return fmt.Errorf("target container: %q is not running", opts.Target)
	}

	if targetContainerInfo.User != "root" && targetContainerInfo.User != "0" && targetContainerInfo.User == "nonroot" {
		if opts.UserN == "" {
			return fmt.Errorf("User of target container: %q is nither root nor nonroot user -u to specify user and group", opts.Target)
		}
	}

	cliStream.PrintAux("Pulling debugger image: %q\n", opts.DbgImg)

	if err := client.PullImage(ctx, opts.DbgImg, opts.Runtime); err != nil {
		return fmt.Errorf("failed to pull debugger image: %w", err)
	}

	cliStream.PrintAux("Creating debugger container...\n")
	debID := getShortRandomID()
	if opts.Name == "" {
		opts.Name = fmt.Sprintf("conxec-debugger-%s", debID)
	}
	if opts.UserN != "" {
		// pass do nothing for now, always run as root.
	}
	targetPID := 1
	if targetContainerInfo.IsPidModeHost {
		targetPID = targetContainerInfo.Pid
	}
	user := "root:root"
	changeuserScript := ""
	if targetContainerInfo.User == "nonroot" {
		user = "nonroot:nonroot"
	} else if opts.UserN != "" {
		// rus as root and change the user inside the container by entrypoint
		changeuserScript = fmt.Sprintf("addgroup -g %s %s \nadduser -D -u %s -G %s %s\n", opts.GroupID, opts.GroupN, opts.UserID, opts.GroupN, opts.UserN)
	}
	entrypointStr := generateEntrypoint(debID, targetPID, opts.Command)

	// There is a issue can't add user addgroup: number 65532 is not in 0..60000 range; adduser: number 65532 is not in 0..60000 range
	_ = changeuserScript
	// if changeuserScript != "" {
	// 	entrypointStr = changeuserScript + entrypointStr
	// }
	// create debugger container
	debugerID, err := client.CreateContainer(ctx, targetContainerInfo, opts.DbgImg, entrypointStr, user, opts.Name, opts.Tty, opts.Stdin)
	if err != nil {
		return fmt.Errorf("failed to create debugger container: %w", err)
	}
	cliStream.PrintAux("Debugger container created: %v\n>>\n", debugerID)
	client.AttachContainer(ctx, debugerID, opts.Tty, opts.Stdin, cliStream)

	return nil
}

// Util functions
func getShortRandomID() string {
	return strings.Split(uuid.NewString(), "-")[0]
}
