package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2/pkg/edge"
	"github.com/jchv/go-webview2/webviewloader"
	"golang.org/x/sys/windows"
)

var (
	user32                   = syscall.NewLazyDLL("user32.dll")
	registerClassEx          = user32.NewProc("RegisterClassExW")
	createWindowEx           = user32.NewProc("CreateWindowExW")
	defWindowProc            = user32.NewProc("DefWindowProcW")
	destroyWindow            = user32.NewProc("DestroyWindow")
	postQuitMessage          = user32.NewProc("PostQuitMessage")
	getMessage               = user32.NewProc("GetMessageW")
	translateMessage         = user32.NewProc("TranslateMessage")
	dispatchMessage          = user32.NewProc("DispatchMessageW")
	postMessage              = user32.NewProc("PostMessageW")
	loadIcon                 = user32.NewProc("LoadIconW")
	loadCursor               = user32.NewProc("LoadCursorW")
	getSystemMetrics         = user32.NewProc("GetSystemMetrics")
	getDpiForSystem          = user32.NewProc("GetDpiForSystem")
	isIconic                 = user32.NewProc("IsIconic")
	showWindow               = user32.NewProc("ShowWindow")
	setForegroundWindow      = user32.NewProc("SetForegroundWindow")
	allowSetForegroundWindow = user32.NewProc("AllowSetForegroundWindow")
)

const (
	wmDestroy    = 0x0002
	wmMove       = 0x0003
	wmSize       = 0x0005
	wmActivate   = 0x0006
	wmClose      = 0x0010
	wmMoving     = 0x0216
	wmCloseNow   = 0x8001
	wmUnanswered = 0x8002
)

//go:generate go run github.com/tc-hib/go-winres@v0.3.3 make --arch 386,amd64,arm64

const iconID = 1

const answerWait = 3 * time.Second

type webWindow struct {
	hwnd      uintptr
	view      *edge.Chromium
	ready     bool
	closeSoon bool
	question  closeQuestion
}

var theWindow *webWindow

func newWindow(address, dataDir string) (window, error) {
	if version, err := webviewloader.GetInstalledVersion(); err != nil {
		return nil, fmt.Errorf("look for the WebView2 runtime: %w", err)
	} else if version == "" {
		return nil, fmt.Errorf("%w: the WebView2 runtime is not installed", errNoWindow)
	}
	if err := writableDir(dataDir); err != nil {
		return nil, fmt.Errorf("the window's data folder: %w", err)
	}

	const browserArgs = "WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS"
	if err := os.Setenv(browserArgs, strings.TrimSpace(os.Getenv(browserArgs)+" --no-proxy-server")); err != nil {
		return nil, err
	}

	w := &webWindow{view: edge.NewChromium()}
	w.view.DataPath = dataDir
	w.view.SetPermission(edge.CoreWebView2PermissionKindClipboardRead, edge.CoreWebView2PermissionStateAllow)
	w.view.MessageCallback = w.message
	w.view.NavigationCompletedCallback = func(*edge.ICoreWebView2, *edge.ICoreWebView2NavigationCompletedEventArgs) {
		w.question.reset()
	}
	theWindow = w
	if err := w.create(); err != nil {
		return nil, err
	}
	release, err := tryWebView2(dataDir, w.hwnd)
	if err != nil {
		_, _, _ = destroyWindow.Call(w.hwnd)
		return nil, err
	}
	defer release()
	if !w.view.Embed(w.hwnd) {
		_, _, _ = destroyWindow.Call(w.hwnd)
		return nil, errors.New("WebView2 did not start")
	}
	w.ready = true
	w.view.Resize()
	if settings, err := w.view.GetSettings(); err == nil {

		_ = settings.PutAreDefaultContextMenusEnabled(true)
		_ = settings.PutAreDevToolsEnabled(false)
	}
	w.view.Init(bridgeScript)
	w.view.Navigate(address)
	if w.closeSoon {
		_, _, _ = destroyWindow.Call(w.hwnd)
	}
	return w, nil
}

