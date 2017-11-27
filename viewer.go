package arw

import (
	"image"
	"log"
	"syscall"
	"time"
	"unsafe"
)

var (
	gdi32 = syscall.NewLazyDLL("gdi32.dll")

	pPatBlt        = gdi32.NewProc("PatBlt")
	pStretchDIBits = gdi32.NewProc("StretchDIBits")
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
)

func getModuleHandle() (syscall.Handle, error) {
	ret, _, err := pGetModuleHandleW.Call(uintptr(0))
	if ret == 0 {
		return 0, err
	}
	return syscall.Handle(ret), nil
}

var (
	user32 = syscall.NewLazyDLL("user32.dll")

	pCreateWindowExW  = user32.NewProc("CreateWindowExW")
	pDefWindowProcW   = user32.NewProc("DefWindowProcW")
	pDestroyWindow    = user32.NewProc("DestroyWindow")
	pDispatchMessageW = user32.NewProc("DispatchMessageW")
	pGetMessageW      = user32.NewProc("GetMessageW")
	pLoadCursorW      = user32.NewProc("LoadCursorW")
	pPostQuitMessage  = user32.NewProc("PostQuitMessage")
	pRegisterClassExW = user32.NewProc("RegisterClassExW")
	pTranslateMessage = user32.NewProc("TranslateMessage")

	pBeginPaint = user32.NewProc("BeginPaint")
	pEndPaint   = user32.NewProc("EndPaint")
)

const (
	cSW_SHOW        = 5
	cSW_USE_DEFAULT = 0x80000000
)

/* Ternary raster operations */
const (
	SRCCOPY     = 0x00CC0020 /* dest = source                   */
	SRCPAINT    = 0x00EE0086 /* dest = source OR dest           */
	SRCAND      = 0x008800C6 /* dest = source AND dest          */
	SRCINVERT   = 0x00660046 /* dest = source XOR dest          */
	SRCERASE    = 0x00440328 /* dest = source AND (NOT dest )   */
	NOTSRCCOPY  = 0x00330008 /* dest = (NOT source)             */
	NOTSRCERASE = 0x001100A6 /* dest = (NOT src) AND (NOT dest) */
	MERGECOPY   = 0x00C000CA /* dest = (source AND pattern)     */
	MERGEPAINT  = 0x00BB0226 /* dest = (NOT source) OR dest     */
)

const (
	cWS_MAXIMIZE_BOX = 0x00010000
	cWS_MINIMIZEBOX  = 0x00020000
	cWS_THICKFRAME   = 0x00040000
	cWS_SYSMENU      = 0x00080000
	cWS_CAPTION      = 0x00C00000
	cWS_VISIBLE      = 0x10000000

	cWS_OVERLAPPEDWINDOW = 0x00CF0000
)

type winBool uint32

type rect struct {
	left   uint32
	top    uint32
	right  uint32
	bottom uint32
}

type paint struct {
	hdc         syscall.Handle
	erase       winBool
	rc          rect
	restore     winBool
	incUpdate   winBool
	rgbReserved [32]byte
}

const (
	PATCOPY   = 0x00F00021
	PATPAINT  = 0x00FB0A09
	PATINVERT = 0x005A0049
	DSTINVERT = 0x00550009
	BLACKNESS = 0x00000042
	WHITENESS = 0x00FF0062
)

type bitmapinfo struct {
	size           uint32
	width          int32
	height         int32
	planes         uint16
	bitcount       uint16
	compression    uint32
	sizeimage      uint32
	xpelspermeter  int32
	ypelspermeter  int32
	biclrused      uint32
	biclrimportant uint32
}

func stretchDIBits(
	hdc syscall.Handle,
	XDest, YDest, nDestWidth, nDestHeight, XSrc, YSrc, nSrcWidth, nSrcHeight int32,
	bits unsafe.Pointer,
	bitsInfo bitmapinfo,
	usage uint,
	rop int) (int, error) {
	ret, _, err := pStretchDIBits.Call(
		uintptr(hdc),
		uintptr(XDest),
		uintptr(YDest),
		uintptr(nDestWidth),
		uintptr(nDestHeight),
		uintptr(XSrc),
		uintptr(YSrc),
		uintptr(nSrcWidth),
		uintptr(nSrcHeight),
		uintptr(bits),
		uintptr(unsafe.Pointer(&bitsInfo)),
		uintptr(usage),
		uintptr(rop),
	)
	if ret == 0 {
		return 0, err
	}

	return int(ret), nil
}

func patBlt(hdc syscall.Handle, nXLeft, nYLeft, nWidth, nHeight int, pattern int) (bool, error) {
	ret, _, err := pPatBlt.Call(
		uintptr(hdc),
		uintptr(nXLeft),
		uintptr(nYLeft),
		uintptr(nWidth),
		uintptr(nHeight),
		uintptr(pattern),
	)
	if int32(ret) == -1 {
		return false, err
	}
	return int32(ret) != 0, nil
}

func beginPaint(hwnd syscall.Handle, pnt *paint) (syscall.Handle, error) {
	ret, _, err := pBeginPaint.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(pnt)),
	)
	if ret == 0 {
		return 0, err
	}
	return syscall.Handle(ret), nil
}

func endPaint(hwnd syscall.Handle, pnt *paint) {
	pEndPaint.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(pnt)),
	)
}

func createWindow(className, windowName string, style uint32, x, y, width, height int64, parent, menu, instance syscall.Handle) (syscall.Handle, error) {
	ret, _, err := pCreateWindowExW.Call(
		uintptr(0),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(className))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(windowName))),
		uintptr(style),
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		uintptr(parent),
		uintptr(menu),
		uintptr(instance),
		uintptr(0),
	)
	if ret == 0 {
		return 0, err
	}
	return syscall.Handle(ret), nil
}

