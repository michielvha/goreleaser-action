package testkey

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Key struct {
	Fingerprint string
	PrivateKey  string
	PublicKey   string
}

// macOS's default temporary path can exceed GnuPG's Unix socket length once nested.
func TempRoot() string {
	if runtime.GOOS == "darwin" {
		return "/tmp"
	}
	return os.TempDir()
}

func Create(ctx context.Context, name string) (key Key, err error) {
	for _, tool := range []string{"gpg", "gpgconf", "gpg-connect-agent"} {
		if _, err := exec.LookPath(tool); err != nil {
			return Key{}, err
		}
	}
	directory, err := os.MkdirTemp(TempRoot(), "release-key-")
	if err != nil {
		return Key{}, err
	}
	defer func() {
		err = errors.Join(err, stopAgent(directory), os.RemoveAll(directory))
	}()
	env := append(os.Environ(), "GNUPGHOME="+directory)
	gpg := func(args ...string) (string, error) {
		flags := append([]string{"--batch", "--pinentry-mode", "loopback", "--passphrase", ""}, args...)
		return output(ctx, env, "gpg", flags...)
	}
	if _, err := gpg("--quick-generate-key", name+" <fixture@example.invalid>", "ed25519", "cert", "0"); err != nil {
		return Key{}, err
	}
	listing, err := gpg("--with-colons", "--list-secret-keys")
	if err != nil {
		return Key{}, err
	}
	for line := range strings.SplitSeq(listing, "\n") {
		fields := strings.Split(line, ":")
		if len(fields) > 9 && fields[0] == "fpr" {
			key.Fingerprint = fields[9]
			break
		}
	}
	if key.Fingerprint == "" {
		return Key{}, errors.New("the fixture key has no primary fingerprint")
	}
	for _, subkey := range [][2]string{{"ed25519", "sign"}, {"cv25519", "encrypt"}} {
		if _, err := gpg("--quick-add-key", key.Fingerprint, subkey[0], subkey[1], "0"); err != nil {
			return Key{}, err
		}
	}
	key.PrivateKey, err = gpg("--armor", "--export-secret-keys", key.Fingerprint)
	if err != nil {
		return Key{}, err
	}
	key.PublicKey, err = gpg("--armor", "--export", key.Fingerprint)
	return key, err
}

func stopAgent(home string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	env := append(os.Environ(), "GNUPGHOME="+home)
	socket, err := output(ctx, env, "gpgconf", "--list-dirs", "agent-socket")
	if err != nil {
		return err
	}
	if _, err := os.Stat(strings.TrimSpace(socket)); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	info, err := output(ctx, env, "gpg-connect-agent", "--no-autostart", "GETINFO pid", "/bye")
	if err != nil {
		return err
	}
	for line := range strings.SplitSeq(info, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[0] != "D" {
			continue
		}
		pid, err := strconv.Atoi(fields[1])
		if err != nil || pid <= 1 {
			return errors.New("the fixture agent has no valid PID")
		}
		process, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		if err := process.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) &&
			!errors.Is(err, syscall.ESRCH) {
			return err
		}
		return nil
	}
	return errors.New("the fixture agent returned no PID")
}

func output(ctx context.Context, env []string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	raw, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return "", fmt.Errorf("%s: %w: %s", filepath.Base(name), err, exit.Stderr)
		}
		return "", err
	}
	return string(raw), nil
}
