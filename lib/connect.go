package connect

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"
	"github.com/mutagen-io/mutagen/pkg/selection"
	serviceSync "github.com/mutagen-io/mutagen/pkg/service/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/terminal"
)

func CalculatePath(alphaPath string, betaPath string, pwd string) string {
	repoPath := strings.TrimPrefix(pwd, alphaPath)
	return filepath.Join(betaPath, repoPath)
}

func GetStates() ([]*synchronization.State, error) {
	daemonConnection, err := daemon.Connect(true, false)
	if err != nil {
		return nil, fmt.Errorf("connect to mutagen daemon: %v", err)
	}
	defer daemonConnection.Close()

	synchronizationService := serviceSync.NewSynchronizationClient(daemonConnection)
	request := &serviceSync.ListRequest{
		Selection: &selection.Selection{All: true},
	}
	response, err := synchronizationService.List(context.Background(), request)
	if err != nil {
		return nil, fmt.Errorf("get list of mutagen sessions: %v", err)
	}
	return response.GetSessionStates(), nil
}

func MatchingState(ctx context.Context, states []*synchronization.State, pwd string) (*synchronization.State, error) {
	var found *synchronization.State
	var disconnected bool
	for _, state := range states {
		if strings.Contains(pwd, state.GetSession().Alpha.Path) {
			if state.Status == synchronization.Status_Disconnected {
				disconnected = true
				continue
			}
			if found != nil {
				return nil, fmt.Errorf("more than one matching session found: betas '%s:%s' and '%s:%s'", found.Session.Beta.Host, found.Session.Beta.Path, state.GetSession().Beta.Host, state.GetSession().Beta.Path)
			}
			found = state
		}
	}
	if disconnected {
		return nil, fmt.Errorf("a matching session is disconnected")
	}
	if found == nil {
		return nil, fmt.Errorf("no session found for current path")
	}
	return found, nil
}

func ConnectState(ctx context.Context, state *synchronization.State, pwd string) error {
	sockAddress := os.Getenv("SSH_AUTH_SOCK")
	socket, err := net.Dial("unix", sockAddress)
	if err != nil {
		return fmt.Errorf("ssh agent: %w", err)
	}

	ag := agent.NewClient(socket)
	usr, err := user.Current()
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}
	config := &ssh.ClientConfig{
		User: usr.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(ag.Signers),
		},
		Timeout:         10 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", state.GetSession().Beta.Host+":22", config)
	if err != nil {
		return fmt.Errorf("ssh dial: %w", err)
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("ssh new session: %w", err)
	}
	defer session.Close()

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	fd := int(os.Stdin.Fd())
	origTerminal, err := terminal.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("terminal make raw: %w", err)
	}
	defer terminal.Restore(fd, origTerminal)

	w, h, err := terminal.GetSize(fd)
	if err != nil {
		return fmt.Errorf("terminal get size: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	term := os.Getenv("TERM")
	if term == "" {
		term = "xterm-256color"
	}
	if err := session.RequestPty(term, h, w, modes); err != nil {
		return fmt.Errorf("session xterm: %w", err)
	}
	stdin, _ := session.StdinPipe()
	go io.Copy(stdin, os.Stdin)
	stdout, _ := session.StdoutPipe()
	go io.Copy(os.Stdout, stdout)
	stderr, _ := session.StderrPipe()
	go io.Copy(os.Stderr, stderr)

	go func() {
		stdin.Write([]byte(fmt.Sprintf("cd %s\n", CalculatePath(state.GetSession().Alpha.Path, state.GetSession().Beta.Path, pwd))))
	}()

	if err := session.Shell(); err != nil {
		return fmt.Errorf("session shell: %w", err)
	}

	if err := session.Wait(); err != nil {
		if e, ok := err.(*ssh.ExitError); ok {
			switch e.ExitStatus() {
			case 130:
				return nil
			}
		}
		return fmt.Errorf("ssh: %w", err)
	}
	return nil
}
