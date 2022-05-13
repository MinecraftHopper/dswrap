package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

var contentTemplate *template.Template

func main() {
	file, err := os.ReadFile("paste.html")
	if err != nil {
		log.Fatal(err)
	}

	contents := string(file)
	contentTemplate, err = template.New("paste").Parse(contents)

	file, err = os.ReadFile("404.html")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", handleRequest)
	log.Println("Listening on port 8080")
	_ = http.ListenAndServe(":8080", nil)
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	req, err := http.Get("https://cdn.discordapp.com/attachments" + r.URL.Path)
	defer req.Body.Close()

	if err != nil {
		log.Println(err)
		_, _ = fmt.Fprintln(w, "An error occurred fetching from the discord api")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if req.StatusCode != http.StatusOK {
		_, _ = fmt.Fprintf(w, "File not found or discord error")
		return
	}
	contents, err := io.ReadAll(req.Body)
	if err != nil {
		log.Println(err)
		_, _ = fmt.Fprintln(w, "An error occurred reading the contents of the response")
		return
	}
	escaped := template.HTMLEscapeString(string(contents))
	corrected := strings.Replace(escaped, "\n", "<br>", -1)

	err = contentTemplate.Execute(w, template.HTML(corrected))
	if err != nil {
		log.Println(err)
		fmt.Fprintln(w, "An error occurred templating the response")
		return
	}
}
