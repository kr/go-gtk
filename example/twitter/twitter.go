package main

import (
	"gtk"
	"gdk"
	"gdkpixbuf"
	"http"
	"net"
	"json"
	"bytes"
	"io"
	"os"
	"strings"
)

func HttpGet(url string) (*http.Response, os.Error) {
	var r *http.Response
	var err os.Error
	if proxy := os.Getenv("HTTP_PROXY"); len(proxy) > 0 {
		proxy_url, _ := http.ParseURL(proxy)
		tcp, _ := net.Dial("tcp", "", proxy_url.Host)
		conn := http.NewClientConn(tcp, nil)
		var req http.Request
		req.URL, _ = http.ParseURL(url)
		req.Method = "GET"
		err = conn.Write(&req)
		r, err = conn.Read()
	} else {
		r, _, err = http.Get(url)
	}
	return r, err
}

func url2pixbuf(url string) *gdkpixbuf.GdkPixbuf {
	if r, err := HttpGet(url); err == nil {
		t := r.GetHeader("Content-Type")
		b := make([]byte, r.ContentLength)
		io.ReadFull(r.Body, b)
		var loader *gdkpixbuf.GdkPixbufLoader
		if strings.Index(t, "jpeg") >= 0 {
			loader, _ = gdkpixbuf.PixbufLoaderWithMimeType("image/jpeg")
		} else {
			loader, _ = gdkpixbuf.PixbufLoaderWithMimeType("image/png")
		}
		loader.SetSize(24, 24)
		loader.Write(b)
		loader.Close()
		return loader.GetPixbuf()
	}
	return nil
}

func main() {
	gdk.ThreadsInit()
	gtk.Init(&os.Args)
	window := gtk.Window(gtk.GTK_WINDOW_TOPLEVEL)
	window.SetTitle("Twitter!")
	window.Connect("destroy", func() {
		gtk.MainQuit()
	},
		nil)

	vbox := gtk.VBox(false, 1)

	scrolledwin := gtk.ScrolledWindow(nil, nil)
	textview := gtk.TextView()
	textview.SetEditable(false)
	textview.SetCursorVisible(false)
	scrolledwin.Add(textview)
	vbox.Add(scrolledwin)

	buffer := textview.GetBuffer()

	tag := buffer.CreateTag("blue", map[string]string{
		"foreground": "#0000FF", "weight": "700"})
	button := gtk.ButtonWithLabel("Update Timeline")
	button.SetTooltipMarkup("update <b>public timeline</b>")
	button.Clicked(func() {
		button.SetSensitive(false)
		go func() {
			gdk.ThreadsEnter()
			r, err := HttpGet("http://twitter.com/statuses/public_timeline.json")
			if err == nil {
				b := make([]byte, r.ContentLength)
				io.ReadFull(r.Body, b)
				var j interface{}
				json.NewDecoder(bytes.NewBuffer(b)).Decode(&j)
				arr := j.([]interface{})
				for i := 0; i < len(arr); i++ {
					data := arr[i].(map[string]interface{})
					icon := data["user"].(map[string]interface{})["profile_image_url"].(string)
					var iter gtk.GtkTextIter
					buffer.GetStartIter(&iter)
					buffer.InsertPixbuf(&iter, url2pixbuf(icon))
					name := data["user"].(map[string]interface{})["screen_name"].(string)
					text := data["text"].(string)
					buffer.Insert(&iter, " ")
					buffer.InsertWithTag(&iter, name, tag)
					buffer.Insert(&iter, ":"+text+"\n")
					gtk.MainIteration()
				}
			}
			button.SetSensitive(true)
			gdk.ThreadsLeave()
		}()
	},
		nil)
	vbox.PackEnd(button, false, false, 0)

	window.Add(vbox)
	window.SetSizeRequest(800, 500)
	window.ShowAll()
	gtk.Main()
}
