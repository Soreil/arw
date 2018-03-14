package arw

import (
	"image"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/gdk"
)

func display(img *image.RGBA, name string) {
	gtk.Init(nil)

	wnd,err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		panic(err)
	}
	pbuf, err := gdk.PixbufNew(gdk.COLORSPACE_RGB,true,8,img.Bounds().Dx(),img.Bounds().Dy())
	if err != nil {
		panic(err)
	}

	buf := pbuf.GetPixels()
	copy(buf,img.Pix)

	pbuf, err = pbuf.RotateSimple(90)
	if err != nil {
		panic(err)
	}

	pbuf,err = pbuf.ScaleSimple(pbuf.GetWidth()/8,pbuf.GetHeight()/8,gdk.INTERP_BILINEAR)
	if err != nil {
		panic(err)
	}

	gtkimg,err := gtk.ImageNewFromPixbuf(pbuf)
	if err != nil {
		panic(err)
	}
	wnd.Add(gtkimg)
	gtkimg.Show()



	wnd.Show()
	gtk.Main()
}
