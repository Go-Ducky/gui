//go:build windows && win32

package native

/*
#cgo windows LDFLAGS: -luser32 -lgdi32 -ldwmapi -ladvapi32

#include <windows.h>
#include <stdlib.h>
#include <winreg.h>
#include <dwmapi.h>

extern LRESULT goWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);

static LRESULT CALLBACK WndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
	return goWndProc(hwnd, msg, wParam, lParam);
}

static ATOM registerClass(void) {
	WNDCLASSEXW wc;
	wc.cbSize = sizeof(wc);
	wc.style = 0;
	wc.lpfnWndProc = WndProc;
	wc.cbClsExtra = 0;
	wc.cbWndExtra = 0;
	wc.hInstance = GetModuleHandleW(NULL);
	wc.hIcon = LoadIconW(NULL, IDI_APPLICATION);
	wc.hCursor = LoadCursorW(NULL, IDC_ARROW);
	wc.hbrBackground = (HBRUSH)GetStockObject(NULL_BRUSH);
	wc.lpszMenuName = NULL;
	wc.lpszClassName = L"GoduckyWin32";
	wc.hIconSm = NULL;
	return RegisterClassExW(&wc);
}

static HWND mainWin(void) {
	return CreateWindowExW(0, L"GoduckyWin32", L"GoDucky",
		WS_OVERLAPPEDWINDOW, CW_USEDEFAULT, CW_USEDEFAULT, 1150, 780,
		NULL, NULL, GetModuleHandleW(NULL), NULL);
}

static HWND mkedit(HWND parent, DWORD extra) {
	return CreateWindowExW(WS_EX_CLIENTEDGE, L"EDIT", L"",
		WS_CHILD | WS_VISIBLE | WS_TABSTOP | extra,
		CW_USEDEFAULT, CW_USEDEFAULT, 0, 0, parent, (HMENU)(INT_PTR)0,
		GetModuleHandleW(NULL), NULL);
}

static HWND mklist(HWND parent) {
	return CreateWindowExW(WS_EX_CLIENTEDGE, L"LISTBOX", L"",
		WS_CHILD | WS_VISIBLE | WS_TABSTOP | WS_VSCROLL,
		CW_USEDEFAULT, CW_USEDEFAULT, 0, 0, parent, (HMENU)(INT_PTR)0,
		GetModuleHandleW(NULL), NULL);
}

static HWND mkbtn(HWND parent, const wchar_t *text, int id) {
	return CreateWindowExW(0, L"BUTTON", text, WS_CHILD | WS_VISIBLE | WS_TABSTOP,
		CW_USEDEFAULT, CW_USEDEFAULT, 0, 0, parent, (HMENU)(INT_PTR)id,
		GetModuleHandleW(NULL), NULL);
}

static HWND mkstat(HWND parent, const wchar_t *text) {
	return CreateWindowExW(0, L"STATIC", text, WS_CHILD | WS_VISIBLE,
		CW_USEDEFAULT, CW_USEDEFAULT, 0, 0, parent, NULL,
		GetModuleHandleW(NULL), NULL);
}

static wchar_t *utf16(const char *s) {
	int n = MultiByteToWideChar(CP_UTF8, 0, s, -1, NULL, 0);
	if (n <= 0) return NULL;
	wchar_t *w = (wchar_t *)malloc((size_t)n * sizeof(wchar_t));
	MultiByteToWideChar(CP_UTF8, 0, s, -1, w, n);
	return w;
}

static char *utf8(const wchar_t *w) {
	int n = WideCharToMultiByte(CP_UTF8, 0, w, -1, NULL, 0, NULL, NULL);
	if (n <= 0) return NULL;
	char *s = (char *)malloc((size_t)n);
	WideCharToMultiByte(CP_UTF8, 0, w, -1, s, n, NULL, NULL);
	return s;
}

static HFONT makeFont(void) {
	return CreateFontW(-15, 0, 0, 0, FW_NORMAL, 0, 0, 0,
		DEFAULT_CHARSET, OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS,
		CLEARTYPE_QUALITY, DEFAULT_PITCH | FF_DONTCARE, L"Segoe UI");
}

static void setFont(HWND h, HFONT f) {
	SendMessageW(h, WM_SETFONT, (WPARAM)f, 1);
}

static void setEditMax(HWND h) {
	SendMessageW(h, EM_LIMITTEXT, 0x7fffffff, 0);
}

static int loWord(WPARAM w) {
	return (int)(w & 0xffff);
}

static int hiWord(WPARAM w) {
	return (int)((w >> 16) & 0xffff);
}

static void move(HWND h, int x, int y, int w, int ht) {
	MoveWindow(h, x, y, w, ht, TRUE);
}

static void paintBg(HWND h, HDC dc, int dark) {
	COLORREF c = dark ? RGB(0x1e, 0x1f, 0x22) : RGB(0xf4, 0xf3, 0xf2);
	HBRUSH br = CreateSolidBrush(c);
	RECT r;
	GetClientRect(h, &r);
	FillRect(dc, &r, br);
	DeleteObject(br);
}

static void setCtlText(HDC dc, int dark) {
	SetBkMode(dc, TRANSPARENT);
	SetTextColor(dc, dark ? RGB(0xe8, 0xe7, 0xe6) : RGB(0x1a, 0x1a, 0x1a));
}

static HBRUSH makeBrush(int dark, int base) {
	COLORREF c;
	if (dark) {
		c = base ? RGB(0x29, 0x2a, 0x26) : RGB(0x22, 0x23, 0x1e);
	} else {
		c = base ? RGB(0xff, 0xff, 0xff) : RGB(0xf4, 0xf3, 0xf2);
	}
	return CreateSolidBrush(c);
}

static void delBrush(HBRUSH b) {
	DeleteObject(b);
}

static intptr_t brushVal(HBRUSH b) {
	return (intptr_t)b;
}

static int listSel(HWND lb) {
	return (int)SendMessageW(lb, LB_GETCURSEL, 0, 0);
}

static void listReset(HWND lb) {
	SendMessageW(lb, LB_RESETCONTENT, 0, 0);
}

static void listAdd(HWND lb, const wchar_t *w) {
	SendMessageW(lb, LB_ADDSTRING, 0, (LPARAM)w);
}

static void setTextW(HWND h, const wchar_t *w) {
	SetWindowTextW(h, w);
}

static wchar_t *editContents(HWND e) {
	int n = GetWindowTextLengthW(e) + 1;
	wchar_t *w = (wchar_t *)malloc((size_t)n * sizeof(wchar_t));
	GetWindowTextW(e, w, n);
	return w;
}

static void setDarkMode(HWND h, int dark) {
	int v = dark;
	DwmSetWindowAttribute(h, DWMWA_USE_IMMERSIVE_DARK_MODE, &v, sizeof(v));
}

static int askYesNo(HWND parent, const wchar_t *title, const wchar_t *msg) {
	return MessageBoxW(parent, msg, title, MB_YESNO);
}
*/
import "C"

