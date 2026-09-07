//go:build linux && gtk

package native

import (
	"os"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/go-ducky/gui/internal/guiservice"
)

const cssGoDucky = `
.go-ducky-root { background-color: @theme_bg_color; }
.go-ducky-root.go-light .send-btn { background-color: #141414; color: #ffffff; border-radius: 9999px; }
.go-ducky-root.go-dark .send-btn { background-color: #ffffff; color: #1a1a1a; border-radius: 9999px; }
.go-ducky-root .composer-box { border: 1px solid @borders; border-radius: 16px; }
.sidebar-title { font-weight: 700; font-size: 15px; }
`

type gtkApp struct {
	eng     *Engine
	app     *gtk.Application
	win     *gtk.ApplicationWindow
	rootBox *gtk.Box

	sessionsList *gtk.ListBox
	chat         *gtk.TextView
	chatBuf      *gtk.TextBuffer
	composer     *gtk.TextView
	composerBuf  *gtk.TextBuffer
	sendBtn      *gtk.Button
	stopBtn      *gtk.Button
	thinkingRow  *gtk.Box
	spinner      *gtk.Spinner
	thinkingLbl  *gtk.Label
	statusLabel  *gtk.Label
	modelPill    *gtk.Label
	providerPill *gtk.Label
	sessionTitle *gtk.Label

	providers  []string
	provider   string
	model      string
	workDir    string
	onboarded  bool
	sessions   []guiservice.SessionView
	messages   []guiservice.MsgView
	activeName string
	running    bool
	version    string
	theme      string

	streamBuf strings.Builder
}

func RunGTK() int {
	app := gtk.NewApplication("dev.goducky.gui", gio.ApplicationFlagsNone)
	g := &gtkApp{app: app, theme: "system"}
	app.ConnectActivate(func() {
		g.build()
		go g.consumeEvents()
	})
	return app.Run(os.Args)
}

