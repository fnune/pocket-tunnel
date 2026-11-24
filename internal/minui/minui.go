package minui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fnune/pocket-tunnel/internal/app"
)

type UI struct {
	binDir   string
	progress *exec.Cmd
}

func New(binDir string) *UI {
	return &UI{binDir: binDir}
}

var _ app.UI = (*UI)(nil)

type features struct {
	Unselectable    bool   `json:"unselectable,omitempty"`
	BackgroundColor string `json:"background_color,omitempty"`
	ConfirmText     string `json:"confirm_text,omitempty"`
}

type item struct {
	Name     string   `json:"name"`
	Options  []string `json:"options,omitempty"`
	Selected int      `json:"selected,omitempty"`
	Features features `json:"features"`
}

func (u *UI) Menu(title string, items []app.MenuItem) (int, error) {
	payload := struct {
		Items []item `json:"items"`
	}{}
	for _, it := range items {
		payload.Items = append(payload.Items, item{
			Name:     it.Label,
			Options:  it.Options,
			Selected: it.Selected,
			Features: features{
				Unselectable:    it.Unselectable,
				BackgroundColor: it.BackgroundColor,
				ConfirmText:     it.ConfirmText,
			},
		})
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return -1, fmt.Errorf("encode menu: %w", err)
	}

	dir, err := os.MkdirTemp("", "pocket-tunnel-menu-")
	if err != nil {
		return -1, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	input := filepath.Join(dir, "input.json")
	output := filepath.Join(dir, "output.txt")
	if err := os.WriteFile(input, data, 0644); err != nil {
		return -1, err
	}

	cmd := exec.Command(filepath.Join(u.binDir, "minui-list"),
		"--file", input, "--format", "json", "--item-key", "items",
		"--write-location", output, "--title", title)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && (exitErr.ExitCode() == 2 || exitErr.ExitCode() == 3) {
			return -1, app.ErrBack
		}
		return -1, fmt.Errorf("minui-list: %w", err)
	}

	selected, err := os.ReadFile(output)
	if err != nil {
		return -1, fmt.Errorf("read menu choice: %w", err)
	}
	label := strings.TrimSpace(string(selected))
	for i, it := range items {
		if it.Label == label {
			return i, nil
		}
	}
	return -1, app.ErrBack
}

func (u *UI) ShowProgress(text string) error {
	u.CloseProgress()
	cmd := exec.Command(filepath.Join(u.binDir, "minui-presenter"), "--message", text, "--timeout", "-1")
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("minui-presenter: %w", err)
	}
	u.progress = cmd
	return nil
}

func (u *UI) CloseProgress() {
	if u.progress == nil {
		return
	}
	_ = u.progress.Process.Kill()
	_ = u.progress.Wait()
	u.progress = nil
}

func (u *UI) Message(text string) error {
	cmd := exec.Command(filepath.Join(u.binDir, "minui-presenter"), "--message", text, "--timeout", "0", "--confirm-show", "--confirm-text", "OK")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 2 {
			return nil
		}
		return fmt.Errorf("minui-presenter: %w", err)
	}
	return nil
}
