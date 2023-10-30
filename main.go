package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/minecrafthopper/dswrap/env"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

var contentTemplate *template.Template
var client *http.Client

//go:embed *.html
var pasteHtml embed.FS

var cache = make(map[string]*DiscordFile)

func main() {
	client = &http.Client{}

	e := gin.Default()

	templ := template.Must(template.New("pastes").Delims("{{", "}}").ParseFS(pasteHtml, "*"))
	e.SetHTMLTemplate(templ)

	e.GET("/*path", handleRequest)
	e.Run()
}

func handleRequest(c *gin.Context) {
	path := strings.TrimPrefix(c.Param("path"), "/")

	parts := strings.Split(path, "/")

	if len(parts) < 3 {
		c.Status(http.StatusNotFound)
		return
	}

	chanId := parts[0]
	messageId := parts[1]
	filename := parts[2]

	var discordFile *DiscordFile
	var err error
	var exists bool

	if discordFile, exists = cache[path]; !exists || discordFile == nil || discordFile.ExpireAt.Before(time.Now()) {
		discordFile, err = getFileForMessageAttachment(chanId, messageId, filename)
		if err != nil {
			panic(err)
		}
		cache[path] = discordFile
	}

	if discordFile == nil || discordFile.Url == "" {
		c.Status(http.StatusNotFound)
		return
	}

	contents, err := getFileFromCDN(discordFile.Url)
	if err != nil {
		panic(err)
	}

	escaped := template.HTMLEscapeString(string(contents))
	escaped = strings.Replace(escaped, "\n", "<br>", -1)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.HTML(http.StatusOK, "paste.html", gin.H{"body": template.HTML(escaped)})
}

func getFileForMessageAttachment(channelId, messageId, filename string) (file *DiscordFile, err error) {
	request := &http.Request{Header: make(http.Header)}
	request.Header.Add("Authorization", "Bot "+env.Get("discord.token"))
	request.URL, err = url.Parse(fmt.Sprintf("https://discord.com/api/v%s/channels/%s/messages/%s", env.GetOr("discord.api.version", "10"), channelId, messageId))
	if err != nil {
		return
	}

	var response *http.Response
	log.Printf("Calling %s", request.URL.String())
	response, err = client.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err = errors.New("unexpected status code: " + response.Status)
		return
	}

	var msg DiscordMessage
	err = json.NewDecoder(response.Body).Decode(&msg)
	if err != nil {
		return
	}

	for _, v := range msg.Attachments {
		if v.Filename == filename {
			expireAt := time.Now()

			u := v.Url
			var ur *url.URL
			ur, err = url.Parse(u)
			if err == nil {
				ex := ur.Query().Get("ex")
				var epoch int64
				epoch, err = strconv.ParseInt(ex, 16, 32)
				if err != nil {
					return
				}
				expireAt = time.Unix(epoch, 0)
			}

			file = &DiscordFile{
				Url:      v.Url,
				ExpireAt: expireAt,
			}
			return
		}
	}

	return
}

func getFileFromCDN(path string) ([]byte, error) {
	req, err := http.Get(path)
	if err != nil {
		return nil, err
	}

	defer req.Body.Close()

	if req.StatusCode != http.StatusOK {
		return nil, errors.New("invalid status code: " + req.Status)
	}
	contents, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	fileEncoding := req.Header.Get("Content-Type")
	if strings.Contains(strings.ToLower(fileEncoding), "charset=utf-16") {
		contents, err = DecodeUTF16(contents)
		if err != nil {
			return nil, err
		}
	}

	return contents, nil
}

func DecodeUTF16(b []byte) ([]byte, error) {
	if len(b)%2 != 0 {
		return make([]byte, 0), fmt.Errorf("must have even length byte slice")
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

type DiscordMessage struct {
	Id          string              `json:"id"`
	Attachments []DiscordAttachment `json:"attachments"`
}

type DiscordAttachment struct {
	Id       string `json:"id"`
	Filename string `json:"filename"`
	Url      string `json:"url"`
}

type DiscordFile struct {
	Url      string
	ExpireAt time.Time
}