import (
	"strings"
	"sync"
	"unsafe"

	"github.com/go-ducky/gui/internal/guiservice"
)

const (
	idSessions = 1
	idChat     = 2
	idComposer = 3
	idNew      = 101
	idTheme    = 102
	idSend     = 103
	idStop     = 104
	idPill     = 201
	idStatus   = 202
)

type winApp struct {
	eng *Engine

	hwnd     C.HWND
	sessions C.HWND
	chat     C.HWND
	composer C.HWND
	sendBtn  C.HWND
	stopBtn  C.HWND
	status   C.HWND
	pill     C.HWND
	newBtn   C.HWND
	themeBtn C.HWND
	font     C.HFONT

	brushBg   C.HBRUSH
	brushBase C.HBRUSH

	theme      string
	systemDark bool

	providers  []string
	provider   string
	model      string
	workDir    string
	onboarded  bool
	sessionsV  []guiservice.SessionView
	messages   []guiservice.MsgView
	activeName string
	running    bool
	streamText strings.Builder

	mu      sync.Mutex
	pending []Event
}

var current *winApp

func RunWin32() int {
	g := &winApp{theme: "system"}
	current = g
	g.eng = New()
	g.theme = g.eng.GetTheme()
	g.systemDark = detectWinDark()
	if !g.createUI() {
		return 1
	}
	g.applyTheme()
	g.reload()
	g.messageLoop()
	return 0
}

