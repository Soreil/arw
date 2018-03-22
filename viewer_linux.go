package arw

import (
	"fmt"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"image"
	"log"
	"os"
	"time"
)

func display(img *image.RGBA, fileName, lensName string, focalLength float32, aperture float32, iso int, shutter time.Duration) {
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

	obj, err = builder.GetObject("view")
	if err != nil {
		panic(err)
	}
	var gtkView *gtk.ScrolledWindow
	if b, ok := obj.(*gtk.ScrolledWindow); ok {
		gtkView = b
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

	obj, err = builder.GetObject("lensname")
	if err != nil {
		panic(err)
	}

	var gtkLens *gtk.Label
	if b, ok := obj.(*gtk.Label); ok {
		gtkLens = b
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

	width, _ := gtkView.GetPreferredWidth()
	height, _ := gtkView.GetPreferredHeight()
	rotation := gdk.PixbufRotation(gdk.PIXBUF_ROTATE_COUNTERCLOCKWISE)
	if rotation == gdk.PIXBUF_ROTATE_COUNTERCLOCKWISE {
		width, height = height, width
	}

	backingBuffer, err = backingBuffer.RotateSimple(rotation)
	if err != nil {
		panic(err)
	}

	frontbuffer, err := gdk.PixbufNew(gdk.COLORSPACE_RGB, true, 8, backingBuffer.GetWidth(), backingBuffer.GetHeight())

	frontbuffer, err = backingBuffer.ScaleSimple(width, height, gdk.INTERP_BILINEAR)
	if err != nil {
		panic(err)
	}
	//SET UP IMAGE BUFFER

	gtkimg.SetFromPixbuf(frontbuffer)
	if err != nil {
		panic(err)
	}

	gtkZoom.Connect("value-changed", func(sb *gtk.ScaleButton, val float64) {
		if val >= 100 {
			return
		}
		frontbuffer, err = backingBuffer.ScaleSimple(int(float64(width)*(100.0-val)/100.0), int(float64(height)*(100.0-val)/100.0), gdk.INTERP_BILINEAR)
		if err != nil {
			panic(err)
		}

		gtkimg.SetFromPixbuf(frontbuffer)
	})

	gtkwindow.Connect("delete-event", func() {
		gtk.MainQuit()
	})

	gtkTextView.SetVisible(false)

	tbuf, err := gtkTextView.GetBuffer()
	if err != nil {
		panic(err)
	}

	_ = tbuf

	gtkFilename.SetText(fileName)
	gtkAperture.SetText(fmt.Sprintf("f/%v", aperture))
	gtkShutter.SetText(fmt.Sprintf("%v", shutter))
	gtkISO.SetText(fmt.Sprintf("%d ISO", iso))
	gtkLens.SetText(fmt.Sprintf("%v @ %vmm", lensName[:len(lensName)-1], int(focalLength)))

	//Set up menu, GLADE can't do this yet so we do it by hand.
	box, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	if err != nil {
		panic(err)
	}

	gtkMenuPopover.Add(box)

	button, err := gtk.ButtonNewWithLabel("Henlo")
	if err != nil {
		panic(err)
	}

	box.Add(button)

	box.ShowAll()
	gtkimg.Show()
	gtkwindow.Show()

	gtk.Main()
}