func escapeMarkup(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

func (g *gtkApp) build() {
	g.eng = New()
	g.version = g.eng.AppVersion()

	applyCss()

	g.win = gtk.NewApplicationWindow(g.app)
	g.win.SetTitle("GoDucky")
	g.win.SetDefaultSize(1150, 780)

	g.rootBox = gtk.NewBox(gtk.OrientationHorizontal, 0)
	g.rootBox.SetCSSClasses([]string{"go-ducky-root", "go-light"})
	g.win.SetChild(g.rootBox)

	sidebar := gtk.NewBox(gtk.OrientationVertical, 6)
	sidebar.SetSizeRequest(270, -1)
	sidebar.SetMarginStart(10)
	sidebar.SetMarginTop(12)
	sidebar.SetMarginBottom(12)

	header := gtk.NewBox(gtk.OrientationHorizontal, 6)
	title := gtk.NewLabel("GoDucky")
	title.SetCSSClasses([]string{"sidebar-title"})
	subtitle := gtk.NewLabel("AI coding agent")
	subtitle.SetCSSClasses([]string{"dim-label"})
	header.Append(title)
	header.Append(subtitle)
	newBtn := gtk.NewButtonFromIconName("document-new-symbolic")
	newBtn.SetTooltipText("New chat")
	newBtn.ConnectClicked(func() {
		if err := g.eng.NewChat(); err == nil {
			g.reload()
		}
	})
	headSpacer := gtk.NewBox(gtk.OrientationVertical, 0)
	headSpacer.SetHExpand(true)
	header.Append(headSpacer)
	header.Append(newBtn)
	sidebar.Append(header)

	g.sessionsList = gtk.NewListBox()
	g.sessionsList.SetSelectionMode(gtk.SelectionSingle)
	g.sessionsList.SetVExpand(true)
	sw := gtk.NewScrolledWindow()
	sw.SetChild(g.sessionsList)
	sw.SetVExpand(true)
	sidebar.Append(sw)

	footer := gtk.NewBox(gtk.OrientationHorizontal, 4)
	settingsBtn := gtk.NewButtonFromIconName("preferences-system-symbolic")
	settingsBtn.SetTooltipText("Settings")
	settingsBtn.ConnectClicked(func() { g.showSettings() })
	themeBtn := gtk.NewButtonFromIconName("display-brightness-symbolic")
	themeBtn.SetTooltipText("Theme")
	themeBtn.ConnectClicked(func() { g.cycleTheme() })
	footer.Append(settingsBtn)
	footer.Append(themeBtn)
	statusSpacer := gtk.NewBox(gtk.OrientationVertical, 0)
	statusSpacer.SetHExpand(true)
	footer.Append(statusSpacer)
	g.statusLabel = gtk.NewLabel("")
	g.statusLabel.SetSelectable(false)
	footer.Append(g.statusLabel)
	sidebar.Append(footer)

	g.rootBox.Append(sidebar)

	main := gtk.NewBox(gtk.OrientationVertical, 0)
	main.SetHExpand(true)
	main.SetVExpand(true)

	top := gtk.NewBox(gtk.OrientationHorizontal, 8)
	top.SetMarginTop(10)
	top.SetMarginEnd(14)
	top.SetMarginStart(14)
	top.SetMarginBottom(4)

	g.sessionTitle = gtk.NewLabel("")
	g.sessionTitle.SetEllipsize(pango.EllipsizeEnd)
	top.Append(g.sessionTitle)

	g.providerPill = gtk.NewLabel("")
	g.providerPill.SetCSSClasses([]string{"dim-label"})
	top.Append(g.providerPill)
	g.modelPill = gtk.NewLabel("")
	g.modelPill.SetCSSClasses([]string{"dim-label"})
	top.Append(g.modelPill)

	spacer := gtk.NewBox(gtk.OrientationVertical, 0)
	spacer.SetHExpand(true)
	top.Append(spacer)

	shareBtn := gtk.NewButtonFromIconName("document-send-symbolic")
	shareBtn.SetTooltipText("Share this chat (copy transcript)")
	shareBtn.ConnectClicked(func() { g.shareActive() })
	top.Append(shareBtn)

	saveBtn := gtk.NewButtonFromIconName("document-save-symbolic")
	saveBtn.SetTooltipText("Save chat")
	saveBtn.ConnectClicked(func() { g.saveActive() })
	top.Append(saveBtn)
	main.Append(top)

	main.Append(gtk.NewSeparator(gtk.OrientationHorizontal))

	g.chat = gtk.NewTextView()
	g.chat.SetEditable(false)
	g.chat.SetCursorVisible(true)
	g.chat.SetWrapMode(gtk.WrapWordChar)
	g.chat.SetTopMargin(12)
	g.chat.SetLeftMargin(20)
	g.chat.SetRightMargin(20)
	g.chat.SetBottomMargin(12)
	g.chatBuf = g.chat.Buffer()

	chatScroller := gtk.NewScrolledWindow()
	chatScroller.SetChild(g.chat)
	chatScroller.SetVExpand(true)
	chatScroller.SetHExpand(true)
	main.Append(chatScroller)

	g.thinkingRow = gtk.NewBox(gtk.OrientationHorizontal, 8)
	g.thinkingRow.SetMarginStart(24)
	g.thinkingRow.SetMarginTop(4)
	g.spinner = gtk.NewSpinner()
	g.spinner.SetSizeRequest(16, 16)
	g.thinkingLbl = gtk.NewLabel("Thinking")
	g.thinkingLbl.SetCSSClasses([]string{"dim-label"})
	g.thinkingRow.Append(g.spinner)
	g.thinkingRow.Append(g.thinkingLbl)
	g.thinkingRow.SetVisible(false)
	main.Append(g.thinkingRow)

	composerWrap := gtk.NewBox(gtk.OrientationVertical, 6)
	composerWrap.SetMarginStart(14)
	composerWrap.SetMarginEnd(14)
	composerWrap.SetMarginBottom(14)

	composerBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	composerBox.SetCSSClasses([]string{"composer-box"})
	composerBox.SetMarginTop(4)
	composerBox.SetMarginBottom(4)
	g.composer = gtk.NewTextView()
	g.composer.SetWrapMode(gtk.WrapWordChar)
	g.composer.SetTopMargin(9)
	g.composer.SetBottomMargin(9)
	g.composer.SetLeftMargin(12)
	g.composer.SetRightMargin(4)
	g.composer.SetHExpand(true)
	g.composer.SetSizeRequest(-1, 46)
	g.composer.SetTooltipText("Enter sends, Shift+Enter for a new line")
	g.composerBuf = g.composer.Buffer()

	g.sendBtn = gtk.NewButtonFromIconName("go-up-symbolic")
	g.sendBtn.SetCSSClasses([]string{"send-btn"})
	g.sendBtn.SetSizeRequest(38, 38)
	g.sendBtn.SetTooltipText("Send")
	g.sendBtn.ConnectClicked(func() { g.send() })

	g.stopBtn = gtk.NewButtonFromIconName("media-playback-stop-symbolic")
	g.stopBtn.SetCSSClasses([]string{"send-btn"})
	g.stopBtn.SetSizeRequest(38, 38)
	g.stopBtn.SetTooltipText("Stop generating")
	g.stopBtn.ConnectClicked(func() { g.stop() })
	g.stopBtn.SetVisible(false)

	composerBox.Append(g.composer)
	composerBox.Append(g.sendBtn)
	composerBox.Append(g.stopBtn)
	composerWrap.Append(composerBox)

	hint := gtk.NewLabel("GoDucky can make mistakes. Check important info.")
	hint.SetHAlign(gtk.AlignStart)
	hint.SetCSSClasses([]string{"dim-label"})
	composerWrap.Append(hint)

	main.Append(composerWrap)
	g.rootBox.Append(main)

	ctrl := gtk.NewEventControllerKey()
	ctrl.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		if (keyval == gdk.KEY_Return || keyval == gdk.KEY_KP_Enter) && state&gdk.ShiftMask == 0 {
			g.send()
			return true
		}
		return false
	})
	g.composer.AddController(ctrl)

	g.win.Show()
	g.reload()
}

