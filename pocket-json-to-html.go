package main

import (
	"encoding/json"
	"io/ioutil"
	"strconv"
        "bufio"
	"sort"
	"time"
	"flag"
	"fmt"
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
	check(err)

	items := make(map[int64]PocketItem)
	var keys []int64
	for _, v := range dump.List {
		stamp, errn := strconv.ParseInt(v.TimeAdded, 10, 64)
		check(errn)
		items[stamp] = v
		keys = append(keys, stamp)
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	fmt.Fprintf(writer, "<!DOCTYPE html><html>\n<head><meta charset=\"utf-8\"><title>Pocket Dump</title></head>\n")
	fmt.Fprintf(writer, "<body><ol>\n")
	for _, key := range keys {
		v := items[key]
		when := time.Unix(key, 0)
		fmt.Fprintf(writer, "<li>")
		if len(v.GivenTitle) == 0 {
			fmt.Fprintf(writer, "%s <a href=\"%s\">%s</a>", when, v.GivenURL, v.GivenURL)
		} else {
			fmt.Fprintf(writer, "%s <a href=\"%s\">%s</a>", when, v.GivenURL, v.GivenTitle)
		}
		
		fmt.Fprintf(writer, "</li>\n")
	}

	fmt.Fprintf(writer, "</ol>\n</body></html>")

        writer.Flush()
}
