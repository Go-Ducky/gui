//go:build linux && qt

package native

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/go-ducky/gui/internal/guiservice"
	"github.com/mappu/miqt/qt6"
)

type qtApp struct {
	eng *Engine

	win          *qt6.QMainWindow
	sessionsList *qt6.QListWidget
	chat         *qt6.QTextEdit
	composer     *qt6.QPlainTextEdit
	sendBtn      *qt6.QPushButton
	stopBtn      *qt6.QPushButton
	thinkingLbl  *qt6.QLabel
	sessionTitle *qt6.QLabel
	pill         *qt6.QLabel
	timer        *qt6.QTimer

	providers  []string
	provider   string
	model      string
	workDir    string
	onboarded  bool
	sessions   []guiservice.SessionView
	messages   []guiservice.MsgView
	activeName string
	running    bool
	streaming  bool
	streamText string
	theme      string
	reloading  bool
	pal        *qt6.QPalette

	mu      sync.Mutex
	pending []Event
}

func qwidgetOf(v any) *qt6.QWidget {
	switch t := v.(type) {
	case *qt6.QWidget:
		return t
	case *qt6.QListWidget:
		return t.QListView.QAbstractItemView.QAbstractScrollArea.QFrame.QWidget
	case *qt6.QTextEdit:
		return t.QAbstractScrollArea.QFrame.QWidget
	case *qt6.QPlainTextEdit:
		return t.QAbstractScrollArea.QFrame.QWidget
	case *qt6.QComboBox:
		return t.QWidget
	case *qt6.QCheckBox:
		return t.QAbstractButton.QWidget
	case *qt6.QLabel:
		return t.QFrame.QWidget
	case *qt6.QLineEdit:
		return t.QWidget
	case *qt6.QDialog:
		return t.QWidget
	case *qt6.QSplitter:
		return t.QFrame.QWidget
	case *qt6.QPushButton:
		return t.QAbstractButton.QWidget
	case *qt6.QStatusBar:
		return t.QWidget
	}
	return nil
}

func RunQt() int {
	qt6.NewQApplication(os.Args)
	q := &qtApp{theme: "system"}
	if detectDark() {
		q.theme = "dark"
	} else {
		q.theme = "light"
	}
	q.build()
	qt6.QApplication_Exec()
	return 0
}

func detectDark() bool {
	defer func() { _ = recover() }()
	pal := qt6.QApplication_Palette(nil)
	if pal == nil {
		return false
	}
	c := pal.Color(qt6.QPalette__Active, qt6.QPalette__Window)
	if c == nil {
		return false
	}
	return c.Lightness() < 128
}

func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

