package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/minecrafthopper/dswrap/env"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

var client *http.Client

//go:embed paste.html
var pasteHtml embed.FS
var cache = make(map[string]*DiscordFile)

var mergedFS = http.FS(NewMergedFS(os.DirFS("."), pasteHtml))

func main() {
	client = &http.Client{}

	e := gin.Default()

	e.GET("/cdn/:channelId/:messageId/:filename", getFromCDN)
	e.NoRoute(getRenderHTML)
	err := e.Run()
	if !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

func getFromCDN(c *gin.Context) {
	chanId := c.Param("channelId")
	messageId := c.Param("messageId")
	filename := c.Param("filename")

	if chanId == "" || messageId == "" || filename == "" {
		c.Status(http.StatusNotFound)
		return
	}

	path := fmt.Sprintf("%s/%s/%s", chanId, messageId, filename)

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
	defer contents.Body.Close()

	c.Header("Content-Type", contents.Header.Get("Content-Type"))
	_, err = io.Copy(c.Writer, contents.Body)
}

func getRenderHTML(c *gin.Context) {
	c.FileFromFS("paste.html", mergedFS)
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

func getFileFromCDN(path string) (*http.Response, error) {
	req, err := http.Get(path)
	if err != nil {
		return nil, err
	}

	if req.StatusCode != http.StatusOK {
		return nil, errors.New("invalid status code: " + req.Status)
	}

	return req, nil
}

type DiscordMessage struct {
	Id          string              `json:"id"`
	Attachments []DiscordAttachment `json:"attachments"`
}

type DiscordAttachment struct {
	Id          string `json:"id"`
	Filename    string `json:"filename"`
	Url         string `json:"url"`
	ContentType string `json:"content_type"`
}

type DiscordFile struct {
	Url      string
	ExpireAt time.Time
}
