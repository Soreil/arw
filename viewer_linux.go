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

	obj, err = builder.GetObject("scale")
	if err != nil {
		panic(err)
	}

	var gtkZoom *gtk.ScaleButton
	if b, ok := obj.(*gtk.ScaleButton); ok {
		gtkZoom = b
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

	obj, err = builder.GetObject("filename")
	if err != nil {
		panic(err)
	}

	var gtkFilename *gtk.Label
	if b, ok := obj.(*gtk.Label); ok {
		gtkFilename = b
	}

	obj, err = builder.GetObject("mainMenuPopover")
	if err != nil {
		panic(err)
	}

	var gtkMenuPopover *gtk.Popover
	if b, ok := obj.(*gtk.Popover); ok {
		gtkMenuPopover = b
	}

	obj, err = builder.GetObject("aperture")
	if err != nil {
		panic(err)
	}

	var gtkAperture *gtk.Label
	if b, ok := obj.(*gtk.Label); ok {
		gtkAperture = b
	}
	obj, err = builder.GetObject("shutter")
	if err != nil {
		panic(err)
	}

	var gtkShutter *gtk.Label
	if b, ok := obj.(*gtk.Label); ok {
		gtkShutter = b
	}
	obj, err = builder.GetObject("iso")
	if err != nil {
		panic(err)
	}

	var gtkISO *gtk.Label
	if b, ok := obj.(*gtk.Label); ok {
		gtkISO = b
	}

	//SETTING UP IMAGE BUFFER
	backingBuffer, err := gdk.PixbufNew(gdk.COLORSPACE_RGB, true, 8, img.Bounds().Dx(), img.Bounds().Dy())
	if err != nil {
		panic(err)
	}

	buf := backingBuffer.GetPixels()
	copy(buf, img.Pix)

	backingBuffer, err = backingBuffer.RotateSimple(0)
	if err != nil {
		panic(err)
	}

	frontbuffer,err := gdk.PixbufNew(gdk.COLORSPACE_RGB,true,8,backingBuffer.GetWidth(),backingBuffer.GetHeight())

	frontbuffer, err = backingBuffer.ScaleSimple(1500, 1000, gdk.INTERP_BILINEAR)
	if err != nil {
		panic(err)
	}
	//SET UP IMAGE BUFFER

	gtkimg.SetFromPixbuf(frontbuffer)
	if err != nil {
		panic(err)
	}

	gtkZoom.Connect("value-changed", func(sb *gtk.ScaleButton, val float64){
		if val >= 100 {
			return
		}
		frontbuffer,err = backingBuffer.ScaleSimple(int(1500.0*(100.0-val)/100.0),int(1000.0*(100.0-val)/100.0),gdk.INTERP_BILINEAR)
		if err != nil {
			panic(err)
		}

		gtkimg.SetFromPixbuf(frontbuffer)
	})

	gtkwindow.Connect("delete-event", func() {
		gtk.MainQuit()
	})


	tbuf, err := gtkTextView.GetBuffer()
	if err != nil {
		panic(err)
	}

	details := fmt.Sprintf("%+v", raw)

	tbuf.SetText(details)

	gtkFilename.SetText(name)

	gtkAperture.SetText(fmt.Sprint(raw.bitDepth))
	gtkShutter.SetText(fmt.Sprint(raw.height))
	gtkISO.SetText(fmt.Sprint(raw.length))

	//Set up menu, GLADE can't do this yet so we do it by hand.
	box,err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL,8)
	if err != nil {
		panic(err)
	}

	gtkMenuPopover.Add(box)

	button,err := gtk.ButtonNewWithLabel("Henlo")
	if err != nil {
		panic(err)
	}

	box.Add(button)

	box.ShowAll()
	gtkimg.Show()
	gtkwindow.Show()

	gtk.Main()
}