func (g *gtkApp) send() {
	text := g.composerText()
	if strings.TrimSpace(text) == "" {
		return
	}
	g.messages = append(g.messages, guiservice.MsgView{Role: "user", Text: text})
	g.renderChat()
	g.clearComposer()
	g.setRunning(true)
	if err := g.eng.Send(text); err != nil {
		g.setRunning(false)
		g.setStatus("Error: " + err.Error())
	}
}

func (g *gtkApp) stop() { g.eng.Stop() }

func (g *gtkApp) composerText() string {
	return g.composerBuf.Text(g.composerBuf.StartIter(), g.composerBuf.EndIter(), true)
}

func (g *gtkApp) clearComposer() {
	g.composerBuf.Delete(g.composerBuf.StartIter(), g.composerBuf.EndIter())
}

func (g *gtkApp) setRunning(run bool) {
	g.running = run
	g.sendBtn.SetVisible(!run)
	g.stopBtn.SetVisible(run)
	g.thinkingRow.SetVisible(run)
	if run {
		g.spinner.Start()
	} else {
		g.spinner.Stop()
	}
}

func (g *gtkApp) setStatus(msg string) {
	g.statusLabel.SetText(msg)
}

func (g *gtkApp) scrollToEnd() {
	g.chat.ScrollToIter(g.chatBuf.EndIter(), 0.0, false, 0.0, 0.0)
}