const (
	cWM_DESTROY = 0x0002
	cWM_CLOSE   = 0x0010
	cWM_PAINT   = 0x000F
)

func defWindowProc(hwnd syscall.Handle, msg uint32, wparam, lparam uintptr) uintptr {
	ret, _, _ := pDefWindowProcW.Call(
		uintptr(hwnd),
		uintptr(msg),
		uintptr(wparam),
		uintptr(lparam),
	)
	return uintptr(ret)
}

func destroyWindow(hwnd syscall.Handle) error {
	ret, _, err := pDestroyWindow.Call(uintptr(hwnd))
	if ret == 0 {
		return err
	}
	return nil
}

type point struct {
	x, y int32
}

type message struct {
	hwnd    syscall.Handle
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

func dispatchMessage(msg *message) {
	pDispatchMessageW.Call(uintptr(unsafe.Pointer(msg)))
}

func getMessage(msg *message, hwnd syscall.Handle, msgFilterMin, msgFilterMax uint32) (bool, error) {
	ret, _, err := pGetMessageW.Call(
		uintptr(unsafe.Pointer(msg)),
		uintptr(hwnd),
		uintptr(msgFilterMin),
		uintptr(msgFilterMax),
	)
	if int32(ret) == -1 {
		return false, err
	}
	return int32(ret) != 0, nil
}

const (
	cIDC_ARROW = 32512
)

func loadCursorResource(cursorName uint32) (syscall.Handle, error) {
	ret, _, err := pLoadCursorW.Call(
		uintptr(0),
		uintptr(uint16(cursorName)),
	)
	if ret == 0 {
		return 0, err
	}
	return syscall.Handle(ret), nil
}

func postQuitMessage(exitCode int32) {
	pPostQuitMessage.Call(uintptr(exitCode))
}

const (
	cCOLOR_WINDOW = 5
)

type tWNDCLASSEXW struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   syscall.Handle
	icon       syscall.Handle
	cursor     syscall.Handle
	background syscall.Handle
	menuName   *uint16
	className  *uint16
	iconSm     syscall.Handle
}

func registerClassEx(wcx *tWNDCLASSEXW) (uint16, error) {
	ret, _, err := pRegisterClassExW.Call(
		uintptr(unsafe.Pointer(wcx)),
	)
	if ret == 0 {
		return 0, err
	}
	return uint16(ret), nil
}

func translateMessage(msg *message) {
	pTranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
}

func display(img *image.RGBA) {
	log.Println(time.Now().Local(), "GUI start")

	className := "testClass"

	instance, err := getModuleHandle()
	if err != nil {
		log.Println(err)
		return
	}

	cursor, err := loadCursorResource(cIDC_ARROW)
	if err != nil {
		log.Println(err)
		return
	}

	fn := func(hwnd syscall.Handle, msg uint32, wparam, lparam uintptr) uintptr {
		switch msg {
		case cWM_CLOSE:
			destroyWindow(hwnd)
		case cWM_DESTROY:
			postQuitMessage(0)
		case cWM_PAINT:
			log.Println(time.Now().Local(), "drawing start")

			var p paint
			deviceContext, err := beginPaint(hwnd, &p)
			if err != nil {
				panic(err)
			}
			x := p.rc.left
			y := p.rc.top
			height := p.rc.bottom - p.rc.top
			width := p.rc.right - p.rc.left
			//log.Println("Planning on redering:",x,y,height,width)
			var binfo bitmapinfo

			binfo.height = -int32(img.Rect.Dy()) //Negative height in BMP means Windows will interpret it as having a top left origin
			binfo.width = int32(img.Rect.Dx())
			binfo.bitcount = 32
			binfo.planes = 1
			binfo.size = uint32(unsafe.Sizeof(binfo))

			//TODO(sjon): figure out proper origin from which to draw the buffer to be scaled, also a proper size would help
			//This code is currently only useful for displaying the initial picture.
			lines, err := stretchDIBits(deviceContext, int32(x), int32(y), int32(width), int32(height), 0, 0, binfo.width, -binfo.height, unsafe.Pointer(&img.Pix[0]), binfo, 0, SRCCOPY)
			if err != nil {
				log.Println(syscall.GetLastError())
			} else {
				log.Println(lines)
			}

			endPaint(hwnd, &p)
			log.Println(time.Now().Local(), "drawing done")

		default:
			ret := defWindowProc(hwnd, msg, wparam, lparam)
			return ret
		}
		return 0
	}

	wcx := tWNDCLASSEXW{
		wndProc:    syscall.NewCallback(fn),
		instance:   instance,
		cursor:     cursor,
		background: cCOLOR_WINDOW + 1,
		className:  syscall.StringToUTF16Ptr(className),
	}
	wcx.size = uint32(unsafe.Sizeof(wcx))

	if _, err = registerClassEx(&wcx); err != nil {
		log.Println(err)
		return
	}

	_, err = createWindow(
		className,
		"Test Window",
		cWS_VISIBLE|cWS_OVERLAPPEDWINDOW,
		cSW_USE_DEFAULT,
		cSW_USE_DEFAULT,
		int64(1800),
		int64(1200),
		0,
		0,
		instance,
	)
	if err != nil {
		log.Println(err)
		return
	}

	for {
		msg := message{}
		gotMessage, err := getMessage(&msg, 0, 0, 0)
		if err != nil {
			log.Println(err)
			return
		}
		if gotMessage {
			translateMessage(&msg)
			dispatchMessage(&msg)
		} else {
			break
		}
	}
}
