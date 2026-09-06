package guiservice

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-ducky/gui/internal/agent"
	"github.com/go-ducky/gui/internal/agent/tools"
	"github.com/go-ducky/gui/internal/config"
	"github.com/go-ducky/gui/internal/provider"
	"github.com/go-ducky/gui/internal/session"
	"github.com/go-ducky/gui/internal/setup"
)

// Event names emitted to the UI via the event sink.
const (
	EvtStream    = "agent:stream"
	EvtToolStart = "agent:tool:start"
	EvtToolEnd   = "agent:tool:end"
	EvtStatus    = "agent:status"
	EvtComplete  = "agent:complete"
	EvtApproval  = "agent:approval"
	EvtError     = "agent:error"
	EvtOllamaOp  = "ollama:op"
	EvtModels    = "models"
)

// Service is the shared agent service for the GoDucky GUI. Native frontends
// (GTK, Qt, …) install an event sink and call its methods directly.
type Service struct {
	mu      sync.Mutex
	cfg     *config.Config
	auth    *config.Auth
	agent   *agent.Agent
	workDir string
	running bool

	runCtx    context.Context
	cancelRun context.CancelFunc

	sessionName string
	messages    []provider.Message

	approvalPending bool
	approvalRespond chan bool

	// sink, when set, receives UI events (installed by native frontends).
	sinkMu sync.RWMutex
	sink   func(name string, data any)
}

// NewService builds a Service that loads config/auth on construction.
func NewService() *Service {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.Default()
	}
	auth, err := config.LoadAuth()
	if err != nil {
		auth = &config.Auth{}
	}
	return &Service{cfg: cfg, auth: auth}
}

// SetEventSink installs a UI event receiver for native frontends. Events are
// dropped when no sink is set; native UIs always install one on startup.
func (s *Service) SetEventSink(fn func(name string, data any)) {
	s.sinkMu.Lock()
	defer s.sinkMu.Unlock()
	s.sink = fn
}

func (s *Service) emit(name string, data any) {
	s.sinkMu.RLock()
	sink := s.sink
	s.sinkMu.RUnlock()
	if sink != nil {
		sink(name, data)
	}
}

// defaultWorkDir mirrors the CLI: project files and chats live in
// ~/Documents/GoDucky Projects on every platform (macOS and Windows included).
func defaultWorkDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return agent.CurrentDir()
	}
	wd := filepath.Join(home, "Documents", "GoDucky Projects")
	if err := os.MkdirAll(wd, 0o755); err == nil {
		return wd
	}
	return agent.CurrentDir()
}

// Info returns the current state snapshot the frontend needs on load.
type Info struct {
	Provider  string        `json:"provider"`
	Model     string        `json:"model"`
	WorkDir   string        `json:"work_dir"`
	Onboarded bool          `json:"onboarded"`
	Name      string        `json:"name"`
	Messages  []MsgView     `json:"messages"`
	Sessions  []SessionView `json:"sessions"`
}

// MsgView is a plain serializable view of a chat message for the frontend.
type MsgView struct {
	Role string `json:"role"` // user | assistant | system
	Text string `json:"text"`
}

// SessionView is a plain serializable summary of a saved session.
type SessionView struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	WorkDir  string `json:"work_dir"`
	Updated  string `json:"updated"`
	Preview  string `json:"preview"`
}

func buildSessionViews(ss []session.Session) []SessionView {
	out := make([]SessionView, 0, len(ss))
	for i := range ss {
		out = append(out, SessionView{
			Name:     ss[i].Name,
			Provider: ss[i].Provider,
			Model:    ss[i].Model,
			WorkDir:  ss[i].WorkDir,
			Updated:  ss[i].UpdatedAt.Format("2006-01-02 15:04"),
			Preview:  preview(ss[i].Messages),
		})
	}
	return out
}

func preview(msgs []provider.Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		for _, b := range msgs[i].Content {
			if b.Type == "text" && b.Text != "" {
				t := strings.TrimSpace(b.Text)
				if len(t) > 80 {
					t = t[:80] + "…"
				}
				return t
			}
		}
	}
	return ""
}

// GetInfo fetches config + auth + sessions all at once.
func (s *Service) GetInfo() *Info {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked()
}

func (s *Service) snapshotLocked() *Info {
	model := provider.ResolveModel(s.cfg, "")
	out := &Info{
		Provider:  s.cfg.Provider,
		Model:     model,
		WorkDir:   s.workDir,
		Onboarded: s.cfg.Onboarded,
		Name:      s.sessionName,
		Messages:  msgsToView(s.messages),
	}
	if ss, err := session.List(); err == nil {
		out.Sessions = buildSessionViews(ss)
	}
	return out
}