func (g *gtkApp) renderChat() {
	buf := g.chatBuf
	buf.Delete(buf.StartIter(), buf.EndIter())
	for i, m := range g.messages {
		if i > 0 {
			buf.Insert(buf.EndIter(), "\n")
		}
		switch m.Role {
		case "user":
			buf.InsertMarkup(buf.EndIter(), "<b>You</b>\n")
			buf.InsertMarkup(buf.EndIter(), "<span weight='bold'>"+escapeMarkup(m.Text)+"</span>")
		case "system":
			buf.InsertMarkup(buf.EndIter(), "<span color='#98989f'>"+escapeMarkup(m.Text)+"</span>")
		default:
			buf.Insert(buf.EndIter(), escapeMarkup(m.Text))
		}
		buf.Insert(buf.EndIter(), "\n")
	}
	g.scrollToEnd()
}

func (g *gtkApp) reload() {
	info := g.eng.GetInfo()
	if info == nil {
		return
	}
	g.provider = info.Provider
	g.model = info.Model
	g.workDir = info.WorkDir
	g.onboarded = info.Onboarded
	g.activeName = info.Name
	g.messages = info.Messages
	g.sessions = info.Sessions

	g.providerPill.SetText(g.provider)
	if g.model != "" {
		g.modelPill.SetText(g.model)
	}
	g.sessionTitle.SetText(g.activeName)

	for {
		child := g.sessionsList.FirstChild()
		if child == nil {
			break
		}
		g.sessionsList.Remove(child)
	}
	for _, s := range g.sessions {
		g.sessionsList.Append(g.newSessionRow(s))
	}
	g.renderChat()
	g.setRunning(g.eng.IsRunning())

	if !g.onboarded {
		g.showOnboarding()
	}
}

func (g *gtkApp) newSessionRow(s guiservice.SessionView) *gtk.ListBoxRow {
	box := gtk.NewBox(gtk.OrientationHorizontal, 6)
	labels := gtk.NewBox(gtk.OrientationVertical, 0)
	name := gtk.NewLabel(s.Name)
	name.SetHAlign(gtk.AlignStart)
	name.SetEllipsize(pango.EllipsizeEnd)
	preview := gtk.NewLabel(s.Preview)
	preview.SetHAlign(gtk.AlignStart)
	preview.SetCSSClasses([]string{"dim-label"})
	preview.SetEllipsize(pango.EllipsizeEnd)
	labels.Append(name)
	labels.Append(preview)
	labels.SetHExpand(true)
	box.Append(labels)

	shareBtn := gtk.NewButtonFromIconName("document-send-symbolic")
	shareBtn.SetTooltipText("Share")
	shareBtn.SetSizeRequest(26, 26)
	shareBtn.ConnectClicked(func() {
		if err := g.eng.ShareSession(s.Name); err == nil {
			g.setStatus("Shared " + s.Name)
		} else {
			g.setStatus(err.Error())
		}
	})
	delBtn := gtk.NewButtonFromIconName("edit-delete-symbolic")
	delBtn.SetTooltipText("Delete")
	delBtn.SetSizeRequest(26, 26)
	delBtn.ConnectClicked(func() { g.confirmDelete(s.Name) })
	renBtn := gtk.NewButtonFromIconName("document-edit-symbolic")
	renBtn.SetTooltipText("Rename")
	renBtn.SetSizeRequest(26, 26)
	renBtn.ConnectClicked(func() { g.promptRename(s.Name) })
	actBox := gtk.NewBox(gtk.OrientationHorizontal, 2)
	actBox.Append(renBtn)
	actBox.Append(shareBtn)
	actBox.Append(delBtn)
	box.Append(actBox)

	row := gtk.NewListBoxRow()
	row.SetChild(box)
	row.SetActivatable(true)
	row.ConnectActivate(func() {
		if err := g.eng.Resume(s.Name); err == nil {
			g.reload()
		} else {
			g.setStatus(err.Error())
		}
	})
	return row
}

