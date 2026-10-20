package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/michielvha/goreleaser-action/internal/testkey"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	output := os.Getenv("GITHUB_OUTPUT")
	temp := os.Getenv("RUNNER_TEMP")
	if output == "" || temp == "" {
		return fmt.Errorf("GITHUB_OUTPUT and RUNNER_TEMP are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	key, err := testkey.Create(ctx, "GoReleaser snapshot fixture")
	if err != nil {
		return err
	}
	escaped := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(key.PrivateKey)
	fmt.Println("::add-mask::" + escaped)
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	_, writeErr := fmt.Fprintf(file, "private-key<<PRIVATE_KEY\n%s\nPRIVATE_KEY\n", key.PrivateKey)
	if err := file.Close(); err != nil {
		return err
	}
	if writeErr != nil {
		return writeErr
	}
	return os.WriteFile(filepath.Join(temp, "release-fixture-public.asc"), []byte(key.PublicKey), 0o600)
}