func (q *qtApp) build() {
	q.eng = New()
	q.providers = q.eng.Providers()
	q.theme = q.eng.GetTheme()

	q.win = qt6.NewQMainWindow2()
	q.win.SetWindowTitle("GoDucky")
	q.win.Resize(1150, 760)

	splitter := qt6.NewQSplitter3(qt6.Horizontal)

	sidebar := qt6.NewQWidget2()
	vb := qt6.NewQBoxLayout2(qt6.QBoxLayout__TopToBottom, sidebar)
	vb.SetContentsMargins(6, 8, 6, 6)

	title := qt6.NewQLabel3("GoDucky")
	title.SetStyleSheet("font-size:17px; font-weight:bold;")
	sub := qt6.NewQLabel3("AI coding agent")
	sub.SetStyleSheet("color:#888;")
	vb.AddWidget2(qwidgetOf(title), 0)
	vb.AddWidget2(qwidgetOf(sub), 0)

	q.sessionsList = qt6.NewQListWidget2()
	q.sessionsList.SetMinimumWidth(220)
	q.sessionsList.OnCurrentItemChanged(func(current, previous *qt6.QListWidgetItem) {
		if current == nil || q.reloading {
			return
		}
		name := current.Text()
		if name == q.activeName {
			return
		}
		if err := q.eng.Resume(name); err == nil {
			q.reload()
		} else {
			q.status("Error: " + err.Error())
		}
	})
	vb.AddWidget2(qwidgetOf(q.sessionsList), 1)

	newBtn := qt6.NewQPushButton3("New")
	newBtn.SetToolTip("Start a new chat")
	newBtn.OnClicked(func() {
		if err := q.eng.NewChat(); err == nil {
			q.reload()
		}
	})
	hb := qt6.NewQBoxLayout(qt6.QBoxLayout__LeftToRight)
	settingsBtn := qt6.NewQPushButton3("Settings")
	settingsBtn.OnClicked(func() { q.showSettings() })
	themeBtn := qt6.NewQPushButton3("Theme")
	themeBtn.OnClicked(func() { q.cycleTheme() })
	hb.AddWidget2(qwidgetOf(newBtn), 1)
	hb.AddWidget2(qwidgetOf(settingsBtn), 1)
	hb.AddWidget2(qwidgetOf(themeBtn), 1)
	vb.AddLayout2(hb.QLayout, 0)
	sidebar.SetLayout(vb.QLayout)

	splitter.AddWidget(qwidgetOf(sidebar))

	main := qt6.NewQWidget2()
	ml := qt6.NewQBoxLayout2(qt6.QBoxLayout__TopToBottom, main)
	ml.SetContentsMargins(10, 10, 10, 6)

	top := qt6.NewQWidget2()
	topL := qt6.NewQBoxLayout2(qt6.QBoxLayout__LeftToRight, top)
	q.sessionTitle = qt6.NewQLabel3("")
	topL.AddWidget2(qwidgetOf(q.sessionTitle), 1)
	q.pill = qt6.NewQLabel3("")
	q.pill.SetStyleSheet("color:#888;")
	topL.AddWidget2(qwidgetOf(q.pill), 0)

	shareBtn := qt6.NewQPushButton3("Share")
	shareBtn.SetToolTip("Copy the conversation to the clipboard")
	shareBtn.OnClicked(func() { q.shareActive() })
	saveBtn := qt6.NewQPushButton3("Save")
	saveBtn.OnClicked(func() { q.saveActive() })
	renameBtn := qt6.NewQPushButton3("Rename")
	renameBtn.OnClicked(func() { q.promptRename() })
	deleteBtn := qt6.NewQPushButton3("Delete")
	deleteBtn.OnClicked(func() { q.confirmDelete() })
	for _, b := range []*qt6.QPushButton{shareBtn, saveBtn, renameBtn, deleteBtn} {
		topL.AddWidget2(qwidgetOf(b), 0)
	}
	top.SetLayout(topL.QLayout)

	q.chat = qt6.NewQTextEdit2()
	q.chat.SetReadOnly(true)
	q.chat.SetHtml("")

	q.thinkingLbl = qt6.NewQLabel3("Thinking")
	q.thinkingLbl.Hide()

	composerRow := qt6.NewQWidget2()
	cL := qt6.NewQBoxLayout2(qt6.QBoxLayout__LeftToRight, composerRow)
	cL.SetContentsMargins(0, 6, 0, 0)
	q.composer = qt6.NewQPlainTextEdit2()
	q.composer.SetPlaceholderText("Message GoDucky…")
	q.composer.SetMaximumHeight(120)
	q.sendBtn = qt6.NewQPushButton3("Send")
	q.sendBtn.SetToolTip("Send the message")
	q.sendBtn.SetEnabled(false)
	q.sendBtn.OnClicked(func() { q.send() })
	q.stopBtn = qt6.NewQPushButton3("Stop")
	q.stopBtn.SetToolTip("Stop generating")
	q.stopBtn.SetEnabled(false)
	q.stopBtn.OnClicked(func() { q.eng.Stop() })
	cL.AddWidget2(qwidgetOf(q.composer), 1)
	cL.AddWidget2(qwidgetOf(q.sendBtn), 0)
	cL.AddWidget2(qwidgetOf(q.stopBtn), 0)
	composerRow.SetLayout(cL.QLayout)

	hint := qt6.NewQLabel3("GoDucky can make mistakes. Check important info.")
	hint.SetStyleSheet("color:#888; font-size:11px;")

	ml.AddWidget2(qwidgetOf(top), 0)
	ml.AddWidget2(qwidgetOf(q.chat), 1)
	ml.AddWidget2(qwidgetOf(q.thinkingLbl), 0)
	ml.AddWidget2(qwidgetOf(composerRow), 0)
	ml.AddWidget2(qwidgetOf(hint), 0)
	main.SetLayout(ml.QLayout)

	splitter.AddWidget(qwidgetOf(main))
	splitter.SetStretchFactor(0, 0)
	splitter.SetStretchFactor(1, 1)

	q.win.SetCentralWidget(splitter.QFrame.QWidget)
	q.win.Show()

	q.composer.OnTextChanged(func() {
		q.sendBtn.SetEnabled(strings.TrimSpace(q.composer.ToPlainText()) != "")
	})
	q.applyTheme()

	q.timer = qt6.NewQTimer()
	q.timer.OnTimeout(func() { q.drain() })
	q.timer.Start(40)

	evs := q.eng.Events()
	go func() {
		for ev := range evs {
			q.mu.Lock()
			q.pending = append(q.pending, ev)
			q.mu.Unlock()
		}
	}()

	q.reload()
	if !q.onboarded {
		q.status("Set up a provider in Settings to start chatting.")
	}
}

