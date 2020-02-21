package main

import (
	"encoding/json"
	"io/ioutil"
        "bufio"
	"flag"
	"fmt"
	"log"
        "os"
)

var (
	inf  = flag.String("in", "/dev/stdin", "input file in JSON")
	outf = flag.String("out", "/dev/stdout", "output file in HTML")
)

type DomainMetadata struct {
	Name          string `json:"name"`
	GreyscaleLogo string `json:"greyscale_logo"`
	Logo          string `json:"logo"`
}

// https://github.com/simonlindblad/go-pocket
type Image struct {
	ItemID  string `json:"item_id"`
	ImageID string `json:"image_id"`
	Src     string `json:"src"`
	Width   string `json:"width"`
	Height  string `json:"height"`
	Credit  string `json:"credit"`
	Caption string `json:"caption"`
}

type Video struct {
	ItemID  string `json:"item_id"`
	VideoID string `json:"video_id"`
	Src     string `json:"src"`
	Width   string `json:"width"`
	Height  string `json:"height"`
	Type    string `json:"type"`
	Vid     string `json:"vid"`
}

type PocketItem struct {
	ItemID         string           `json:"item_id"`
	ResolvedID     string           `json:"resolved_id"`
	GivenURL       string           `json:"given_url"`
	GivenTitle     string           `json:"given_title"`
	Favorite       string           `json:"favorite"`
	Status         string           `json:"status"`
	ResolvedTitle  string           `json:"resolved_title"`
	ResolvedURL    string           `json:"resolved_url"`
	Excerpt        string           `json:"excerpt"`
	IsArticle      string           `json:"is_article"`
	HasVideo       string           `json:"has_video"`
	HasImage       string           `json:"has_image"`
	WordCount      string           `json:"word_count"`
	Images         map[string]Image `json:"images"`
	Videos         map[string]Video `json:"videos"`
	TimeAdded      string           `json:"time_added"`
	TimeRead       string           `json:"time_read"`
	TimeFavorited  string           `json:"time_favorited"`
	DomainMeta     DomainMetadata   `json:"domain_metadata"`
	SortId         int              `json:"sort_id"`
	Lang           string           `json:"lang"`
	IsIndex        string           `json:"is_index"`
	ListenEstimate int              `json:"listen_duration_estimate"`
}

type retrieveResponse struct {
	Status int                   `json:"status"`
	List   map[string]PocketItem `json:"list"`
}

func check(err error) {
    if err != nil {
        panic(err)
    }
}

func main() {
	flag.Parse()
	input, err := ioutil.ReadFile(*inf)
        check(err)

        output, erro := os.Create(*outf)
        check(erro)
        defer output.Close()

        writer := bufio.NewWriter(output)

	var dump retrieveResponse
	err = json.Unmarshal([]byte(input), &dump)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(dump.Status)
	for _, v := range dump.List {
		if len(v.GivenTitle) == 0 {
			fmt.Fprintf(writer, "<a href=\"%s\">%s</a>\n", v.GivenURL, v.GivenURL)
		} else {
			fmt.Fprintf(writer, "<a href=\"%s\">%s</a>\n", v.GivenURL, v.GivenTitle)
		}
	}

        writer.Flush()
}
