package login

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/cli/cli/internal/config"
	"github.com/cli/cli/pkg/cmdutil"
	"github.com/cli/cli/pkg/httpmock"
	"github.com/cli/cli/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
)

func Test_NewCmdLogin(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		stdin    string
		stdinTTY bool
		wants    LoginOptions
		wantsErr bool
	}{
		{
			name:  "nontty, with-token",
			stdin: "abc123\n",
			cli:   "--with-token",
			wants: LoginOptions{
				Hostname: "github.com",
				Token:    "abc123",
			},
		},
		{
			name:     "tty, with-token",
			stdinTTY: true,
			wantsErr: true,
			cli:      "--with-token",
		},
		{
			name:     "nontty, hostname",
			cli:      "--hostname claire.redfield",
			wantsErr: true,
		},
		{
			name:     "nontty",
			cli:      "",
			wantsErr: true,
		},
		{
			name:  "nontty, with-token, hostname",
			cli:   "--hostname claire.redfield --with-token",
			stdin: "abc123\n",
			wants: LoginOptions{
				Hostname: "claire.redfield",
				Token:    "abc123",
			},
		},
		{
			name:     "tty, with-token, hostname",
			stdinTTY: true,
			wantsErr: true,
			cli:      "--with-token",
		},
		{
			name:     "tty, hostname",
			stdinTTY: true,
			cli:      "--hostname barry.burton",
			wants: LoginOptions{
				Hostname: "barry.burton",
				Token:    "",
			},
		},
		{
			name:     "tty",
			stdinTTY: true,
			cli:      "",
			wants: LoginOptions{
				Hostname: "",
				Token:    "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			io, stdin, _, _ := iostreams.Test()
			f := &cmdutil.Factory{
				IOStreams: io,
			}

			io.SetStdinTTY(tt.stdinTTY)
			if tt.stdin != "" {
				stdin.WriteString(tt.stdin)
			}

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			var gotOpts *LoginOptions
			cmd := NewCmdLogin(f, func(opts *LoginOptions) error {
				gotOpts = opts
				return nil
			})

			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			_, err = cmd.ExecuteC()
			if tt.wantsErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, tt.wants.Token, gotOpts.Token)
			assert.Equal(t, tt.wants.Hostname, gotOpts.Hostname)
		})
	}
}

func Test_loginRun(t *testing.T) {
	tests := []struct {
		name       string
		opts       *LoginOptions
		tty        bool
		httpStubs  func(*httpmock.Registry)
		wantStdout string
		wantStderr string
		wantHosts  string
		wantsErr   bool
	}{
		{
			name:     "non-tty no arguments",
			opts:     &LoginOptions{},
			wantsErr: true,
		},
		{
			name: "non-tty with token",
			opts: &LoginOptions{
				Token: "abc123",
			},
			wantHosts: "LOL TODO",
			httpStubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "user"),
					func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: 200,
							Request:    req,
							Header: map[string][]string{
								"X-Oauth-Scopes": {"repo,read:org"},
							},
							Body: ioutil.NopCloser(bytes.NewBufferString("")),
						}, nil
					},
				)
			},
		},
		// TODO non-tty with token + hostname
		// TODO non-tty with hostname
	}

	for _, tt := range tests {
		io, _, stdout, stderr := iostreams.Test()

		io.SetStdinTTY(tt.tty)
		io.SetStdoutTTY(tt.tty)

		tt.opts.Config = func() (config.Config, error) {
			return config.NewBlankConfig(), nil
		}

		reg := &httpmock.Registry{}
		tt.opts.HttpClient = func() (*http.Client, error) {
			return &http.Client{Transport: reg}, nil
		}

		if tt.httpStubs != nil {
			tt.httpStubs(reg)
		}

		tt.opts.IO = io
		t.Run(tt.name, func(t *testing.T) {
			mainBuf := bytes.Buffer{}
			// TODO DEBUG WHY hostsBuf is EMPTY
			hostsBuf := bytes.Buffer{}
			defer config.StubWriteConfig(&mainBuf, &hostsBuf)()

			if err := loginRun(tt.opts); (err != nil) != tt.wantsErr {
				t.Errorf("loginRun() error = %v, wantErr %v", err, tt.wantsErr)
			}

			defer reg.Verify(t)

			assert.Equal(t, tt.wantStdout, stdout.String())
			assert.Equal(t, tt.wantStderr, stderr.String())
			assert.Equal(t, tt.wantHosts, hostsBuf.String())
		})
	}
}

/*
func Test_loginRun_ConfiguresProtocol(t *testing.T) {
	io, _, _, _ := iostreams.Test()

	reg := &httpmock.Registry{}
	opts := &LoginOptions{
		IO: io,
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: reg}, nil
		},
	}
	cmd := NewCmdLogin()

}
*/
