package guiservice

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"

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
