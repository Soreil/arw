package arw

import (
	"fmt"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"image"
	"log"
	"os"
	"strings"
)

func display(img *image.RGBA, name string, raw rawDetails) {
	gtk.Init(nil)

	builder, err := gtk.BuilderNew()
	if err != nil {
		panic(err)
	}
	err = builder.AddFromFile("../ui.glade")
	if err != nil {
		log.Println(os.Getwd())
		panic(err)
	}
	obj, err := builder.GetObject("mainImage")
	if err != nil {
		panic(err)
	}
	var gtkimg *gtk.Image
	if b, ok := obj.(*gtk.Image); ok {
		gtkimg = b
	}

	obj, err = builder.GetObject("mainWindow")
	if err != nil {
		panic(err)
	}

	var gtkwindow *gtk.Window
	if b, ok := obj.(*gtk.Window); ok {
		gtkwindow = b
	}

	obj, err = builder.GetObject("mainTextView")
	if err != nil {
		panic(err)
	}

	var gtkTextView *gtk.TextView
	if b, ok := obj.(*gtk.TextView); ok {
		gtkTextView = b
	}

	obj, err = builder.GetObject("mainStatusBar")
	if err != nil {
		panic(err)
	}

	var gtkStatusBar *gtk.Statusbar
	if b, ok := obj.(*gtk.Statusbar); ok {
		gtkStatusBar = b
	}

	//SETTING UP IMAGE BUFFER
	pbuf, err := gdk.PixbufNew(gdk.COLORSPACE_RGB, true, 8, img.Bounds().Dx(), img.Bounds().Dy())
	if err != nil {
		panic(err)
	}

	buf := pbuf.GetPixels()
	copy(buf, img.Pix)

	pbuf, err = pbuf.RotateSimple(0)
	if err != nil {
		panic(err)
	}

	pbuf, err = pbuf.ScaleSimple(pbuf.GetWidth()/6, pbuf.GetHeight()/6, gdk.INTERP_BILINEAR)
	if err != nil {
		panic(err)
	}
	//SET UP IMAGE BUFFER

	gtkimg.SetFromPixbuf(pbuf)
	if err != nil {
		panic(err)
	}

	gtkwindow.Connect("delete-event", func() {
		gtk.MainQuit()
	})

	gtkwindow.SetTitle(name)

	tbuf, err := gtkTextView.GetBuffer()
	if err != nil {
		panic(err)
	}

	details := fmt.Sprintf("%+v", raw)

	var lastpos int
	for range details {
		pos := strings.IndexRune(details[lastpos:], ':')
		if pos == -1 {
			break
		}
		lastpos += pos
		toReplace := strings.LastIndex(details[:lastpos], " ")
		if toReplace == -1 {
			lastpos++
			continue
		} else {
			details = details[:toReplace] + "\n" + details[toReplace+1:]
		}
	}

	tbuf.SetText(details)

	gtkStatusBar.Push(0, "We were statusbars and stuff")

	gtkimg.Show()
	gtkwindow.Show()

	gtk.Main()
}