func (q *qtApp) send() {
	text := q.composer.ToPlainText()
	if strings.TrimSpace(text) == "" {
		return
	}
	q.composer.SetPlainText("")
	q.messages = append(q.messages, guiservice.MsgView{Role: "user", Text: text})
	q.streaming = false
	q.streamText = ""
	q.renderChat()
	q.setRunning(true)
	if err := q.eng.Send(text); err != nil {
		q.setRunning(false)
		q.status("Error: " + err.Error())
	}
}

func (q *qtApp) status(msg string) {
	q.win.StatusBar().ShowMessage(msg)
}

func (q *qtApp) setRunning(run bool) {
	q.running = run
	q.stopBtn.SetEnabled(run)
	q.sendBtn.SetEnabled(!run && strings.TrimSpace(q.composer.ToPlainText()) != "")
	if run {
		q.thinkingLbl.Show()
	} else {
		q.thinkingLbl.Hide()
	}
}

func (q *qtApp) renderChat() {
	var b strings.Builder
	for _, m := range q.messages {
		if b.Len() > 0 {
			b.WriteString("<hr/>")
		}
		switch m.Role {
		case "user":
			fmt.Fprintf(&b, `<p><b>You</b><br/><span style="background:#ededed;padding:6px 10px;border-radius:8px;display:inline-block;">%s</span></p>`, escapeHTML(m.Text))
		case "system":
			fmt.Fprintf(&b, `<p style="color:#98989f;">%s</p>`, escapeHTML(m.Text))
		default:
			fmt.Fprintf(&b, `<p>%s</p>`, escapeHTML(m.Text))
		}
	}
	q.chat.SetHtml(b.String())
	q.chat.MoveCursor(qt6.QTextCursor__End)
}

func (q *qtApp) reload() {
	q.reloading = true
	defer func() { q.reloading = false }()

	info := q.eng.GetInfo()
	if info == nil {
		return
	}
	q.provider = info.Provider
	q.model = info.Model
	q.workDir = info.WorkDir
	q.onboarded = info.Onboarded
	q.activeName = info.Name
	q.messages = info.Messages
	q.sessions = info.Sessions

	q.sessionTitle.SetText(q.activeName)
	modelLabel := info.Provider
	if info.Model != "" {
		modelLabel = info.Provider + " · " + info.Model
	}
	q.pill.SetText(modelLabel)

	q.sessionsList.Clear()
	for _, s := range info.Sessions {
		q.sessionsList.AddItem(s.Name)
	}

	q.renderChat()
	q.setRunning(q.eng.IsRunning())
}

func (q *qtApp) drain() {
	q.mu.Lock()
	evs := q.pending
	q.pending = nil
	q.mu.Unlock()

	for _, ev := range evs {
		switch ev.Name {
		case EvtStream:
			if data, ok := ev.Data.(map[string]any); ok {
				if text, ok := data["text"].(string); ok {
					q.streaming = true
					q.streamText += text
					q.ensureStreamingMsg()
					q.renderChat()
				}
			}
		case EvtStatus:
			if data, ok := ev.Data.(map[string]any); ok {
				if msg, ok := data["msg"].(string); ok {
					q.status(msg)
				}
			}
		case EvtComplete:
			q.streaming = false
			q.streamText = ""
			q.setRunning(false)
			q.reload()
		case EvtError:
			q.streaming = false
			q.streamText = ""
			if data, ok := ev.Data.(map[string]any); ok {
				if msg, ok := data["error"].(string); ok {
					q.status("Error: " + msg)
				}
			}
			q.setRunning(false)
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
				q.showApproval(req)
			}
		}
	}
}

