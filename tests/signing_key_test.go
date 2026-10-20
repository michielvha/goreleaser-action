package tests

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/michielvha/goreleaser-action/internal/testkey"
)

type result struct {
	code   int
	stdout string
	stderr string
}

type importedKey struct {
	result      result
	home        string
	fingerprint string
}

type fixture struct {
	directory string
	keyrings  []string
}

func TestSigningKey(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	key, err := testkey.Create(ctx, "Release signing fixture")
	if err != nil {
		t.Fatal(err)
	}
	other, err := testkey.Create(ctx, "Unrelated signing fixture")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("private primary key with multiple subkeys is isolated", func(t *testing.T) {
		f := newFixture(t)
		imported := f.importKey(t, key.PrivateKey, nil)
		if imported.result.code != 0 {
			t.Fatal(imported.result.stderr)
		}
		if imported.fingerprint != key.Fingerprint {
			t.Fatalf("fingerprint = %s, want the primary fingerprint %s", imported.fingerprint, key.Fingerprint)
		}
		info, err := os.Stat(imported.home)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Errorf("keyring permissions = %o, want 700", info.Mode().Perm())
		}
		if strings.Contains(imported.result.stdout+imported.result.stderr, "-----BEGIN PGP PRIVATE KEY BLOCK-----") {
			t.Error("private key material was logged")
		}
	})
	t.Run("checksum signatures verify and tampering fails", func(t *testing.T) {
		f := newFixture(t)
		imported := f.importKey(t, key.PrivateKey, nil)
		if imported.result.code != 0 {
			t.Fatal(imported.result.stderr)
		}
		checksums := filepath.Join(f.directory, "checksums.txt")
		signature := checksums + ".sig"
		write(t, checksums, fmt.Sprintf("%x  fixture.tar.gz\n", sha256.Sum256([]byte("fixture artifact"))))
		env := []string{"GNUPGHOME=" + imported.home}
		if got := command(t, env, "gpg", "--batch", "--local-user", imported.fingerprint,
			"--output", signature, "--detach-sign", checksums); got.code != 0 {
			t.Fatal(got.stderr)
		}
		if got := command(t, env, "gpg", "--batch", "--verify", signature, checksums); got.code != 0 {
			t.Fatal(got.stderr)
		}
		write(t, checksums, "tampered checksum\n")
		if command(t, env, "gpg", "--batch", "--verify", signature, checksums).code == 0 {
			t.Fatal("the altered checksum passed signature verification")
		}
	})
	t.Run("the caller keyring is not modified", func(t *testing.T) {
		f := newFixture(t)
		caller := f.importKey(t, other.PrivateKey, nil)
		if caller.result.code != 0 {
			t.Fatal(caller.result.stderr)
		}
		env := []string{"GNUPGHOME=" + caller.home}
		before := command(t, env, "gpg", "--batch", "--with-colons", "--list-secret-keys")
		if before.code != 0 {
			t.Fatal(before.stderr)
		}
		imported := f.importKey(t, key.PrivateKey, env)
		if imported.result.code != 0 || imported.home == caller.home {
			t.Fatalf("import did not use a separate keyring: %s", imported.result.stderr)
		}
		after := command(t, env, "gpg", "--batch", "--with-colons", "--list-secret-keys")
		if after.code != 0 || after.stdout != before.stdout {
			t.Fatalf("the caller's keyring changed: %s", after.stderr)
		}
	})
	t.Run("empty invalid and public-only inputs fail", func(t *testing.T) {
		f := newFixture(t)
		for _, input := range []string{"", "not an OpenPGP key", key.PublicKey} {
			imported := f.importKey(t, input, nil)
			if imported.result.code == 0 || imported.fingerprint != "" {
				t.Fatal("invalid input produced a successful private-key import")
			}
		}
	})
	t.Run("multiple primary private keys fail explicitly", func(t *testing.T) {
		f := newFixture(t)
		imported := f.importKey(t, key.PrivateKey+"\n"+other.PrivateKey, nil)
		if imported.result.code == 0 || imported.fingerprint != "" ||
			!strings.Contains(imported.result.stderr, "exactly one private primary key") {
			t.Fatalf("expected an ambiguous-key error, got:\n%s", imported.result.stderr)
		}
	})
	t.Run("cleanup removes only the imported keyring", func(t *testing.T) {
		f := newFixture(t)
		imported := f.importKey(t, key.PrivateKey, nil)
		if imported.result.code != 0 {
			t.Fatal(imported.result.stderr)
		}
		unrelated := filepath.Join(f.directory, "unrelated")
		write(t, unrelated, "preserve")
		if f.run(t, "cleanup", []string{"GNUPGHOME=" + f.directory}).code == 0 {
			t.Fatal("cleanup accepted a directory not created by the action")
		}
		if got := f.run(t, "cleanup", []string{"GNUPGHOME=" + imported.home}); got.code != 0 {
			t.Fatal(got.stderr)
		}
		if _, err := os.Stat(imported.home); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("the imported keyring remains: %v", err)
		}
		raw, err := os.ReadFile(unrelated)
		if err != nil || string(raw) != "preserve" {
			t.Fatalf("cleanup changed an unrelated file: %v", err)
		}
	})
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	directory, err := os.MkdirTemp(testkey.TempRoot(), "release-import-")
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{directory: directory}
	t.Cleanup(func() {
		for _, home := range f.keyrings {
			if _, err := os.Stat(home); errors.Is(err, os.ErrNotExist) {
				continue
			} else if err != nil {
				t.Error(err)
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			cmd := exec.CommandContext(ctx, "bash", "../scripts/signing-key.sh", "cleanup")
			cmd.Env = append(os.Environ(), "RUNNER_TEMP="+directory, "GNUPGHOME="+home)
			if raw, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("keyring cleanup failed: %v: %s", err, raw)
			}
			cancel()
		}
		if err := os.RemoveAll(directory); err != nil {
			t.Error(err)
		}
	})
	return f
}

func (f *fixture) run(t *testing.T, operation string, env []string) result {
	t.Helper()
	env = append([]string{"RUNNER_TEMP=" + f.directory, "GITHUB_OUTPUT=" + filepath.Join(f.directory, "outputs")}, env...)
	return command(t, env, "bash", "../scripts/signing-key.sh", operation)
}

func (f *fixture) importKey(t *testing.T, key string, env []string) importedKey {
	t.Helper()
	output := filepath.Join(f.directory, "outputs")
	write(t, output, "")
	got := f.run(t, "import", append(env, "PGP_PRIVATE_KEY="+key))
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	imported := importedKey{result: got}
	for line := range strings.SplitSeq(string(raw), "\n") {
		name, value, _ := strings.Cut(line, "=")
		switch name {
		case "gnupghome":
			imported.home = value
			f.keyrings = append(f.keyrings, value)
		case "fingerprint":
			imported.fingerprint = value
		}
	}
	return imported
}

func command(t *testing.T, env []string, name string, args ...string) result {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatal(ctx.Err())
	}
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatal(err)
		}
		code = exit.ExitCode()
	}
	return result{code, stdout.String(), stderr.String()}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
