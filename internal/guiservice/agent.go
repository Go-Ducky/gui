package guiservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-ducky/gui/internal/agent"
	"github.com/go-ducky/gui/internal/agent/tools"
	"github.com/go-ducky/gui/internal/config"
	"github.com/go-ducky/gui/internal/provider"
	"github.com/go-ducky/gui/internal/session"
	"github.com/go-ducky/gui/internal/setup"
)

// Send starts a new agent turn with the given user prompt. It returns
// immediately; streaming output is pushed to the event sink.
func (s *Service) Send(prompt string) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("an agent turn is already running")
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		s.mu.Unlock()
		return errors.New("empty prompt")
	}
	if s.agent == nil {
		s.initAgent()
	}
	agent := s.agent
	cfg := s.cfg
	workDir := s.workDir

	// Seed the config for a fresh agent build if the user never started.
	s.mu.Unlock()

	// Append the user message locally.
	s.mu.Lock()
	s.messages = append(s.messages, provider.NewTextMessage(provider.RoleUser, prompt))
	s.running = true
	ctx, cancel := context.WithCancel(context.Background())
	s.runCtx = ctx
	s.cancelRun = cancel
	s.mu.Unlock()
	// Ensure agent picks up settings like auto-approve at run time.
	s.setAutoApprove()

	history := make([]provider.Message, len(s.messages))
	copy(history, s.messages)

	go s.runAgent(agent, workDir, cfg, history, ctx)
	return nil
}

func (s *Service) setAutoApprove() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.agent != nil {
		s.agent.SetAutoApprove(s.cfg.Agent.AutoApprove)
	}
}

// Stop cancels the in-flight agent turn.
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancelRun != nil {
		s.cancelRun()
		s.cancelRun = nil
	}
}

// IsRunning reports whether an agent turn is in progress.
func (s *Service) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

type ApprovalRequest struct {
	ID   string         `json:"id"`
	Desc string         `json:"desc"`
	Args map[string]any `json:"args"`
}

func (s *Service) runAgent(a *agent.Agent, workDir string, cfg *config.Config, history []provider.Message, ctx context.Context) {
	cb := &guiCallback{s: s}
	a.SetApprover(func(desc string, args map[string]any) bool {
		id := fmt.Sprintf("ap-%d", time.Now().UnixNano())
		ch := make(chan bool, 1)
		s.mu.Lock()
		s.approvalPending = true
		s.approvalRespond = ch
		s.mu.Unlock()
		s.emit(EvtApproval, ApprovalRequest{ID: id, Desc: approvalLabel(desc, args), Args: args})
		select {
		case ok := <-ch:
			return ok
		case <-ctx.Done():
			return false
		case <-time.After(10 * time.Minute):
			return false
		}
	})

	result, _, err := a.Run(ctx, history, cb)

	s.mu.Lock()
	s.running = false
	s.approvalPending = false
	s.approvalRespond = nil
	s.autosaveLocked()
	s.mu.Unlock()

	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			s.emit(EvtComplete, map[string]any{"stopped": true, "text": result})
			return
		}
		s.emit(EvtError, map[string]any{"error": err.Error()})
		return
	}
	s.emit(EvtComplete, map[string]any{"stopped": false, "text": result})
}

// Approve answers a pending tool-approval prompt.
func (s *Service) Approve(id string, ok bool) bool {
	s.mu.Lock()
	ch := s.approvalRespond
	s.mu.Unlock()
	if ch == nil {
		return false
	}
	select {
	case ch <- ok:
		return true
	default:
		return false
	}
}

// OllamaStatus reports whether the Ollama server is installed and running.
func (s *Service) OllamaStatus() map[string]any {
	return map[string]any{
		"installed": setup.IsOllamaInstalled(),
		"running":   setup.IsOllamaRunning(),
	}
}

// PullModel pulls (downloads) a model through Ollama in the background.
func (s *Service) PullModel(model string) {
	go func() {
		o := provider.NewOllama(s.cfg)
		err := o.Pull(context.Background(), model)
		s.emit(EvtOllamaOp, map[string]any{"action": "pull", "model": model, "error": errString(err)})
	}()
}

