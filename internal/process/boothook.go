package process

import (
	"fmt"
	"os"
	"strings"

	"github.com/fnune/pocket-tunnel/internal/app"
)

const (
	markerStart = "# BEGIN POCKET-TUNNEL"
	markerEnd   = "# END POCKET-TUNNEL"
)

type BootHook struct {
	autoSh  string
	command []string
	logFile string
}

func NewBootHook(autoSh string, command []string, logFile string) *BootHook {
	return &BootHook{autoSh: autoSh, command: command, logFile: logFile}
}

var _ app.BootHook = (*BootHook)(nil)

func (b *BootHook) Enabled() bool {
	content, err := os.ReadFile(b.autoSh)
	return err == nil && strings.Contains(string(content), markerStart)
}

func (b *BootHook) Enable() error {
	if b.Enabled() {
		return nil
	}
	content, err := os.ReadFile(b.autoSh)
	if os.IsNotExist(err) {
		content = []byte("#!/bin/sh\n")
	} else if err != nil {
		return fmt.Errorf("read %s: %w", b.autoSh, err)
	}
	text := string(content)
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return b.write(text + "\n" + b.block())
}

func (b *BootHook) Disable() error {
	content, err := os.ReadFile(b.autoSh)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", b.autoSh, err)
	}
	return b.write(removeBlock(string(content)))
}

func (b *BootHook) write(content string) error {
	if err := os.WriteFile(b.autoSh, []byte(content), 0755); err != nil {
		return fmt.Errorf("write %s: %w", b.autoSh, err)
	}
	return nil
}

func (b *BootHook) block() string {
	quoted := make([]string, len(b.command))
	for i, arg := range b.command {
		quoted[i] = shellQuote(arg)
	}
	return fmt.Sprintf("%s\n%s >> %s 2>&1 &\n%s\n", markerStart, strings.Join(quoted, " "), shellQuote(b.logFile), markerEnd)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func removeBlock(content string) string {
	start := strings.Index(content, markerStart)
	end := strings.Index(content, markerEnd)
	if start == -1 || end == -1 {
		return content
	}
	end += len(markerEnd)
	for end < len(content) && content[end] == '\n' {
		end++
	}
	for start > 0 && content[start-1] == '\n' {
		start--
	}
	if start > 0 {
		start++
	}
	return content[:start] + content[end:]
}
