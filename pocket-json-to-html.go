package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strconv"
	"time"
)

type TimeValue struct {
	Time *time.Time
}

func (t TimeValue) String() string {
	if t.Time != nil {
		return t.Time.String()
	}

	return ""
}

func (t TimeValue) Set(s string) error {
	if tm, err := time.Parse(time.RFC3339, s); err != nil {
		return err
	} else {
		*t.Time = tm
	}

	return nil
}

var tstart = &time.Time{}
var tend = &time.Time{}

var (
	inf    = flag.String("in", "/dev/stdin", "input file in JSON")
	outf   = flag.String("out", "/dev/stdout", "output file in HTML")
	drange = flag.Bool("range", false, "print range of dates represented in the dump")
	rev    = flag.Bool("reverse", false, "sort reverse-chronologically (most recent first)")
	upat   = flag.String("url-regex", "", "print only bookmarks where URL matches regex")
	tpat   = flag.String("title-regex", "", "print only bookmarks where title matches regex")
	max    = flag.Int("max", -1, "maximum number of bookmarks printed, -1 for unlimited")
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
	flag.Var(&TimeValue{tstart}, "start", "dump bookmarks from this date and after, RFC 3339 format (2017-11-01T00:00:00-07:00) (Default is beginning of file)")
	flag.Var(&TimeValue{tend}, "end", "dump bookmarks from this date and before, in RFC 3339 format (2017-11-01T00:00:00-07:00) (Default is end of file)")
	flag.Parse()

	if tend.Before(*tstart) {
		panic("range is nonsensical")
	}

	if (*max == 0) || (*max < -1) {
		panic("maximum is nonsensical")
	}

	var ret *regexp.Regexp
	if len(*tpat) > 0 {
		ret = regexp.MustCompile(*tpat)
	}

	var reu *regexp.Regexp
	if len(*upat) > 0 {
		reu = regexp.MustCompile(*upat)
	}

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

	if *rev {
		sort.Slice(keys, func(i, j int) bool { return keys[i] > keys[j] })
	} else {
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	}

	if *drange {
		fmt.Fprintf(writer, "%s - %s\n", time.Unix(keys[0], 0), time.Unix(keys[len(keys)-1], 0))
		writer.Flush()
		output.Close()
		return
	}

	st := keys[0]
	if !tstart.IsZero() {
		st = tstart.Unix()
	}

	et := keys[len(keys)-1]
	if !tend.IsZero() {
		et = tend.Unix()
	}

	fmt.Fprintf(writer, "<!DOCTYPE html><html>\n<head><meta charset=\"utf-8\"><title>Pocket Dump</title></head>\n")
	fmt.Fprintf(writer, "<body><ol>\n")
	for _, key := range keys {
		if key < st {
			continue
		}

		if key > et {
			continue
		}

		v := items[key]

		if ret != nil {
			if !ret.Match([]byte(v.GivenTitle)) {
				continue
			}
		}

		if reu != nil {
			if !reu.Match([]byte(v.GivenURL)) {
				continue
			}
		}

		if *max != -1 {
			if *max > 0 {
				*max--
			} else {
				break
			}
		}

		when := time.Unix(key, 0)
		fmt.Fprintf(writer, "<li>")
		if len(v.GivenTitle) == 0 {
			fmt.Fprintf(writer, "%s <a href=\"%s\">%s</a>", when.Format(time.UnixDate), v.GivenURL, v.GivenURL)
		} else {
			fmt.Fprintf(writer, "%s <a href=\"%s\">%s</a>", when.Format(time.UnixDate), v.GivenURL, html.EscapeString(v.GivenTitle))
		}

		fmt.Fprintf(writer, "</li>\n")
	}

	fmt.Fprintf(writer, "</ol>\n</body></html>")

	writer.Flush()
}