func (g *gtkApp) consumeEvents() {
	for ev := range g.eng.Events() {
		switch ev.Name {
		case EvtStream:
			if data, ok := ev.Data.(map[string]any); ok {
				if text, ok := data["text"].(string); ok {
					g.streamBuf.WriteString(text)
					glib.IdleAdd(func() {
						g.messages = append(g.messages, guiservice.MsgView{Role: "assistant", Text: g.streamBuf.String()})
						g.streamBuf.Reset()
						g.renderChat()
					})
				}
			}
		case EvtStatus:
			if data, ok := ev.Data.(map[string]any); ok {
				if msg, ok := data["msg"].(string); ok {
					glib.IdleAdd(func() { g.setStatus(msg) })
				}
			}
		case EvtComplete:
			glib.IdleAdd(func() {
				g.setRunning(false)
				g.reload()
			})
		case EvtError:
			if data, ok := ev.Data.(map[string]any); ok {
				if msg, ok := data["error"].(string); ok {
					msgCopy := msg
					glib.IdleAdd(func() {
						g.setRunning(false)
						g.setStatus("Error: " + msgCopy)
					})
				}
			}
		case EvtApproval:
			var req guiservice.ApprovalRequest
			switch v := ev.Data.(type) {
			case guiservice.ApprovalRequest:
				req = v
			case map[string]any:
				req = guiservice.ApprovalRequest{
					ID:   toString(v["id"]),
					Desc: toString(v["desc"]),
				}
			}
			if req.ID != "" {
				glib.IdleAdd(func() { g.showApproval(req) })
			}
		}
	}
}

func (g *gtkApp) shareActive() {
	if err := g.eng.ShareSession(g.activeName); err != nil {
		g.setStatus("Share failed: " + err.Error())
		return
	}
	g.setStatus("Conversation copied to clipboard")
}

func (g *gtkApp) saveActive() {
	if err := g.eng.SaveSession(g.activeName); err != nil {
		g.setStatus("Save failed: " + err.Error())
		return
	}
	g.setStatus("Saved")
}

func (g *gtkApp) dialog(title string, w, h int, content gtk.Widgetter) *gtk.Window {
	wnd := gtk.NewWindow()
	wnd.SetTitle(title)
	wnd.SetTransientFor(&g.win.Window)
	wnd.SetModal(true)
	wnd.SetResizable(false)
	wnd.SetDefaultSize(w, h)
	box := gtk.NewBox(gtk.OrientationVertical, 12)
	box.SetMarginTop(18)
	box.SetMarginBottom(18)
	box.SetMarginStart(18)
	box.SetMarginEnd(18)
	box.Append(content)
	wnd.SetChild(box)
	wnd.Show()
	return wnd
}

func (g *gtkApp) confirmDelete(name string) {
	content := gtk.NewBox(gtk.OrientationVertical, 12)
	lbl := gtk.NewLabel("Delete \"" + name + "\"? This cannot be undone.")
	lbl.SetWrap(true)
	content.Append(lbl)
	row := gtk.NewBox(gtk.OrientationHorizontal, 8)
	row.SetHAlign(gtk.AlignEnd)
	cancel := gtk.NewButtonWithLabel("Cancel")
	del := gtk.NewButtonWithLabel("Delete")
	del.AddCSSClass("destructive-action")
	del.ConnectClicked(func() {
		if g.activeName == name {
			g.messages = nil
			g.activeName = ""
		}
		_ = g.eng.DeleteSession(name)
		g.reload()
	})
	row.Append(cancel)
	row.Append(del)
	content.Append(row)
	wnd := g.dialog("Delete chat", 380, 140, content)
	cancel.ConnectClicked(func() { wnd.Close() })
	del.ConnectClicked(func() { wnd.Close() })
}