// RemoveModel removes a locally pulled Ollama model.
func (s *Service) RemoveModel(model string) {
	go func() {
		o := provider.NewOllama(s.cfg)
		err := o.Remove(context.Background(), model)
		s.emit(EvtOllamaOp, map[string]any{"action": "rm", "model": model, "error": errString(err)})
	}()
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// HasModel checks whether the given Ollama model is already pulled.
func (s *Service) HasModel(model string) bool {
	o := provider.NewOllama(s.cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return o.HasModel(ctx, model)
}

func approvalLabel(desc string, args map[string]any) string {
	var sb strings.Builder
	sb.WriteString(desc)
	if len(args) > 0 {
		sb.WriteString(" ")
		parts := make([]string, 0, len(args))
		for k, v := range args {
			vs := fmt.Sprintf("%v", v)
			if len([]rune(vs)) > 100 {
				vs = string([]rune(vs)[:100]) + "…"
			}
			parts = append(parts, k+"="+vs)
		}
		sb.WriteString(strings.Join(parts, " "))
	}
	return sb.String()
}

// SaveSession saves (or overwrites) the current chat under the given name.
// SaveSession writes the current conversation to disk. An empty name gets a
// generated one.
func (s *Service) SaveSession(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.messages) == 0 {
		return errors.New("nothing to save yet")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = session.AutoName()
	}
	ss := &session.Session{
		Name:     name,
		Provider: s.cfg.Provider,
		Model:    provider.ResolveModel(s.cfg, ""),
		WorkDir:  s.workDir,
		Messages: s.messages,
	}
	if err := session.Save(ss); err != nil {
		return err
	}
	s.sessionName = name
	s.emit(EvtStatus, map[string]any{"msg": "Chat saved as " + name})
	return nil
}

// autosaveLocked persists the current conversation without emitting events.
// Must be called with s.mu held.
func (s *Service) autosaveLocked() {
	if len(s.messages) == 0 {
		return
	}
	name := strings.TrimSpace(s.sessionName)
	if name == "" {
		name = session.AutoName()
	}
	ss := &session.Session{
		Name:     name,
		Provider: s.cfg.Provider,
		Model:    provider.ResolveModel(s.cfg, ""),
		WorkDir:  s.workDir,
		Messages: s.messages,
	}
	if err := session.Save(ss); err == nil {
		s.sessionName = name
	}
}

// NewChat clears the current conversation, auto-saving it first if non-empty.
func (s *Service) NewChat() error {
	s.mu.Lock()
	if len(s.messages) == 0 {
		s.messages = nil
		s.sessionName = ""
		s.mu.Unlock()
		return nil
	}
	// Auto-save the current chat before starting fresh.
	ss := &session.Session{
		Name:     s.sessionName,
		Provider: s.cfg.Provider,
		Model:    provider.ResolveModel(s.cfg, ""),
		WorkDir:  s.workDir,
		Messages: s.messages,
	}
	if s.sessionName != "" {
		s.messages = nil
		s.sessionName = ""
		s.mu.Unlock()
		_ = session.Save(ss)
		return nil
	}
	s.messages = nil
	s.sessionName = ""
	s.mu.Unlock()
	ss.Name = session.AutoName()
	_ = session.Save(ss)
	return nil
}

// ListSessions returns the saved chats, newest first.
func (s *Service) ListSessions() []SessionView {
	ss, err := session.List()
	if err != nil {
		return []SessionView{}
	}
	return buildSessionViews(ss)
}

// Resume loads a saved chat into the working conversation.
func (s *Service) Resume(nameOrNum string) error {
	ss, err := session.Load(nameOrNum)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if len(s.messages) > 0 && s.sessionName == "" {
		old := &session.Session{
			Name:     session.AutoName(),
			Provider: s.cfg.Provider,
			Model:    provider.ResolveModel(s.cfg, ""),
			WorkDir:  s.workDir,
			Messages: s.messages,
		}
		_ = session.Save(old)
	}
	s.sessionName = ss.Name
	s.messages = ss.Messages
	s.cfg.Provider = ss.Provider
	s.cfg.SetProviderModel(ss.Provider, ss.Model)
	if ss.WorkDir != "" {
		if err := os.MkdirAll(ss.WorkDir, 0o755); err == nil {
			s.workDir = ss.WorkDir
		}
	}
	s.mu.Unlock()
	_ = s.cfg.Save()
	s.initAgent()
	s.emit(EvtStatus, map[string]any{"msg": "Resumed chat " + ss.Name})
	return nil
}

// RenameSession renames a saved chat.
func (s *Service) RenameSession(oldName, newName string) error {
	if err := session.Rename(oldName, newName); err != nil {
		return err
	}
	s.mu.Lock()
	if s.sessionName == oldName {
		s.sessionName = newName
	}
	s.mu.Unlock()
	return nil
}

// DeleteSession deletes a saved chat.
func (s *Service) DeleteSession(name string) error {
	ss, err := session.Load(name)
	if err != nil {
		return err
	}
	path, err := session.PathFor(ss.Name)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if s.sessionName == ss.Name {
		s.sessionName = ""
		s.messages = nil
	}
	s.mu.Unlock()
	return os.Remove(path)
}

// SaveAPIKey validates and saves an API key for a provider, then switches to it.
func (s *Service) SaveAPIKey(providerName, key string) error {
	providerName = strings.ToLower(strings.TrimSpace(providerName))
	key = strings.TrimSpace(key)
	if !config.ValidProvider(providerName) {
		return errors.New("unknown provider")
	}
	if key == "" {
		return errors.New("no key entered")
	}
	verified := true
	if err := provider.ValidateAPIKey(providerName, key); err != nil {
		if strings.Contains(err.Error(), "rejected") {
			return err
		}
		verified = false
	}
	auth, err := config.LoadAuth()
	if err != nil {
		return err
	}
	setAuthKey(auth, providerName, key)
	if err := auth.Save(); err != nil {
		return err
	}
	s.mu.Lock()
	s.cfg.Provider = providerName
	s.mu.Unlock()
	_ = s.cfg.Save()
	s.initAgent()
	msg := "API key for " + providerName + " saved."
	if !verified {
		msg += " (couldn't verify — offline?)"
	}
	s.emit(EvtStatus, map[string]any{"msg": msg})
	return nil
}

// HasAPIKey reports whether a key is available for the provider.
func (s *Service) HasAPIKey(providerName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	var akey, env string
	switch providerName {
	case "groq":
		akey, env = s.auth.GroqAPIKey, s.cfg.Groq.EnvKey
	case "openai", "openai_compatible":
		akey, env = s.auth.OpenAIAPIKey, s.cfg.OpenAI.EnvKey
	case "openrouter":
		akey, env = s.auth.OpenRouterAPIKey, s.cfg.OpenRouter.EnvKey
	case "anthropic":
		akey, env = s.auth.AnthropicAPIKey, s.cfg.Anthropic.EnvKey
	case "gemini":
		akey, env = s.auth.GeminiAPIKey, s.cfg.Gemini.EnvKey
	}
	return akey != "" || (env != "" && os.Getenv(env) != "")
}

// SetAutoApprove toggles auto-approval of file/command actions.
func (s *Service) SetAutoApprove(on bool) error {
	s.cfg.Agent.AutoApprove = on
	s.setAutoApprove()
	return s.cfg.Save()
}

// AutoApprove reports whether auto-approval is enabled.
func (s *Service) AutoApprove() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.Agent.AutoApprove
}

