/*
Copyright 2013 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/nf/sigourney/ui"

	"code.google.com/p/portaudio-go/portaudio"
	"github.com/gorilla/websocket"
	"github.com/rakyll/portmidi"
)

var (
	listenAddr = flag.String("listen", "localhost:8080", "listen address")
	doDemo     = flag.Bool("demo", false, "play demo sound")
	doBrowser  = flag.Bool("browser", true, "open web browser")
)

func main() {
	flag.Parse()

	portaudio.Initialize()
	defer portaudio.Terminate()

	portmidi.Initialize()
	defer portmidi.Terminate()

	if *doDemo {
		if err := demo(); err != nil {
			log.Println(err)
		}
		return
	}

	http.Handle("/", http.FileServer(http.Dir("static")))
	http.HandleFunc("/socket", socketHandler)

	l, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	go http.Serve(l, nil)

	u := fmt.Sprintf("http://%v/", *listenAddr)
	if !*doBrowser || !openBrowser(u) {
		fmt.Printf("Open your web browser to %v\n\n", u)
	}

	fmt.Println("Press enter to quit...")
	os.Stdin.Read([]byte{0})
}

func socketHandler(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if err != nil {
		log.Println(err)
		return
	}

	u, err := ui.New()
	if err != nil {
		log.Println(err)
		return
	}
	defer u.Close()

	go func() {
		for m := range u.M {
			if err := c.WriteJSON(m); err != nil {
				if err != io.EOF {
					log.Println(err)
				}
				return
			}
		}
	}()

	for {
		m := new(ui.Message)
		if err := c.ReadJSON(m); err != nil {
			if err != io.EOF {
				log.Println(err)
			}
			return
		}
		if err := u.Handle(m); err != nil {
			log.Println(err)
		}
	}
}

// openBrowser tries to open the URL in a browser,
// and returns whether it succeed in doing so.
func openBrowser(url string) bool {
	var args []string
	switch runtime.GOOS {
	case "darwin":
		args = []string{"open"}
	case "windows":
		args = []string{"cmd", "/c", "start"}
	default:
		args = []string{"xdg-open"}
	}
	cmd := exec.Command(args[0], append(args[1:], url)...)
	return cmd.Start() == nil
}