func (g *gtkApp) promptRename(oldName string) {
	content := gtk.NewBox(gtk.OrientationVertical, 12)
	entry := gtk.NewEntry()
	entry.SetText(oldName)
	content.Append(entry)
	row := gtk.NewBox(gtk.OrientationHorizontal, 8)
	row.SetHAlign(gtk.AlignEnd)
	cancel := gtk.NewButtonWithLabel("Cancel")
	ok := gtk.NewButtonWithLabel("Rename")
	ok.ConnectClicked(func() {
		_ = g.eng.RenameSession(oldName, entry.Text())
		g.reload()
	})
	row.Append(cancel)
	row.Append(ok)
	content.Append(row)
	wnd := g.dialog("Rename chat", 340, 110, content)
	cancel.ConnectClicked(func() { wnd.Close() })
	ok.ConnectClicked(func() { wnd.Close() })
}

func (g *gtkApp) showApproval(req guiservice.ApprovalRequest) {
	content := gtk.NewBox(gtk.OrientationVertical, 12)
	lbl := gtk.NewLabel(req.Desc)
	lbl.SetWrap(true)
	content.Append(lbl)
	row := gtk.NewBox(gtk.OrientationHorizontal, 8)
	row.SetHAlign(gtk.AlignEnd)
	deny := gtk.NewButtonWithLabel("Deny")
	allow := gtk.NewButtonWithLabel("Allow")
	allow.AddCSSClass("suggested-action")
	allow.ConnectClicked(func() { g.eng.Approve(req.ID, true) })
	deny.ConnectClicked(func() { g.eng.Approve(req.ID, false) })
	row.Append(deny)
	row.Append(allow)
	content.Append(row)
	wnd := g.dialog("Approve action", 440, 170, content)
	deny.ConnectClicked(func() { wnd.Close() })
	allow.ConnectClicked(func() { wnd.Close() })
}

func (g *gtkApp) showSettings() {
	content := gtk.NewBox(gtk.OrientationVertical, 12)

	wdTitle := gtk.NewLabel("Working directory")
	wdTitle.SetHAlign(gtk.AlignStart)
	content.Append(wdTitle)
	wdEntry := gtk.NewEntry()
	wdEntry.SetText(g.workDir)
	content.Append(wdEntry)

	apToggle := gtk.NewCheckButtonWithLabel("Auto-approve file & command actions")
	apToggle.SetActive(g.eng.AutoApprove())
	content.Append(apToggle)

	providers := g.eng.Providers()
	provDrop := gtk.NewDropDownFromStrings(providers)
	provDrop.SetSelected(uint(indexOf(g.provider, providers, 0)))
	pTitle := gtk.NewLabel("Provider")
	pTitle.SetHAlign(gtk.AlignStart)
	content.Append(pTitle)
	content.Append(provDrop)

	models := g.eng.GetModels(g.provider)
	modelDrop := gtk.NewDropDownFromStrings(models)
	modelDrop.SetSelected(uint(indexOf(g.model, models, 0)))
	mTitle := gtk.NewLabel("Model")
	mTitle.SetHAlign(gtk.AlignStart)
	content.Append(mTitle)
	content.Append(modelDrop)

	keyBtn := gtk.NewButtonWithLabel("Set API key…")
	keyBtn.ConnectClicked(func() { g.showApiKey() })
	content.Append(keyBtn)

	row := gtk.NewBox(gtk.OrientationHorizontal, 8)
	row.SetHAlign(gtk.AlignEnd)
	cancel := gtk.NewButtonWithLabel("Close")
	apply := gtk.NewButtonWithLabel("Apply")
	apply.AddCSSClass("suggested-action")
	apply.ConnectClicked(func() {
		if p := wdEntry.Text(); p != "" {
			_ = g.eng.SetWorkDir(p)
		}
		_ = g.eng.SetAutoApprove(apToggle.Active())
		if sel := int(provDrop.Selected()); sel >= 0 && sel < len(providers) {
			if providers[sel] != g.provider {
				_ = g.eng.SwitchProvider(providers[sel])
			}
		}
		if sel := int(modelDrop.Selected()); sel >= 0 && sel < len(models) && models[sel] != "" {
			_ = g.eng.SetModel(models[sel])
		}
		g.reload()
	})
	row.Append(cancel)
	row.Append(apply)
	content.Append(row)

	wnd := g.dialog("Settings", 460, 460, content)
	cancel.ConnectClicked(func() { wnd.Close() })
	apply.ConnectClicked(func() { wnd.Close() })
}

