//go:build darwin && cocoa

package native

/*
#cgo LDFLAGS: -framework Cocoa

#include <AppKit/AppKit.h>
#include <objc/runtime.h>
#include <stdlib.h>
#include <string.h>

extern void goButton(int tag);
extern void goPopup(int tag);
extern void goTick(void);

static void buttonClick(id self, SEL _cmd, id sender) {
	goButton((int)[(NSControl *)sender tag]);
}

static void popupChanged(id self, SEL _cmd, id sender) {
	goPopup((int)[(NSControl *)sender tag]);
}

static void timerTick(id self, SEL _cmd, id timer) {
	goTick();
}

static BOOL appShouldTerminate(id self, SEL _cmd, id sender) {
	return YES;
}

static void *makeController(void) {
	Class cls = objc_allocateClassPair([NSObject class], "GoDuckyController", 0);
	class_addMethod(cls, sel_registerName("buttonClick:"), (IMP)buttonClick, "v@:@");
	class_addMethod(cls, sel_registerName("popupChanged:"), (IMP)popupChanged, "v@:@");
	class_addMethod(cls, sel_registerName("timerTick:"), (IMP)timerTick, "v@:@");
	objc_registerClassPair(cls);
	return (void *)[[(id)cls alloc] init];
}

static void *makeAppDelegate(void) {
	Class cls = objc_allocateClassPair([NSObject class], "GoDuckyAppDelegate", 0);
	class_addMethod(cls, sel_registerName("applicationShouldTerminateAfterLastWindowClosed:"),
		(IMP)appShouldTerminate, "c@:@");
	objc_registerClassPair(cls);
	return (void *)[[(id)cls alloc] init];
}

static void *sharedApp(void) {
	return (void *)[NSApplication sharedApplication];
}

static void setDelegate(void *del) {
	[NSApp setDelegate:(id)del];
}

static void activateApp(void) {
	NSApp.activationPolicy = NSApplicationActivationPolicyRegular;
	[NSApp activateIgnoringOtherApps:YES];
}

static void runApp(void) {
	[NSApp run];
}

static void *makeWindow(void) {
	NSWindow *win = [[NSWindow alloc] initWithContentRect:NSMakeRect(0, 0, 1150, 780)
		styleMask:(NSWindowStyleMaskTitled | NSWindowStyleMaskClosable |
			NSWindowStyleMaskMiniaturizable | NSWindowStyleMaskResizable)
		backing:NSBackingStoreBuffered defer:NO];
	[win setTitle:@"GoDucky"];
	[win center];
	return (void *)win;
}

static void *contentView(void *win) {
	return (void *)[(NSWindow *)win contentView];
}

static void showWindow(void *win) {
	[(NSWindow *)win makeKeyAndOrderFront:nil];
	[NSApp activateIgnoringOtherApps:YES];
}

static void *makeTextView(void *cv, double x, double y, double w, double h, int flags, int readOnly) {
	NSScrollView *sv = [[NSScrollView alloc] initWithFrame:NSMakeRect(x, y, w, h)];
	NSTextView *tv = [[NSTextView alloc] initWithFrame:NSMakeRect(0, 0, w, h)];
	[tv setEditable:readOnly ? NO : YES];
	[tv setSelectable:YES];
	[tv setRichText:NO];
	[tv setFont:[NSFont monospacedSystemFontOfSize:13 weight:NSFontWeightRegular]];
	[tv setAutoresizingMask:NSViewWidthSizable];
	[sv setDocumentView:tv];
	[sv setHasVerticalScroller:YES];
	[sv setAutohidesScrollers:YES];
	[sv setBorderType:NSBezelBorder];
	[sv setAutoresizingMask:flags];
	[(NSView *)cv addSubview:sv];
	return (void *)tv;
}

static void *makeButton(void *cv, const char *title, double x, double y, double w, double h, int tag, int flags, void *ctrl) {
	NSButton *b = [[NSButton alloc] initWithFrame:NSMakeRect(x, y, w, h)];
	[b setBezelStyle:NSBezelStyleRounded];
	[[b cell] setFont:[NSFont systemFontOfSize:13]];
	[b setTitle:[NSString stringWithUTF8String:title]];
	[b setTag:tag];
	[b setTarget:(id)ctrl];
	[b setAction:@selector(buttonClick:)];
	[b setAutoresizingMask:flags];
	[(NSView *)cv addSubview:b];
	return (void *)b;
}

static void *makePopup(void *cv, double x, double y, double w, double h, int tag, int flags, void *ctrl) {
	NSPopUpButton *p = [[NSPopUpButton alloc] initWithFrame:NSMakeRect(x, y, w, h) pullsDown:NO];
	[p setTag:tag];
	[p setTarget:(id)ctrl];
	[p setAction:@selector(popupChanged:)];
	[p setAutoresizingMask:flags];
	[(NSView *)cv addSubview:p];
	return (void *)p;
}

static void *makeLabel(void *cv, const char *text, double x, double y, double w, double h, int flags) {
	NSTextField *l = [[NSTextField alloc] initWithFrame:NSMakeRect(x, y, w, h)];
	[l setBezeled:NO];
	[l setDrawsBackground:NO];
	[l setEditable:NO];
	[l setSelectable:NO];
	[l setFont:[NSFont systemFontOfSize:13]];
	[l setStringValue:[NSString stringWithUTF8String:text]];
	[l setAutoresizingMask:flags];
	[(NSView *)cv addSubview:l];
	return (void *)l;
}

static void labelSet(void *l, const char *s) {
	[(NSTextField *)l setStringValue:[NSString stringWithUTF8String:s]];
}

static void setEnabled(void *c, int on) {
	[(NSControl *)c setEnabled:on ? YES : NO];
}

static void setViewText(void *tv, const char *s) {
	NSTextView *v = (NSTextView *)tv;
	[v setString:[NSString stringWithUTF8String:s]];
	[v scrollRangeToVisible:NSMakeRange([[v string] length], 0)];
}

static char *viewText(void *tv) {
	const char *s = [[(NSTextView *)tv string] UTF8String];
	return strdup(s ? s : "");
}

static void styleTextView(void *tv, int dark) {
	NSTextView *v = (NSTextView *)tv;
	NSColor *bg;
	NSColor *fg;
	if (dark) {
		bg = [NSColor colorWithSRGBRed:0x22 / 255.0 green:0x23 / 255.0 blue:0x1e / 255.0 alpha:1];
		fg = [NSColor colorWithSRGBRed:0xe8 / 255.0 green:0xe7 / 255.0 blue:0xe6 / 255.0 alpha:1];
	} else {
		bg = [NSColor whiteColor];
		fg = [NSColor blackColor];
	}
	[v setBackgroundColor:bg];
	[v setTextColor:fg];
	[v setInsertionPointColor:fg];
	NSScrollView *sv = [v enclosingScrollView];
	if (sv) {
		[sv setDrawsBackground:YES];
		[sv setBackgroundColor:bg];
	}
}

static void popupClear(void *p) {
	[(NSPopUpButton *)p removeAllItems];
}

static void popupAdd(void *p, const char *s) {
	[(NSPopUpButton *)p addItemWithTitle:[NSString stringWithUTF8String:s]];
}

static int popupSel(void *p) {
	return (int)[(NSPopUpButton *)p indexOfSelectedItem];
}

static void popupSetSel(void *p, int i) {
	[(NSPopUpButton *)p selectItemAtIndex:i];
}

static void setWindowAppearance(void *win, int mode) {
	NSWindow *w = (NSWindow *)win;
	NSAppearance *ap = nil;
	if (mode == 1) {
		ap = [NSAppearance appearanceNamed:NSAppearanceNameDarkAqua];
	} else if (mode == 2) {
		ap = [NSAppearance appearanceNamed:NSAppearanceNameAqua];
	}
	[w setAppearance:ap];
}

static int systemDark(void) {
	NSAppearanceName best = [[NSApp effectiveAppearance] bestMatchFromAppearancesWithNames:@[
		NSAppearanceNameDarkAqua, NSAppearanceNameAqua]];
	return [best isEqualToString:NSAppearanceNameDarkAqua] ? 1 : 0;
}

static void startTimer(void *ctrl) {
	[NSTimer scheduledTimerWithTimeInterval:0.05 target:(id)ctrl selector:@selector(timerTick:)
		userInfo:nil repeats:YES];
}

static int alertAsk(const char *msg) {
	NSAlert *a = [[NSAlert alloc] init];
	[a setMessageText:@"Approve action?"];
	[a setInformativeText:[NSString stringWithUTF8String:msg]];
	[a addButtonWithTitle:@"Allow"];
	[a addButtonWithTitle:@"Deny"];
	int r = (int)[a runModal];
	[a release];
	return r == NSAlertFirstButtonReturn ? 1 : 0;
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
	idNew   = 1
	idTheme = 2
	idSend  = 3
	idStop  = 4
	idPopup = 5
)

type cocoaApp struct {
	eng    *Engine
	win    unsafe.Pointer
	chat   unsafe.Pointer
	compos unsafe.Pointer
	send   unsafe.Pointer
	stop   unsafe.Pointer
	popup  unsafe.Pointer
	status unsafe.Pointer
	pill   unsafe.Pointer
	ctrl   unsafe.Pointer

	theme      string
	systemDark bool

	provider   string
	model      string
	onboarded  bool
	sessions   []guiservice.SessionView
	messages   []guiservice.MsgView
	activeName string
	running    bool
	streamText strings.Builder

	mu      sync.Mutex
	pending []Event
}

var cocoaCur *cocoaApp

func RunCocoa() int {
	g := &cocoaApp{theme: "system"}
	cocoaCur = g
	g.eng = New()
	g.theme = g.eng.GetTheme()
	C.sharedApp()
	g.ctrl = C.makeController()
	C.setDelegate(C.makeAppDelegate())
	C.activateApp()
	g.systemDark = C.systemDark() == 1
	g.build()
	g.initEvents()
	C.startTimer(g.ctrl)
	C.runApp()
	return 0
}

func (g *cocoaApp) build() {
	g.win = C.makeWindow()
	content := C.contentView(g.win)
	wdt := 1150.0
	hgt := 780.0

	C.makeLabel(content, "GoDucky", 12, hgt-38, 180, 24, 8)
	g.pill = C.makeLabel(content, "", wdt-210, hgt-38, 200, 24, 9)

	g.popup = C.makePopup(content, 12, hgt-76, 251, 30, idPopup, 8, g.ctrl)
	C.makeButton(content, "New chat", 276, hgt-76, 96, 30, idNew, 8, g.ctrl)
	C.makeButton(content, "Theme", 380, hgt-76, 80, 30, idTheme, 8, g.ctrl)

	g.chat = C.makeTextView(content, 276, 132, wdt-276, hgt-132-150, 18, 1)
	g.compos = C.makeTextView(content, 276, 30, wdt-276-152-16, 86, 34, 0)

	g.send = C.makeButton(content, "Send", wdt-152, 30, 134, 50, idSend, 32, g.ctrl)
	g.stop = C.makeButton(content, "Stop", wdt-152, 88, 134, 50, idStop, 32, g.ctrl)
	g.status = C.makeLabel(content, "", 276, 6, wdt-276, 16, 34)

	C.setEnabled(g.stop, 0)
	C.showWindow(g.win)
	g.applyTheme()
	g.reload()
}

func (g *cocoaApp) initEvents() {
	evs := g.eng.Events()
	go func() {
		for ev := range evs {
			g.mu.Lock()
			g.pending = append(g.pending, ev)
			g.mu.Unlock()
		}
	}()
}

//export goButton
func goButton(tag C.int) {
	if cocoaCur != nil {
		cocoaCur.button(int(tag))
	}
}

//export goPopup
func goPopup(tag C.int) {
	if cocoaCur != nil {
		cocoaCur.popupSel(int(tag))
	}
}

//export goTick
func goTick() {
	if cocoaCur != nil {
		cocoaCur.drain()
	}
}

func (g *cocoaApp) button(id int) {
	switch id {
	case idNew:
		if err := g.eng.NewChat(); err == nil {
			g.reload()
		}
	case idTheme:
		g.cycleTheme()
	case idSend:
		g.send()
	case idStop:
		g.eng.Stop()
	}
}

func (g *cocoaApp) popupSel(tag int) {
	if tag != idPopup {
		return
	}
	sel := int(C.popupSel(g.popup))
	if sel <= 0 || sel > len(g.sessions) {
		return
	}
	name := g.sessions[sel-1].Name
	if name == "" || name == g.activeName {
		return
	}
	if err := g.eng.Resume(name); err == nil {
		g.reload()
	} else {
		g.setStatus("Error: " + err.Error())
	}
}

func (g *cocoaApp) dark() bool {
	if g.theme == "dark" {
		return true
	}
	if g.theme == "light" {
		return false
	}
	return g.systemDark
}

func (g *cocoaApp) setRunning(run bool) {
	g.running = run
	C.setEnabled(g.send, C.int(boolToInt(!run)))
	C.setEnabled(g.stop, C.int(boolToInt(run)))
}

func (g *cocoaApp) setStatus(s string) {
	c := C.CString(s)
	C.labelSet(g.status, c)
	C.free(unsafe.Pointer(c))
}

func (g *cocoaApp) setPill(s string) {
	c := C.CString(s)
	C.labelSet(g.pill, c)
	C.free(unsafe.Pointer(c))
}

func (g *cocoaApp) setText(tv unsafe.Pointer, s string) {
	c := C.CString(s)
	C.setViewText(tv, c)
	C.free(unsafe.Pointer(c))
}

func (g *cocoaApp) composerText() string {
	cs := C.viewText(g.compos)
	defer C.free(unsafe.Pointer(cs))
	return C.GoString(cs)
}

func (g *cocoaApp) send() {
	text := strings.TrimSpace(g.composerText())
	if text == "" {
		return
	}
	g.messages = append(g.messages, guiservice.MsgView{Role: "user", Text: text})
	g.streamText.Reset()
	g.renderChat()
	g.setText(g.compos, "")
	g.setRunning(true)
	if err := g.eng.Send(text); err != nil {
		g.setRunning(false)
		g.setStatus("Error: " + err.Error())
	}
}

func (g *cocoaApp) renderChat() {
	var b strings.Builder
	for i, m := range g.messages {
		if i > 0 {
			b.WriteString("\n\n")
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

func (g *cocoaApp) drain() {
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

func (g *cocoaApp) appendAssistant(text string) {
	if len(g.messages) > 0 && g.messages[len(g.messages)-1].Role == "assistant" {
		g.messages[len(g.messages)-1].Text = text
	} else {
		g.messages = append(g.messages, guiservice.MsgView{Role: "assistant", Text: text})
	}
	g.renderChat()
}

func (g *cocoaApp) handleApproval(ev Event) {
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
	c := C.CString(req.Desc)
	ok := C.alertAsk(c) == 1
	C.free(unsafe.Pointer(c))
	g.eng.Approve(req.ID, ok)
}

func (g *cocoaApp) reload() {
	info := g.eng.GetInfo()
	if info == nil {
		return
	}
	g.provider = info.Provider
	g.model = info.Model
	g.onboarded = info.Onboarded
	g.activeName = info.Name
	g.messages = info.Messages
	g.sessions = info.Sessions

	pill := info.Provider
	if info.Model != "" {
		pill = info.Provider + " · " + info.Model
	}
	g.setPill(pill)
	g.setPopupSessions()

	g.renderChat()
	g.setRunning(g.eng.IsRunning())
	if !g.onboarded {
		g.setStatus("Set up a provider to start chatting.")
	}
}

func (g *cocoaApp) setPopupSessions() {
	C.popupClear(g.popup)
	C.popupAdd(g.popup, "Sessions")
	for _, s := range g.sessions {
		C.popupAdd(g.popup, s.Name)
	}
	C.popupSetSel(g.popup, 0)
}

func (g *cocoaApp) cycleTheme() {
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

func (g *cocoaApp) applyTheme() {
	mode := 0
	if g.theme == "dark" {
		mode = 1
	} else if g.theme == "light" {
		mode = 2
	}
	C.setWindowAppearance(g.win, C.int(mode))
	dark := C.int(boolToInt(g.dark()))
	C.styleTextView(g.chat, dark)
	C.styleTextView(g.compos, dark)
}