func wstr(s string) *C.wchar_t {
	return C.utf16(C.CString(s))
}

func (g *winApp) setText(hwnd C.HWND, s string) {
	w := wstr(s)
	C.setTextW(hwnd, w)
	C.free(unsafe.Pointer(w))
}

func (g *winApp) createUI() bool {
	if C.registerClass() == 0 {
		return false
	}
	g.hwnd = C.mainWin()
	if g.hwnd == nil {
		return false
	}

	g.sessions = C.mklist(g.hwnd)
	C.SetWindowLongPtrW(g.sessions, C.GWLP_ID, C.LONG_PTR(intptr(idSessions)))
	g.chat = C.mkedit(g.hwnd, C.ES_MULTILINE|C.ES_READONLY|C.ES_AUTOVSCROLL|C.WS_VSCROLL)
	C.SetWindowLongPtrW(g.chat, C.GWLP_ID, C.LONG_PTR(intptr(idChat)))
	C.setEditMax(g.chat)
	g.composer = C.mkedit(g.hwnd, C.ES_MULTILINE|C.ES_AUTOVSCROLL|C.WS_VSCROLL)
	C.SetWindowLongPtrW(g.composer, C.GWLP_ID, C.LONG_PTR(intptr(idComposer)))
	C.setEditMax(g.composer)

	g.sendBtn = C.mkbtn(g.hwnd, wstr("Send"), idSend)
	g.stopBtn = C.mkbtn(g.hwnd, wstr("Stop"), idStop)
	g.newBtn = C.mkbtn(g.hwnd, wstr("New chat"), idNew)
	g.themeBtn = C.mkbtn(g.hwnd, wstr("Theme"), idTheme)
	g.pill = C.mkstat(g.hwnd, wstr(""))
	g.status = C.mkstat(g.hwnd, wstr(""))

	g.font = C.makeFont()
	for _, h := range []C.HWND{g.sessions, g.chat, g.composer, g.sendBtn, g.stopBtn, g.newBtn, g.themeBtn, g.pill, g.status} {
		C.setFont(h, g.font)
	}

	C.SetTimer(g.hwnd, 1, 50, nil)
	C.ShowWindow(g.hwnd, C.SW_SHOW)
	C.UpdateWindow(g.hwnd)
	return true
}

func (g *winApp) messageLoop() {
	evs := g.eng.Events()
	go func() {
		for ev := range evs {
			g.mu.Lock()
			g.pending = append(g.pending, ev)
			g.mu.Unlock()
		}
	}()
	var msg C.MSG
	for {
		r := C.GetMessageW(&msg, nil, 0, 0)
		if r == 0 {
			return
		}
		if r == -1 {
			return
		}
		C.TranslateMessage(&msg)
		C.DispatchMessageW(&msg)
	}
}

//export goWndProc
func goWndProc(hwnd C.HWND, msg C.UINT, wParam C.WPARAM, lParam C.LPARAM) C.LRESULT {
	if current == nil {
		return C.DefWindowProcW(hwnd, msg, wParam, lParam)
	}
	return current.dispatch(hwnd, msg, wParam, lParam)
}

func (g *winApp) dispatch(hwnd C.HWND, msg C.UINT, wParam C.WPARAM, lParam C.LPARAM) C.LRESULT {
	switch msg {
	case C.WM_DESTROY:
		C.PostQuitMessage(0)
		return 0
	case C.WM_SIZE:
		g.layout()
		return 0
	case C.WM_TIMER:
		g.drain()
		return 0
	case C.WM_ERASEBKGND:
		C.paintBg(g.hwnd, C.HDC(unsafe.Pointer(wParam)), C.int(boolToInt(g.dark())))
		return 1
	case C.WM_CTLCOLORSTATIC, C.WM_CTLCOLORBTN:
		C.setCtlText(C.HDC(unsafe.Pointer(wParam)), C.int(boolToInt(g.dark())))
		return C.LRESULT(C.brushVal(g.brushBg))
	case C.WM_CTLCOLOREDIT, C.WM_CTLCOLORLISTBOX:
		C.setCtlText(C.HDC(unsafe.Pointer(wParam)), C.int(boolToInt(g.dark())))
		return C.LRESULT(C.brushVal(g.brushBase))
	case C.WM_COMMAND:
		g.command(C.loWord(wParam), C.hiWord(wParam))
		return 0
	}
	return C.DefWindowProcW(hwnd, msg, wParam, lParam)
}

