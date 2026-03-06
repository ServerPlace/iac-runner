package git

import (
	"bytes"
	"os/exec"
	"strings"
)

type Client struct{}

func (c *Client) GetChangedFiles(baseBranch string) ([]string, error) {
	// Nota: Em um ambiente real, garanta que o git fetch foi feito
	cmd := exec.Command("git", "diff", "--name-only", baseBranch)
	var out bytes.Buffer
	cmd.Stdout = &out
	// Se o comando falhar (ex: fora de um repo git), retornamos erro
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	raw := strings.Split(strings.TrimSpace(out.String()), "\n")
	var files []string
	for _, f := range raw {
		if f != "" {
			files = append(files, f)
		}
	}
	return files, nil
}
