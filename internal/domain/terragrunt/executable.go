package terragrunt

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

func initTerragrunt(ctx context.Context, path string) (string, error) {
	if err := CheckExecutable(path); err != nil {
		return "", err
	}

	version, err := GetVersion(ctx, path)
	if err != nil {
		return "", fmt.Errorf("could not get terrgrunt version %w", err)
	}
	var vReg = regexp.MustCompile(`(?i)^terragrunt\s+version\s+v?(\d+\.\d+\.\d+)`)
	matches := vReg.FindStringSubmatch(version)

	if len(matches) < 2 {
		return "", fmt.Errorf("invalid terragrunt output: '%s'", version)
	}
	return matches[1], nil
}
func initTerraForm(ctx context.Context, path string) (string, error) {
	if err := CheckExecutable(path); err != nil {
		return "", err
	}

	version, err := GetVersion(ctx, path)
	if err != nil {
		return "", fmt.Errorf("could not get terraform version %w", err)
	}
	var vReg = regexp.MustCompile(`(?i)^Terraform\s+v?(\d+\.\d+\.\d+)`)
	matches := vReg.FindStringSubmatch(version)

	if len(matches) < 2 {
		return "", fmt.Errorf("invalid terraform output: '%s'", version)
	}
	return matches[1], nil
}
func CheckExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", path)
		}
		return fmt.Errorf("failed to check file status: %w", err)
	}

	// Ensure it is not a directory
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not an executable file: %s", path)
	}

	// Check for execution bit (User, Group, or Other)
	// 0111 octal mask checks if --x--x--x is present
	if info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("file is not executable (missing +x permission): %s", path)
	}

	return nil
}
func GetVersion(ctx context.Context, binPath string) (string, error) {
	// 1. Define a Timeout (Safety First)
	// Se o comando demorar mais de 2s para responder a versão, algo está errado.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 2. Prepare the command
	cmd := exec.CommandContext(ctx, binPath, "-version")

	// 3. Prepare o novo ambiente
	var newEnv []string
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "TF_LOG=") {
			newEnv = append(newEnv, entry)
		}
	}
	cmd.Env = newEnv
	// 3. Capture Output (both stdout and stderr to debug if fails)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	// 4. Execute
	if err := cmd.Run(); err != nil {
		// Verifica se foi timeout
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("command timed out: %s --version", binPath)
		}
		return "", fmt.Errorf("failed to get version for %s: %w. Output: %s", binPath, err, out.String())
	}

	// 5. Clean result (Remove \n, \r and extra spaces)
	return strings.TrimSpace(out.String()), nil
}