func (g *winApp) dark() bool {
	if g.theme == "dark" {
		return true
	}
	if g.theme == "light" {
		return false
	}
	return g.systemDark
}

func (g *winApp) move(h C.HWND, x, y, w, ht int) {
	C.move(h, C.int(x), C.int(y), C.int(w), C.int(ht))
}

func (g *winApp) layout() {
	var rc C.RECT
	C.GetClientRect(g.hwnd, &rc)
	w := int(rc.right)
	h := int(rc.bottom)
	m, side, top, bottom := 8, 260, 44, 118
	g.move(g.sessions, m, top, side, h-top-bottom)
	chX := m + side + 8
	chW := w - chX - m
	g.move(g.chat, chX, top, chW, h-top-bottom)
	g.move(g.composer, chX, h-bottom, chW-150, 84)
	g.move(g.sendBtn, w-m-140, h-bottom, 132, 40)
	g.move(g.stopBtn, w-m-140, h-bottom+44, 132, 40)
	g.move(g.pill, m, 10, 220, 22)
	g.move(g.newBtn, 250, 10, 1, 1)
	g.move(g.themeBtn, 250, 10, 1, 1)
	g.move(g.status, chX, h-22, chW, 18)
}

func (g *winApp) command(lo, hi int) {
	if lo == idSessions && hi == C.LBN_SELCHANGE {
		g.onSessionSel()
		return
	}
	switch lo {
	case idNew:
		g.onNew()
	case idTheme:
		g.cycleTheme()
	case idSend:
		g.send()
	case idStop:
		g.eng.Stop()
	}
}

func (g *winApp) onSessionSel() {
	sel := C.listSel(g.sessions)
	if sel < 0 || sel >= len(g.sessionsV) {
		return
	}
	name := g.sessionsV[sel].Name
	if name == "" || name == g.activeName {
		return
	}
	if err := g.eng.Resume(name); err == nil {
		g.reload()
	} else {
		g.setStatus("Error: " + err.Error())
	}
}

func (g *winApp) onNew() {
	if err := g.eng.NewChat(); err == nil {
		g.reload()
	}
}

func (g *winApp) setRunning(run bool) {
	g.running = run
	g.setEnabled(g.sendBtn, !run)
	g.setEnabled(g.stopBtn, run)
}

func (g *winApp) setEnabled(h C.HWND, on bool) {
	C.EnableWindow(h, C.BOOL(boolToInt(on)))
}



func (g *winApp) setStatus(s string) {
	g.setText(g.status, s)
}

func (g *winApp) composerText() string {
	w := C.editContents(g.composer)
	defer C.free(unsafe.Pointer(w))
	cs := C.utf8(w)
	defer C.free(unsafe.Pointer(cs))
	return C.GoString(cs)
}

func (g *winApp) setComposer(s string) {
	g.setText(g.composer, s)
}

func (g *winApp) send() {
	text := strings.TrimSpace(g.composerText())
	if text == "" {
		return
	}
	g.messages = append(g.messages, guiservice.MsgView{Role: "user", Text: text})
	g.streamText.Reset()
	g.renderChat()
	g.setComposer("")
	g.setRunning(true)
	if err := g.eng.Send(text); err != nil {
		g.setRunning(false)
		g.setStatus("Error: " + err.Error())
	}
}