func (q *qtApp) ensureStreamingMsg() {
	if len(q.messages) == 0 || q.messages[len(q.messages)-1].Role != "assistant" {
		q.messages = append(q.messages, guiservice.MsgView{Role: "assistant", Text: q.streamText})
		return
	}
	q.messages[len(q.messages)-1].Text = q.streamText
}

func (q *qtApp) shareActive() {
	if err := q.eng.ShareSession(q.activeName); err != nil {
		q.status("Share failed: " + err.Error())
		return
	}
	q.status("Conversation copied to clipboard")
}

func (q *qtApp) saveActive() {
	if err := q.eng.SaveSession(q.activeName); err != nil {
		q.status("Save failed: " + err.Error())
		return
	}
	q.status("Saved")
}

func (q *qtApp) promptRename() {
	if q.activeName == "" {
		return
	}
	name := qt6.QInputDialog_GetText2(q.win.QWidget, "Rename chat", "New name:", qt6.QLineEdit__Normal)
	if strings.TrimSpace(name) != "" {
		_ = q.eng.RenameSession(q.activeName, strings.TrimSpace(name))
		q.reload()
	}
}

func (q *qtApp) confirmDelete() {
	if q.activeName == "" {
		return
	}
	name := q.activeName
	btn := qt6.QMessageBox_Question2(q.win.QWidget, "Delete chat",
		"Delete \""+name+"\"? This cannot be undone.",
		qt6.QMessageBox__Yes, qt6.QMessageBox__No)
	if btn == int(qt6.QMessageBox__Yes) {
		q.messages = nil
		q.activeName = ""
		_ = q.eng.DeleteSession(name)
		q.reload()
	}
}

func (q *qtApp) showApproval(req guiservice.ApprovalRequest) {
	btn := qt6.QMessageBox_Question2(q.win.QWidget, "Approve action", req.Desc,
		qt6.QMessageBox__Yes, qt6.QMessageBox__No)
	q.eng.Approve(req.ID, btn == int(qt6.QMessageBox__Yes))
}

func (q *qtApp) cycleTheme() {
	opts := []string{"light", "dark", "system"}
	idx := 0
	for i, t := range opts {
		if t == q.theme {
			idx = i
			break
		}
	}
	q.theme = opts[(idx+1)%len(opts)]
	_ = q.eng.SetConfigValue("theme", q.theme)
	q.applyTheme()
	q.status("Theme: " + q.theme)
}

func (q *qtApp) applyTheme() {
	effective := q.theme
	if effective == "system" {
		if detectDark() {
			effective = "dark"
		} else {
			effective = "light"
		}
	}
	if q.pal == nil {
		q.pal = qt6.NewQPalette()
	}
	if effective == "dark" {
		q.applyPalette(0x1e1f22, 0x26272b, 0x2e3035, 0xe6e7e8, 0x2b2c30, 0x3d7eff)
	} else {
		q.applyPalette(0xffffff, 0xffffff, 0xf7f8f9, 0x1a1a1a, 0xf2f3f4, 0x3d7eff)
	}
	qt6.QApplication_SetPalette(q.pal)
	if effective == "dark" {
		q.sendBtn.SetStyleSheet(`QPushButton{background:#ffffff;color:#1a1a1a;border-radius:18px;padding:8px 16px;font-weight:bold;}QPushButton:disabled{background:#444;color:#999;}`)
		q.stopBtn.SetStyleSheet(`QPushButton{background:#ffffff;color:#1a1a1a;border-radius:18px;padding:8px 16px;font-weight:bold;}QPushButton:disabled{background:#444;color:#999;}`)
	} else {
		q.sendBtn.SetStyleSheet(`QPushButton{background:#141414;color:#ffffff;border-radius:18px;padding:8px 16px;font-weight:bold;}QPushButton:disabled{background:#ccc;color:#888;}`)
		q.stopBtn.SetStyleSheet(`QPushButton{background:#141414;color:#ffffff;border-radius:18px;padding:8px 16px;font-weight:bold;}QPushButton:disabled{background:#ccc;color:#888;}`)
	}
}