func msgsToView(msgs []provider.Message) []MsgView {
	out := make([]MsgView, 0, len(msgs))
	for _, m := range msgs {
		for _, b := range m.Content {
			if b.Type == "text" && b.Text != "" {
				role := "assistant"
				if m.Role == provider.RoleUser {
					role = "user"
				}
				out = append(out, MsgView{Role: role, Text: b.Text})
			}
		}
	}
	return out
}

// initAgent (re)builds the agent backend from the current cfg+auth.
func (s *Service) initAgent() {
	if s.workDir == "" {
		s.workDir = defaultWorkDir()
	}
	p, err := provider.New(s.cfg, s.auth)
	if err != nil {
		s.emit(EvtError, map[string]any{"error": err.Error()})
		return
	}
	modelName := provider.ResolveModel(s.cfg, "")
	reg := tools.DefaultRegistry()
	sys := agent.SystemPrompt(s.workDir)
	a := agent.New(p, modelName, sys, s.workDir, &agent.Config{
		MaxIterations:  s.cfg.Agent.MaxIterations,
		MaxOutputChars: s.cfg.Agent.MaxOutputChars,
		AutoApprove:    s.cfg.Agent.AutoApprove,
	}, reg)
	s.agent = a
}

// SetWorkDir changes the working directory (creates it if needed).
func (s *Service) SetWorkDir(dir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return err
	}
	s.workDir = abs
	s.initAgent()
	s.emit(EvtStatus, map[string]any{"msg": "Working directory: " + abs})
	return nil
}

// GetModels lists models for a provider (live where possible, else curated).
func (s *Service) GetModels(providerName string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := s.cfg
	auth := s.auth
	return fetchModels(providerName, cfg, auth)
}

func fetchModels(prov string, cfg *config.Config, auth *config.Auth) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var out []string
	switch prov {
	case "ollama":
		installed, err := provider.NewOllama(cfg).ListModels(ctx)
		if err == nil {
			out = append(out, installed...)
		}
		out = append(out, setup.RecommendedModelIDs()...)
	case "openrouter":
		if free, err := provider.OpenRouterFreeModels(ctx, ""); err == nil && len(free) > 0 {
			return free
		}
		out = curatedModels(prov)
	case "groq":
		if models, err := provider.NewGroq(cfg, auth, true).ListModels(ctx); err == nil && len(models) > 0 {
			return models
		}
		out = curatedModels(prov)
	case "openai", "openai_compatible":
		if models, err := provider.NewOpenAI(cfg, auth, prov == "openai_compatible").ListModels(ctx); err == nil && len(models) > 0 {
			return models
		}
		out = curatedModels(prov)
	default:
		out = curatedModels(prov)
	}
	return dedupe(out)
}

func dedupe(list []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(list))
	for _, m := range list {
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}

func curatedModels(prov string) []string {
	switch prov {
	case "groq":
		return []string{"llama-3.3-70b-versatile", "llama-3.1-8b-instant", "llama3-8b-8192"}
	case "openai":
		return []string{"gpt-4o-mini", "gpt-4o", "gpt-5-mini"}
	case "openai_compatible":
		return []string{"qwen2.5-coder:7b", "gpt-4o-mini"}
	case "anthropic":
		return []string{"claude-3-5-haiku-latest", "claude-3-5-sonnet-latest"}
	case "gemini":
		return []string{"gemini-2.0-flash", "gemini-1.5-flash", "gemini-1.5-pro"}
	case "openrouter":
		return []string{"openrouter/free", "qwen/qwen-2.5-coder-7b-instruct"}
	}
	return setup.RecommendedModelIDs()
}

// Providers lists the supported provider names.
func (s *Service) Providers() []string {
	return []string{"ollama", "groq", "openai", "openai_compatible", "anthropic", "gemini", "openrouter"}
}

// SwitchProvider sets the active provider (does not require a key for ollama).
func (s *Service) SwitchProvider(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.ToLower(strings.TrimSpace(name))
	if !config.ValidProvider(name) {
		return fmt.Errorf("unknown provider %q", name)
	}
	s.cfg.Provider = name
	modelName := provider.ResolveModel(s.cfg, "")
	if modelName == "" {
		modelName = s.cfg.Model
	}
	s.cfg.SetProviderModel(name, modelName)
	if err := s.cfg.Save(); err != nil {
		s.emit(EvtError, map[string]any{"error": err.Error()})
	}
	s.initAgent()
	s.emit(EvtStatus, map[string]any{"provider": name, "model": modelName, "msg": "Switched to " + name})
	return nil
}

// SetModel sets a model for the active provider.
func (s *Service) SetModel(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("model cannot be empty")
	}
	s.cfg.SetProviderModel(s.cfg.Provider, name)
	if err := s.cfg.Save(); err != nil {
		s.emit(EvtError, map[string]any{"error": err.Error()})
	}
	s.initAgent()
	s.emit(EvtStatus, map[string]any{"provider": s.cfg.Provider, "model": name, "msg": "Using model " + name})
	return nil
}