func (w *webWindow) create() error {
	var instance windows.Handle
	if err := windows.GetModuleHandleEx(0, nil, &instance); err != nil {
		return err
	}
	const idcArrow, colorWindow = 32512, 5
	icon, _, _ := loadIcon.Call(uintptr(instance), iconID)
	cursor, _, _ := loadCursor.Call(0, idcArrow)
	className := windows.StringToUTF16Ptr("HaganeWorks")
	class := wndClassEx{
		wndProc:    windows.NewCallback(windowProc),
		instance:   uintptr(instance),
		icon:       icon,
		cursor:     cursor,
		background: colorWindow + 1,
		className:  className,
	}
	class.size = uint32(unsafe.Sizeof(class))
	if atom, _, err := registerClassEx.Call(uintptr(unsafe.Pointer(&class))); atom == 0 {
		return fmt.Errorf("register the window class: %w", err)
	}
	const smCxScreen, smCyScreen = 0, 1
	dpi, _, _ := getDpiForSystem.Call()
	screenWidth, _, _ := getSystemMetrics.Call(smCxScreen)
	screenHeight, _, _ := getSystemMetrics.Call(smCyScreen)
	width, height := min(1600*dpi/96, screenWidth*9/10), min(1000*dpi/96, screenHeight*9/10)
	const wsOverlappedWindow, swShowDefault = 0x00CF0000, 10
	hwnd, _, err := createWindowEx.Call(0, uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr("Hagane Works"))), wsOverlappedWindow,
		(screenWidth-width)/2, (screenHeight-height)/2, width, height, 0, 0, uintptr(instance), 0)
	if hwnd == 0 {
		return fmt.Errorf("create the window: %w", err)
	}
	w.hwnd = hwnd
	_, _, _ = showWindow.Call(hwnd, swShowDefault)
	return nil
}

type wndClassEx struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   uintptr
	icon       uintptr
	cursor     uintptr
	background uintptr
	menuName   *uint16
	className  *uint16
	iconSm     uintptr
}

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	x, y    int32
	private uint32
}

func pump(done func() bool) {
	var m msg
	for !done() {
		if r, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0); int32(r) <= 0 {
			return
		}
		_, _, _ = translateMessage.Call(uintptr(unsafe.Pointer(&m)))
		_, _, _ = dispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func windowProc(hwnd, message, wParam, lParam uintptr) uintptr {
	w := theWindow
	switch message {
	case wmSize:
		w.view.Resize()
		return 0
	case wmMove, wmMoving:
		_ = w.view.NotifyParentWindowPositionChanged()
		return 0
	case wmActivate:
		if wParam&0xFFFF != 0 {
			w.view.Focus()
		}
		return 0
	case wmClose:
		w.askToClose()
		return 0
	case wmCloseNow:
		_, _, _ = destroyWindow.Call(hwnd)
		return 0
	case wmUnanswered:
		if w.question.unanswered(int(wParam)) {
			_, _, _ = destroyWindow.Call(hwnd)
		}
		return 0
	case wmDestroy:
		if w.ready {
			_, _, _ = postQuitMessage.Call(0)
		}
		return 0
	}
	r, _, _ := defWindowProc.Call(hwnd, message, wParam, lParam)
	return r
}

func (w *webWindow) askToClose() {
	if !w.ready {
		w.closeSoon = true
		return
	}
	id := w.question.start()
	if id == 0 {
		return
	}
	w.view.Eval(closeRequest)
	time.AfterFunc(answerWait, func() { _, _, _ = postMessage.Call(w.hwnd, wmUnanswered, uintptr(id), 0) })
}

func (w *webWindow) message(raw string) {
	var m struct {
		Open  string `json:"open"`
		Close string `json:"close"`
	}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return
	}
	if m.Open != "" {
		if err := openOutside(m.Open); err != nil {
			log.Printf("open %s: %v", m.Open, err)
		}
	}
	if m.Close != "" && w.question.answer(m.Close) {
		_, _, _ = destroyWindow.Call(w.hwnd)
	} else if m.Close == "asking" {
		w.show()
	}
}

func (w *webWindow) run() {
	pump(func() bool { return false })
	closePopups()
}

func (w *webWindow) close() { _, _, _ = postMessage.Call(w.hwnd, wmCloseNow, 0, 0) }

func (w *webWindow) show() {
	const swRestore = 9
	if minimized, _, _ := isIconic.Call(w.hwnd); minimized != 0 {
		_, _, _ = showWindow.Call(w.hwnd, swRestore)
	}
	_, _, _ = setForegroundWindow.Call(w.hwnd)
}

type closeQuestion struct {
	id     int
	asked  int
	asking bool
}

func (q *closeQuestion) start() int {
	if q.id != 0 {
		return 0
	}
	q.asked++
	q.id, q.asking = q.asked, false
	return q.id
}

