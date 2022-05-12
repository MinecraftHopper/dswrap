package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	file, err := os.Open("paste.html")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(file)
	contents := buf.String()
	tmpl, err := template.New("paste").Parse(contents)

	file, err = os.Open("404.html")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	buf = new(bytes.Buffer)
	buf.ReadFrom(file)
	notfound := buf.String()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		req, err := http.Get("https://cdn.discordapp.com/attachments" + r.URL.Path)
		if err != nil {
			log.Println(err)
			fmt.Fprintln(w, "An error occured fetching from the discord api")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		defer req.Body.Close()
		if req.StatusCode != 200 {
			fmt.Fprintf(w, notfound)
			return
		}
		contents, err := io.ReadAll(req.Body)
		if err != nil {
			log.Println(err)
			fmt.Fprintln(w, "An error occured reading the contents of the response")
			return
		}
		escaped := template.HTMLEscapeString(string(contents))
		corrected := strings.Replace(escaped, "\n", "<br>", -1)
		tmpl.Execute(w, template.HTML(corrected))
		if err != nil {
			log.Println(err)
			fmt.Fprintln(w, "An error occured templating the response")
			return
		}
	})
	log.Println("Listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