// SetConfigValue sets a dotted config key (like the CLI's /config).
func (s *Service) SetConfigValue(key, value string) error {
	if err := s.cfg.Set(key, value); err != nil {
		return err
	}
	if err := s.cfg.Save(); err != nil {
		return err
	}
	s.setAutoApprove()
	s.initAgent()
	s.emit(EvtStatus, map[string]any{"msg": "Saved config: " + key + " = " + value})
	return nil
}

// ApplySetup finishes the first-run wizard: mark onboarded, save config, init agent.
func (s *Service) ApplySetup() error {
	s.mu.Lock()
	s.cfg.Onboarded = true
	s.mu.Unlock()
	_ = s.cfg.Save()
	s.initAgent()
	return nil
}

// EnsureOllamaModel pulls a model through Ollama if it isn't local yet.
func (s *Service) EnsureOllamaModel(model string) {
	if s.HasModel(model) {
		return
	}
	s.PullModel(model)
}

func setAuthKey(a *config.Auth, prov, key string) {
	switch prov {
	case "groq":
		a.GroqAPIKey = key
	case "openai", "openai_compatible":
		a.OpenAIAPIKey = key
	case "anthropic":
		a.AnthropicAPIKey = key
	case "gemini":
		a.GeminiAPIKey = key
	case "openrouter":
		a.OpenRouterAPIKey = key
	}
}

// guiCallback streams agent activity to the event sink (owned by the native UI).
type guiCallback struct {
	s *Service
}

func (c *guiCallback) OnText(text string)                           { c.s.emit(EvtStream, map[string]any{"text": text}) }
func (c *guiCallback) OnStatus(msg string)                          { c.s.emit(EvtStatus, map[string]any{"msg": msg}) }
func (c *guiCallback) OnComplete(resp string, usage provider.Usage) {}
func (c *guiCallback) OnToolStart(name string, args json.RawMessage) {
	c.s.emit(EvtToolStart, map[string]any{"name": name, "args": string(args)})
}
func (c *guiCallback) OnToolEnd(name string, result *tools.Result) {
	c.s.emit(EvtToolEnd, map[string]any{"name": name, "content": result.Content, "is_error": result.IsError})
}