func (q *closeQuestion) answer(answer string) bool {
	if q.id == 0 {
		return false
	}
	switch answer {
	case "close":
		q.id = 0
		return true
	case "asking":
		q.asking = true
	default:
		q.id = 0
	}
	return false
}

func (q *closeQuestion) unanswered(id int) bool {
	if q.id != id || q.asking {
		return false
	}
	q.id = 0
	return true
}

func (q *closeQuestion) reset() { q.id = 0 }

const bridgeScript = `window.haganeWorks = Object.freeze({
  open: (url) => window.chrome.webview.postMessage(JSON.stringify({ open: String(url) })),
  close: (answer) => window.chrome.webview.postMessage(JSON.stringify({ close: String(answer) })),
});`

const closeRequest = `typeof window.haganeWorksCloseRequested === "function"
  ? window.haganeWorksCloseRequested()
  : window.haganeWorks.close("close");`

func openOutside(address string) error {
	u, err := url.Parse(address)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("not a web address: %q", address)
	}
	return openDefaultBrowser(u.String())
}

func writableDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	probe, err := os.CreateTemp(dir, "write-check-*")
	if err != nil {
		return err
	}
	probe.Close()
	return os.Remove(probe.Name())
}

func tryWebView2(dataDir string, hwnd uintptr) (release func(), err error) {
	environment := newCompletedHandler()
	result, err := webviewloader.CreateCoreWebView2EnvironmentWithOptions(nil, windows.StringToUTF16Ptr(dataDir), 0, uintptr(unsafe.Pointer(environment)))
	if err != nil {
		return nil, fmt.Errorf("load WebView2: %w", err)
	}
	if err := environment.wait(result); err != nil {
		return nil, err
	}
	controller := newCompletedHandler()

	result, _, _ = syscall.SyscallN(environment.object.methods[3], uintptr(unsafe.Pointer(environment.object)), hwnd, uintptr(unsafe.Pointer(controller)))
	if err := controller.wait(result); err != nil {
		environment.object.call(2)
		return nil, err
	}
	return func() {
		controller.object.call(2)
		environment.object.call(2)
	}, nil
}

type comObject struct{ methods *[4]uintptr }

func (o *comObject) call(method int) {
	_, _, _ = syscall.SyscallN(o.methods[method], uintptr(unsafe.Pointer(o)))
}

type completedHandler struct {
	methods *[4]uintptr
	done    bool
	result  uintptr
	object  *comObject
}

func newCompletedHandler() *completedHandler {
	return &completedHandler{methods: &completedHandlerMethods}
}

func (h *completedHandler) wait(result uintptr) error {
	if int32(result) >= 0 {
		pump(func() bool { return h.done })
		result = h.result
	}
	runtime.KeepAlive(h)
	if int32(result) < 0 || h.object == nil {
		return fmt.Errorf("WebView2 could not start (error 0x%08X)", uint32(result))
	}
	return nil
}

var completedHandlerMethods = [4]uintptr{
	windows.NewCallback(func(this *completedHandler, _ uintptr, object *uintptr) uintptr {
		*object = uintptr(unsafe.Pointer(this))
		return 0
	}),
	windows.NewCallback(func(*completedHandler) uintptr { return 1 }),
	windows.NewCallback(func(*completedHandler) uintptr { return 1 }),
	windows.NewCallback(func(this *completedHandler, result uintptr, object *comObject) uintptr {
		this.done, this.result = true, result
		if int32(result) >= 0 && object != nil {
			object.call(1)
			this.object = object
		}
		return 0
	}),
}

func closePopups() {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return
	}
	defer windows.CloseHandle(snapshot)
	browsers := map[uint32]bool{}
	self := uint32(os.Getpid())
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for err := windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		if entry.ParentProcessID == self && strings.EqualFold(filepath.Base(windows.UTF16ToString(entry.ExeFile[:])), "msedgewebview2.exe") {
			browsers[entry.ProcessID] = true
		}
	}
	_ = windows.EnumWindows(windows.NewCallback(func(hwnd windows.HWND, _ uintptr) uintptr {
		var pid uint32
		if _, err := windows.GetWindowThreadProcessId(hwnd, &pid); err == nil && browsers[pid] && windows.IsWindowVisible(hwnd) {
			_, _, _ = postMessage.Call(uintptr(hwnd), wmClose, 0, 0)
		}
		return 1
	}), nil)
}

func allowForeground() {
	const asfwAny = ^uintptr(0)
	_, _, _ = allowSetForegroundWindow.Call(asfwAny)
}
