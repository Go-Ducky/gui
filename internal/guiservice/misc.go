package guiservice

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/go-ducky/gui/internal/provider"
	"github.com/go-ducky/gui/internal/session"
	"github.com/go-ducky/gui/internal/setup"
)

// Version is the app version, set at build time via -ldflags.
var Version = "0.1.0"

// AppVersion returns the GoDucky GUI version.
func (s *Service) AppVersion() string { return Version }

// OpenURL opens a URL in the system browser.
func (s *Service) OpenURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not open browser: %w", err)
	}
	return nil
}

// CopyText copies plain text to the system clipboard (used by share/export).
func (s *Service) CopyText(text string) error {
	if strings.TrimSpace(text) == "" {
		return errors.New("nothing to copy")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "windows":
		cmd = exec.Command("cmd", "/c", "clip")
	default:
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else {
			cmd = exec.Command("wl-copy")
		}
	}
	cmd.Stdin = strings.NewReader(text)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("clipboard copy failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ShareSession copies a saved (or the current) conversation to the clipboard
// as a Markdown transcript. An empty name shares the active chat.
func (s *Service) ShareSession(name string) error {
	md, err := s.sessionMarkdown(name)
	if err != nil {
		return err
	}
	if err := s.CopyText(md); err != nil {
		return err
	}
	s.emit(EvtStatus, map[string]any{"msg": "Conversation copied to clipboard"})
	return nil
}

func (s *Service) sessionMarkdown(name string) (string, error) {
	name = strings.TrimSpace(name)
	var msgs []provider.Message
	if name == "" {
		s.mu.Lock()
		msgs = make([]provider.Message, len(s.messages))
		copy(msgs, s.messages)
		s.mu.Unlock()
	} else {
		ss, err := session.Load(name)
		if err != nil {
			return "", err
		}
		msgs = ss.Messages
	}
	if len(msgs) == 0 {
		return "", errors.New("this chat is empty")
	}
	b := &strings.Builder{}
	b.WriteString("# GoDucky conversation\n\n")
	for _, m := range msgs {
		who := "GoDucky"
		if m.Role == provider.RoleUser {
			who = "You"
		}
		b.WriteString("## " + who + "\n\n")
		for _, c := range m.Content {
			if c.Type == "text" && c.Text != "" {
				b.WriteString(c.Text + "\n\n")
			}
		}
	}
	return strings.TrimSpace(b.String()), nil
}

// InstallOllama downloads and installs Ollama for the current OS in the
// background, streaming status updates as events.
func (s *Service) InstallOllama() {
	go func() {
		status := func(msg string) {
			s.emit(EvtStatus, map[string]any{"msg": msg})
		}
		ctx, cancel := contextWithTimeout(30 * 60 * time.Minute)
		defer cancel()
		_ = setup.InstallOllama(ctx, status)
		status("Ollama install finished. Checking again...")
	}()
}

// RecommendedLocalModels lists the curated local model shortlist.
func (s *Service) RecommendedLocalModels() []string {
	return setup.RecommendedModelIDs()
}

// StartLocalModelPulls starts pulling each of the given models through Ollama.
func (s *Service) StartLocalModelPulls(models []string) {
	for _, m := range models {
		if s.HasModel(m) {
			continue
		}
		s.PullModel(m)
	}
}
