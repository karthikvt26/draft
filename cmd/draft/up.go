package main

import (
	"fmt"
	"io"
	"os"

	"github.com/Azure/draft/pkg/build"
	"github.com/Azure/draft/pkg/cmdline"
	"github.com/Azure/draft/pkg/draft"
	"github.com/Azure/draft/pkg/draft/manifest"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

const upDesc = `
This command archives the current directory into a tar archive and uploads it to
the draft server.

Adding the "watch" option to draft.toml makes draft automatically archive and
upload whenever local files are saved. Draft delays a couple seconds to ensure
that changes have stopped before uploading, but that can be altered by the
"watch_delay" option.
`

const (
	environmentEnvVar = "DRAFT_ENV"
	ignoreFileName    = ".draftignore"
)

type upCmd struct {
	client *draft.Client
	out    io.Writer
	src    string
}

func newUpCmd(out io.Writer) *cobra.Command {
	var (
		up                 = &upCmd{out: out}
		runningEnvironment string
	)

	cmd := &cobra.Command{
		Use:     "up [path]",
		Short:   "upload the current directory to the draft server for deployment",
		Long:    upDesc,
		PreRunE: setupConnection,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) > 0 {
				up.src = args[0]
			}
			up.client = ensureDraftClient(up.client)
			if up.src == "" || up.src == "." {
				if up.src, err = os.Getwd(); err != nil {
					return err
				}
			}
			return up.run(runningEnvironment)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&runningEnvironment, "environment", "e", defaultDraftEnvironment(), "the environment (development, staging, qa, etc) that draft will run under")

	return cmd
}

func (u *upCmd) run(environment string) (err error) {
	var buildctx *build.Context
	if buildctx, err = build.LoadWithEnv(u.src, environment); err != nil {
		return fmt.Errorf("failed loading build context with env %q: %v", environment, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err = u.client.Up(ctx, buildctx); err != nil {
			err = fmt.Errorf("there was an error running 'draft up': %v", err)
		}
		cancel()
	}()
	cmdline.Display(ctx, buildctx.Env.Name, u.client.Results())
	return nil
}

func defaultDraftEnvironment() string {
	env := os.Getenv(environmentEnvVar)
	if env == "" {
		env = manifest.DefaultEnvironmentName
	}
	return env
}
