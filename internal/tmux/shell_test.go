package tmux

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os/exec"
	"reflect"
	"sync"
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/log/logtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give NewSessionRequest
		want []string
	}{
		{
			desc: "empty",
			want: []string{"new-session"},
		},
		{
			desc: "name",
			give: NewSessionRequest{Name: "foo"},
			want: []string{"new-session", "-s", "foo"},
		},
		{
			desc: "format",
			give: NewSessionRequest{Format: "#{window_id}"},
			want: []string{"new-session", "-P", "-F", "#{window_id}"},
		},
		{
			desc: "width",
			give: NewSessionRequest{Width: 42},
			want: []string{"new-session", "-x", "42"},
		},
		{
			desc: "height",
			give: NewSessionRequest{Height: 42},
			want: []string{"new-session", "-y", "42"},
		},
		{
			desc: "detached",
			give: NewSessionRequest{Detached: true},
			want: []string{"new-session", "-d"},
		},
		{
			desc: "env",
			give: NewSessionRequest{
				Env:     []string{"FOO=bar", "BAZ=qux"},
				Command: []string{"/bin/bash"},
			},
			want: []string{"new-session", "/usr/bin/env", "FOO=bar", "BAZ=qux", "/bin/bash"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			blob := make([]byte, 10)
			randRead(t, blob)

			r := newFakeRunner(t)
			r.ExpectOutput("tmux", tt.want...).Stdout(blob)

			driver := ShellDriver{
				run: r.Runner(),
				log: logtest.NewLogger(t),
			}
			got, err := driver.NewSession(tt.give)
			require.NoError(t, err)
			assert.Equal(t, blob, got)
		})
	}
}

func TestCapturePaneArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give CapturePaneRequest
		want []string
	}{
		{
			desc: "empty",
			want: []string{"capture-pane", "-p", "-J"},
		},
		{
			desc: "pane",
			give: CapturePaneRequest{Pane: "%42"},
			want: []string{"capture-pane", "-p", "-J", "-t", "%42"},
		},
		{
			desc: "start line",
			give: CapturePaneRequest{StartLine: 42},
			want: []string{"capture-pane", "-p", "-J", "-S", "42"},
		},
		{
			desc: "end line",
			give: CapturePaneRequest{EndLine: 42},
			want: []string{"capture-pane", "-p", "-J", "-E", "42"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			blob := make([]byte, 10)
			randRead(t, blob)

			r := newFakeRunner(t)
			r.ExpectOutput("tmux", tt.want...).Stdout(blob)

			driver := ShellDriver{
				run: r.Runner(),
				log: logtest.NewLogger(t),
			}
			got, err := driver.CapturePane(tt.give)
			require.NoError(t, err)
			assert.Equal(t, blob, got)
		})
	}
}

func TestDisplayMessageArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give DisplayMessageRequest
		want []string
	}{
		{
			desc: "empty",
			want: []string{"display-message", "-p", ""},
		},
		{
			desc: "pane",
			give: DisplayMessageRequest{Pane: "%42"},
			want: []string{"display-message", "-p", "-t", "%42", ""},
		},
		{
			desc: "message",
			give: DisplayMessageRequest{Pane: "%42", Message: "#{pane_id}"},
			want: []string{"display-message", "-p", "-t", "%42", "#{pane_id}"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			blob := make([]byte, 10)
			randRead(t, blob)

			r := newFakeRunner(t)
			r.ExpectOutput("tmux", tt.want...).Stdout(blob)

			driver := ShellDriver{
				run: r.Runner(),
				log: logtest.NewLogger(t),
			}
			got, err := driver.DisplayMessage(tt.give)
			require.NoError(t, err)
			assert.Equal(t, blob, got)
		})
	}
}

func TestSetOptionArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give SetOptionRequest
		want []string
	}{
		{
			desc: "local",
			give: SetOptionRequest{Name: "foo", Value: "bar"},
			want: []string{"set-option", "foo", "bar"},
		},
		{
			desc: "global",
			give: SetOptionRequest{Name: "foo", Value: "bar", Global: true},
			want: []string{"set-option", "-g", "foo", "bar"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			r := newFakeRunner(t)
			r.ExpectOutput("tmux", tt.want...)

			driver := ShellDriver{
				run: r.Runner(),
				log: logtest.NewLogger(t),
			}
			err := driver.SetOption(tt.give)
			assert.NoError(t, err)
		})
	}
}

func TestSwapPaneArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give SwapPaneRequest
		want []string
	}{
		{
			desc: "minimal",
			give: SwapPaneRequest{Destination: "%42"},
			want: []string{"swap-pane", "-t", "%42"},
		},
		{
			desc: "source",
			give: SwapPaneRequest{Source: "%43", Destination: "%42"},
			want: []string{"swap-pane", "-t", "%42", "-s", "%43"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			r := newFakeRunner(t)
			r.ExpectOutput("tmux", tt.want...)

			driver := ShellDriver{
				run: r.Runner(),
				log: logtest.NewLogger(t),
			}
			err := driver.SwapPane(tt.give)
			assert.NoError(t, err)
		})
	}
}

func TestResizePaneArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give ResizePaneRequest
		want []string
	}{
		{
			desc: "minimal",
			give: ResizePaneRequest{Target: "%42"},
			want: []string{"resize-pane", "-t", "%42"},
		},
		{
			desc: "source",
			give: ResizePaneRequest{Target: "%43", ToggleZoom: true},
			want: []string{"resize-pane", "-t", "%43", "-Z"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			r := newFakeRunner(t)
			r.ExpectOutput("tmux", tt.want...)

			driver := ShellDriver{
				run: r.Runner(),
				log: logtest.NewLogger(t),
			}
			err := driver.ResizePane(tt.give)
			assert.NoError(t, err)
		})
	}
}

func TestWaitForSignalArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give string
		want []string
	}{
		{
			desc: "sig",
			give: "foo",
			want: []string{"wait-for", "foo"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			r := newFakeRunner(t)
			r.ExpectOutput("tmux", tt.want...)

			driver := ShellDriver{
				run: r.Runner(),
				log: logtest.NewLogger(t),
			}
			err := driver.WaitForSignal(tt.give)
			assert.NoError(t, err)
		})
	}
}

func TestSendSignalArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give string
		want []string
	}{
		{
			desc: "sig",
			give: "foo",
			want: []string{"wait-for", "-S", "foo"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			r := newFakeRunner(t)
			r.ExpectOutput("tmux", tt.want...)

			driver := ShellDriver{
				run: r.Runner(),
				log: logtest.NewLogger(t),
			}
			err := driver.SendSignal(tt.give)
			assert.NoError(t, err)
		})
	}
}

func TestResizeWindow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give ResizeWindowRequest
		want []string
	}{
		{
			desc: "empty",
			want: []string{"resize-window"},
		},
		{
			desc: "window",
			give: ResizeWindowRequest{Window: "foo"},
			want: []string{"resize-window", "-t", "foo"},
		},
		{
			desc: "width",
			give: ResizeWindowRequest{Width: 42},
			want: []string{"resize-window", "-x", "42"},
		},
		{
			desc: "height",
			give: ResizeWindowRequest{Height: 42},
			want: []string{"resize-window", "-y", "42"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			r := newFakeRunner(t)
			r.ExpectOutput("tmux", tt.want...)

			driver := ShellDriver{
				run: r.Runner(),
				log: logtest.NewLogger(t),
			}
			err := driver.ResizeWindow(tt.give)
			assert.NoError(t, err)
		})
	}
}

func TestShowOptionsArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give ShowOptionsRequest
		want []string
	}{
		{
			desc: "empty",
			want: []string{"show-options"},
		},
		{
			desc: "global",
			give: ShowOptionsRequest{Global: true},
			want: []string{"show-options", "-g"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			blob := make([]byte, 10)
			randRead(t, blob)

			r := newFakeRunner(t)
			r.ExpectOutput("tmux", tt.want...).Stdout(blob)

			driver := ShellDriver{
				run: r.Runner(),
				log: logtest.NewLogger(t),
			}
			got, err := driver.ShowOptions(tt.give)
			require.NoError(t, err)
			assert.Equal(t, blob, got)
		})
	}
}

type fakeCall struct {
	name string
	args []string
	out  []byte
}

func (c *fakeCall) Stdout(out []byte) *fakeCall {
	c.out = out
	return c
}

func (c *fakeCall) String() string {
	return fmt.Sprintf("%v %q", c.name, c.args)
}

func (c *fakeCall) matches(cmd *exec.Cmd) bool {
	return c.name == cmd.Args[0] && reflect.DeepEqual(c.args, cmd.Args[1:])
}

type fakeRunner struct {
	t     testing.TB
	mu    sync.Mutex
	calls []*fakeCall
}

func newFakeRunner(t testing.TB) *fakeRunner {
	t.Helper()

	r := &fakeRunner{t: t}
	t.Cleanup(r._verify)
	return r
}

func (r *fakeRunner) Runner() *runner {
	return &runner{
		Output: r.Output,
		Run:    r.Run,
	}
}

func (r *fakeRunner) ExpectOutput(name string, args ...string) *fakeCall {
	call := &fakeCall{name: name, args: args}
	r.mu.Lock()
	r.calls = append(r.calls, call)
	r.mu.Unlock()
	return call
}

func (r *fakeRunner) Run(cmd *exec.Cmd) error {
	r.t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	for i, c := range r.calls {
		if !c.matches(cmd) {
			continue
		}

		// Match!
		copy(r.calls[i:], r.calls[i+1:])
		r.calls = r.calls[:len(r.calls)-1]
		return nil
	}

	r.t.Errorf("unexpected runner.Run call: %v %q", cmd.Args[0], cmd.Args[1:])
	return errors.New("unexpected call")
}

func (r *fakeRunner) Output(cmd *exec.Cmd) ([]byte, error) {
	r.t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	for i, c := range r.calls {
		if !c.matches(cmd) {
			continue
		}

		// Match!
		copy(r.calls[i:], r.calls[i+1:])
		r.calls = r.calls[:len(r.calls)-1]
		return c.out, nil
	}

	r.t.Errorf("unexpected runner.Output call: %v %q", cmd.Args[0], cmd.Args[1:])
	return nil, errors.New("unexpected call")
}

func (r *fakeRunner) _verify() {
	r.t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, c := range r.calls {
		r.t.Errorf("missing call: %v", c)
	}
}

func randRead(t testing.TB, bs []byte) {
	t.Helper()

	_, err := rand.Read(bs)
	require.NoError(t, err)
}
