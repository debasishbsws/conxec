package exec

const (
	defaultDebuggerImage = "cgr.dev/chainguard/busybox:latest"
)

func New(opt []Option) error {
	exec := &Exec{}
	for _, o := range opt {
		if err := o(exec); err != nil {
			return err
		}
	}

	return nil
}

type Exec struct {
	Target  string   // target is the container id or name
	Command []string // cmd is the command to execute
	DbgImg  string   // dbgImg is the debugger image
}

type Option func(*Exec) error

func WithTarget(target string) Option {
	return func(opt *Exec) error {
		opt.Target = target
		return nil
	}
}

func WithCommand(command []string) Option {
	return func(opt *Exec) error {
		opt.Command = command
		return nil
	}
}

func WithDebuggerImage(dbgImg string) Option {
	if dbgImg == "" {
		dbgImg = defaultDebuggerImage
	}
	return func(opt *Exec) error {
		opt.DbgImg = dbgImg
		return nil
	}
}