func (g *gtkApp) showApiKey() {
	providers := []string{"groq", "openai", "openrouter", "anthropic", "gemini"}
	content := gtk.NewBox(gtk.OrientationVertical, 12)
	provDrop := gtk.NewDropDownFromStrings(providers)
	pTitle := gtk.NewLabel("Provider")
	pTitle.SetHAlign(gtk.AlignStart)
	content.Append(pTitle)
	content.Append(provDrop)

	keyEntry := gtk.NewEntry()
	keyEntry.SetVisibility(false)
	keyEntry.SetPlaceholderText("API key (Ollama doesn't need one)")
	kTitle := gtk.NewLabel("API key")
	kTitle.SetHAlign(gtk.AlignStart)
	content.Append(kTitle)
	content.Append(keyEntry)

	row := gtk.NewBox(gtk.OrientationHorizontal, 8)
	row.SetHAlign(gtk.AlignEnd)
	cancel := gtk.NewButtonWithLabel("Cancel")
	ok := gtk.NewButtonWithLabel("Save")
	ok.AddCSSClass("suggested-action")
	ok.ConnectClicked(func() {
		p := providers[provDrop.Selected()]
		if key := keyEntry.Text(); key != "" {
			if err := g.eng.SaveAPIKey(p, key); err != nil {
				g.setStatus("Key error: " + err.Error())
			} else {
				g.setStatus("API key saved for " + p)
			}
		}
		g.reload()
	})
	row.Append(cancel)
	row.Append(ok)
	content.Append(row)

	wnd := g.dialog("API key", 440, 280, content)
	cancel.ConnectClicked(func() { wnd.Close() })
	ok.ConnectClicked(func() { wnd.Close() })
}

func (g *gtkApp) showOnboarding() {
	content := gtk.NewBox(gtk.OrientationVertical, 12)
	welcome := gtk.NewLabel("Welcome to GoDucky")
	welcome.SetHAlign(gtk.AlignStart)
	welcome.AddCSSClass("title-1")
	content.Append(welcome)
	body := gtk.NewLabel("Pick how you want to run your coding agent.")
	body.SetHAlign(gtk.AlignStart)
	body.SetWrap(true)
	content.Append(body)

	localBtn := gtk.NewButtonWithLabel("Use local models (Ollama)")
	cloudBtn := gtk.NewButtonWithLabel("Use a cloud provider (API key)")
	skipBtn := gtk.NewButtonWithLabel("Skip for now")
	content.Append(localBtn)
	content.Append(cloudBtn)
	content.Append(skipBtn)

	wnd := g.dialog("Set up GoDucky", 520, 240, content)
	localBtn.ConnectClicked(func() {
		wnd.Close()
		g.showOnboardingLocal()
	})
	cloudBtn.ConnectClicked(func() {
		wnd.Close()
		g.showOnboardingCloud()
	})
	skipBtn.ConnectClicked(func() {
		_ = g.eng.ApplySetup()
		g.reload()
		wnd.Close()
	})
}

