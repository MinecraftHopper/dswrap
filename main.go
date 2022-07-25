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
	"unicode/utf16"
	"unicode/utf8"
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
	if err != nil {
		log.Println(err)
		_, _ = fmt.Fprintln(w, "An error occurred fetching from the discord api")
		return
	}

	defer req.Body.Close()
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
	ctype_encoding := strings.Split(req.Header.Get("Content-Type"), ";%20")
	if strings.ToLower(ctype_encoding[len(ctype_encoding)-1]) == "charset=utf-16" {
		contents, err = DecodeUTF16(contents)
		if err != nil {
			log.Println(err)
			fmt.Fprintln(w, "An error occurred decoding the response")
			return
		}
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

func DecodeUTF16(b []byte) ([]byte, error) {

	if len(b)%2 != 0 {
		return make([]byte, 0), fmt.Errorf("Must have even length byte slice")
	}

	u16s := make([]uint16, 1)

	ret := &bytes.Buffer{}

	b8buf := make([]byte, 4)

	lb := len(b)
	for i := 0; i < lb; i += 2 {
		u16s[0] = uint16(b[i]) + (uint16(b[i+1]) << 8)
		r := utf16.Decode(u16s)
		n := utf8.EncodeRune(b8buf, r[0])
		ret.Write(b8buf[:n])
	}

	return ret.Bytes(), nil
}