func (q *qtApp) applyPalette(window, base, alt, text, button, highlight uint32) {
	set := func(cr qt6.QPalette__ColorRole, rgb uint32) {
		q.pal.SetColor(qt6.QPalette__All, cr, qt6.NewQColor3(int((rgb>>16)&0xff), int((rgb>>8)&0xff), int(rgb&0xff)))
	}
	set(qt6.QPalette__Window, window)
	set(qt6.QPalette__WindowText, text)
	set(qt6.QPalette__Base, base)
	set(qt6.QPalette__AlternateBase, alt)
	set(qt6.QPalette__Text, text)
	set(qt6.QPalette__Button, button)
	set(qt6.QPalette__ButtonText, text)
	set(qt6.QPalette__Highlight, highlight)
	set(qt6.QPalette__HighlightedText, 0xffffff)
	set(qt6.QPalette__ToolTipBase, base)
	set(qt6.QPalette__ToolTipText, text)
}

func (q *qtApp) showSettings() {
	dia := qt6.NewQDialog2()
	dia.SetModal(true)
	dia.SetWindowTitle("Settings")

	lay := qt6.NewQBoxLayout2(qt6.QBoxLayout__TopToBottom, dia.QWidget)

	lay.AddWidget2(qwidgetOf(qt6.NewQLabel3("Working directory")), 0)
	wd := qt6.NewQLineEdit3(q.workDir)
	lay.AddWidget2(qwidgetOf(wd), 0)

	ap := qt6.NewQCheckBox3("Auto-approve file & command actions")
	ap.SetChecked(q.eng.AutoApprove())
	lay.AddWidget2(qwidgetOf(ap), 0)

	providers := q.providers
	lay.AddWidget2(qwidgetOf(qt6.NewQLabel3("Provider")), 0)
	prov := qt6.NewQComboBox2()
	for _, p := range providers {
		prov.AddItem(p)
	}
	if i := indexOf(q.provider, providers, 0); i >= 0 {
		prov.SetCurrentIndex(i)
	}
	lay.AddWidget2(qwidgetOf(prov), 0)

	models := q.eng.GetModels(q.provider)
	lay.AddWidget2(qwidgetOf(qt6.NewQLabel3("Model")), 0)
	model := qt6.NewQComboBox2()
	model.AddItem("")
	for _, m := range models {
		model.AddItem(m)
	}
	if i := indexOf(q.model, models, 0); i >= 0 {
		model.SetCurrentIndex(i + 1)
	}
	lay.AddWidget2(qwidgetOf(model), 0)

	lay.AddWidget2(qwidgetOf(qt6.NewQLabel3("API key (Ollama doesn't need one)")), 0)
	key := qt6.NewQLineEdit3("")
	key.SetEchoMode(qt6.QLineEdit__Password)
	lay.AddWidget2(qwidgetOf(key), 0)

	rowLay := qt6.NewQBoxLayout2(qt6.QBoxLayout__LeftToRight, nil)
	closeBtn := qt6.NewQPushButton3("Close")
	applyBtn := qt6.NewQPushButton3("Apply")
	applyBtn.OnClicked(func() {
		if selP := prov.CurrentIndex(); selP >= 0 && selP < len(providers) {
			q.eng.SwitchProvider(providers[selP])
		}
		if p := wd.Text(); p != "" {
			q.eng.SetWorkDir(p)
		}
		q.eng.SetAutoApprove(ap.IsChecked())
		if k := key.Text(); k != "" {
			q.eng.SaveAPIKey(q.provider, k)
		}
		if selM := model.CurrentIndex(); selM > 0 && selM-1 < len(models) {
			q.eng.SetModel(models[selM-1])
		}
		q.reload()
		dia.Accept()
	})
	closeBtn.OnClicked(func() { dia.Reject() })
	rowLay.AddWidget2(qwidgetOf(closeBtn), 1)
	rowLay.AddWidget2(qwidgetOf(applyBtn), 0)
	lay.AddLayout2(rowLay.QLayout, 0)

	dia.Resize(420, 380)
	dia.Exec()
}