func (g *gtkApp) showOnboardingLocal() {
	content := gtk.NewBox(gtk.OrientationVertical, 10)
	title := gtk.NewLabel("Local models (Ollama)")
	title.SetHAlign(gtk.AlignStart)
	content.Append(title)
	sub := gtk.NewLabel("Pick models to pull for local reasoning.")
	sub.SetHAlign(gtk.AlignStart)
	content.Append(sub)

	toggles := map[string]*gtk.CheckButton{}
	for _, m := range g.eng.RecommendedLocalModels() {
		tb := gtk.NewCheckButtonWithLabel(m)
		tb.SetActive(true)
		toggles[m] = tb
		content.Append(tb)
	}

	installBtn := gtk.NewButtonWithLabel("Install Ollama…")
	installBtn.ConnectClicked(func() {
		g.eng.InstallOllama()
		g.setStatus("Installing Ollama in the background…")
	})
	row := gtk.NewBox(gtk.OrientationHorizontal, 8)
	row.SetHAlign(gtk.AlignEnd)
	cancel := gtk.NewButtonWithLabel("Back")
	cont := gtk.NewButtonWithLabel("Continue")
	cont.AddCSSClass("suggested-action")
	cont.ConnectClicked(func() {
		var chosen []string
		for m, tb := range toggles {
			if tb.Active() {
				chosen = append(chosen, m)
			}
		}
		if len(chosen) > 0 {
			g.eng.StartLocalModelPulls(chosen)
		}
		_ = g.eng.SwitchProvider("ollama")
		_ = g.eng.ApplySetup()
		g.reload()
		g.setStatus("Pulling local models…")
	})
	row.Append(installBtn)
	row.Append(cancel)
	row.Append(cont)
	content.Append(row)

	wnd := g.dialog("Local models", 460, 400, content)
	cancel.ConnectClicked(func() {
		wnd.Close()
		g.showOnboarding()
	})
	cont.ConnectClicked(func() { wnd.Close() })
}

func (g *gtkApp) showOnboardingCloud() {
	providers := []string{"groq", "openai", "openrouter", "anthropic", "gemini"}
	content := gtk.NewBox(gtk.OrientationVertical, 10)
	title := gtk.NewLabel("Cloud provider")
	title.SetHAlign(gtk.AlignStart)
	content.Append(title)
	provDrop := gtk.NewDropDownFromStrings(providers)
	content.Append(provDrop)

	keyEntry := gtk.NewEntry()
	keyEntry.SetVisibility(false)
	keyEntry.SetPlaceholderText("Paste your API key")
	pTitle := gtk.NewLabel("API key")
	pTitle.SetHAlign(gtk.AlignStart)
	content.Append(pTitle)
	content.Append(keyEntry)

	row := gtk.NewBox(gtk.OrientationHorizontal, 8)
	row.SetHAlign(gtk.AlignEnd)
	back := gtk.NewButtonWithLabel("Back")
	cont := gtk.NewButtonWithLabel("Continue")
	cont.AddCSSClass("suggested-action")
	cont.ConnectClicked(func() {
		p := providers[provDrop.Selected()]
		key := keyEntry.Text()
		if key == "" {
			g.setStatus("Enter an API key for " + p)
			return
		}
		if err := g.eng.SaveAPIKey(p, key); err != nil {
			g.setStatus("Key error: " + err.Error())
			return
		}
		_ = g.eng.ApplySetup()
		g.reload()
		g.setStatus("Using " + p)
	})
	row.Append(back)
	row.Append(cont)
	content.Append(row)

	wnd := g.dialog("Cloud provider", 460, 320, content)
	back.ConnectClicked(func() {
		wnd.Close()
		g.showOnboarding()
	})
	cont.ConnectClicked(func() { wnd.Close() })
}

func (g *gtkApp) cycleTheme() {
	order := []string{"system", "light", "dark"}
	next := "system"
	for i, t := range order {
		if t == g.theme {
			next = order[(i+1)%len(order)]
			break
		}
	}
	g.theme = next
	classes := []string{"go-ducky-root"}
	switch g.theme {
	case "dark":
		classes = append(classes, "go-dark")
	case "light":
		classes = append(classes, "go-light")
	}
	g.rootBox.SetCSSClasses(classes)
	g.setStatus("Theme: " + g.theme)
}

func applyCss() {
	provider := gtk.NewCSSProvider()
	provider.LoadFromData(cssGoDucky)
	if display := gdk.DisplayGetDefault(); display != nil {
		gtk.StyleContextAddProviderForDisplay(display, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
	}
}