func (g *winApp) renderChat() {
	var b strings.Builder
	for i, m := range g.messages {
		if i > 0 {
			b.WriteString("\r\n\r\n")
		}
		switch m.Role {
		case "user":
			b.WriteString("[You] " + m.Text)
		case "system":
			b.WriteString("- " + m.Text + " -")
		default:
			b.WriteString(m.Text)
		}
	}
	g.setText(g.chat, b.String())
}

func (g *winApp) drain() {
	g.mu.Lock()
	evs := g.pending
	g.pending = nil
	g.mu.Unlock()

	for _, ev := range evs {
		switch ev.Name {
		case EvtStream:
			if data, ok := ev.Data.(map[string]any); ok {
				if text, ok := data["text"].(string); ok {
					g.streamText.WriteString(text)
					g.appendAssistant(g.streamText.String())
				}
			}
		case EvtStatus:
			if data, ok := ev.Data.(map[string]any); ok {
				if msg, ok := data["msg"].(string); ok {
					g.setStatus(msg)
				}
			}
		case EvtComplete:
			g.setRunning(false)
			g.reload()
		case EvtError:
			if data, ok := ev.Data.(map[string]any); ok {
				if msg, ok := data["error"].(string); ok {
					g.setStatus("Error: " + msg)
				}
			}
			g.setRunning(false)
		case EvtApproval:
			g.handleApproval(ev)
		}
	}
}

func (g *winApp) appendAssistant(text string) {
	if len(g.messages) > 0 && g.messages[len(g.messages)-1].Role == "assistant" {
		g.messages[len(g.messages)-1].Text = text
	} else {
		g.messages = append(g.messages, guiservice.MsgView{Role: "assistant", Text: text})
	}
	g.renderChat()
}

func (g *winApp) handleApproval(ev Event) {
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
	if req.ID == "" {
		return
	}
	title := wstr("Approve action")
	msg := wstr(req.Desc)
	res := C.askYesNo(g.hwnd, title, msg)
	C.free(unsafe.Pointer(title))
	C.free(unsafe.Pointer(msg))
	g.eng.Approve(req.ID, res == C.IDYES)
}

func (g *winApp) reload() {
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
	g.sessionsV = info.Sessions

	pill := info.Provider
	if info.Model != "" {
		pill = info.Provider + " · " + info.Model
	}
	g.setText(g.pill, pill)

	C.listReset(g.sessions)
	for _, s := range info.Sessions {
		w := wstr(s.Name)
		C.listAdd(g.sessions, w)
		C.free(unsafe.Pointer(w))
	}

	g.renderChat()
	g.setRunning(g.eng.IsRunning())
	if !g.onboarded {
		g.setStatus("Set up a provider to start chatting.")
	}
}

func (g *winApp) cycleTheme() {
	order := []string{"system", "light", "dark"}
	next := "system"
	for i, t := range order {
		if t == g.theme {
			next = order[(i+1)%len(order)]
			break
		}
	}
	g.theme = next
	_ = g.eng.SetConfigValue("theme", g.theme)
	g.applyTheme()
	g.setStatus("Theme: " + g.theme)
}

func (g *winApp) applyTheme() {
	if g.brushBg != nil {
		C.delBrush(g.brushBg)
	}
	if g.brushBase != nil {
		C.delBrush(g.brushBase)
	}
	g.brushBg = C.makeBrush(C.int(boolToInt(g.dark())), 0)
	g.brushBase = C.makeBrush(C.int(boolToInt(g.dark())), 1)
	C.setDarkMode(g.hwnd, C.int(boolToInt(g.dark())))
	C.InvalidateRect(g.hwnd, nil, C.TRUE)
	C.UpdateWindow(g.hwnd)
}

func detectWinDark() bool {
	var dark C.DWORD
	sz := C.DWORD(unsafe.Sizeof(dark))
	key := wstr(`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`)
	name := wstr("AppsUseLightTheme")
	defer C.free(unsafe.Pointer(key))
	defer C.free(unsafe.Pointer(name))
	if C.RegGetValueW(C.HKEY_CURRENT_USER, key, name, C.RRF_RT_REG_DWORD, nil,
		unsafe.Pointer(&dark), &sz) != 0 {
		return false
	}
	return dark == 0
}