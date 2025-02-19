package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
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

func MakeTime(str string) (time.Time, error) {
	const fmt = "2006-01-02T15:04:05"
	reh := regexp.MustCompile(`.+[tT](\d\d)`)
	rem := regexp.MustCompile(`.+[tT](\d\d):(\d\d)`)
	ret := regexp.MustCompile(`.+[tT](\d\d):(\d\d):(\d\d)`)
	rez := regexp.MustCompile(`.+([zZ]|([+\-](\d\d):(\d\d)))`)
	tnow := time.Now()
	location := tnow.Location()
	strs := ""
	if rez.MatchString(str) {
		if tm, err := time.Parse(time.RFC3339, str); err != nil {
			return tnow, err
		} else {
			return tm, nil
		}

	} else if ret.MatchString(str) {
		strs = str
	} else if rem.MatchString(str) {
		strs = str + ":00"
	} else if reh.MatchString(str) {
		strs = str + ":00:00"
	} else {
		strs = str + "T00:00:00"
	}

	if tm, err := time.ParseInLocation(fmt, strs, location); err != nil {
		return tnow, err
	} else {
		return tm, nil
	}
}

func (t TimeValue) Set(str string) error {
	if tm, err := MakeTime(str); err != nil {
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
	title  = flag.String("title", "Pocket Dump", "HTML title attribute of output file")
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
	ItemID         string           `json:"item_id,omitempty"`
	ResolvedID     string           `json:"resolved_id,omitempty"`
	GivenURL       string           `json:"given_url,omitempty"`
	GivenTitle     string           `json:"given_title,omitempty"`
	Favorite       string           `json:"favorite,omitempty"`
	Status         string           `json:"status",omitempty`
	ResolvedTitle  string           `json:"resolved_title,omitempty"`
	ResolvedURL    string           `json:"resolved_url,omitempty"`
	Excerpt        string           `json:"excerpt,omitempty"`
	IsArticle      string           `json:"is_article,omitempty"`
	HasVideo       string           `json:"has_video,omitempty"`
	HasImage       string           `json:"has_image,omitempty"`
	WordCount      string           `json:"word_count,omitempty"`
	Images         map[string]Image `json:"images,omitempty"`
	Videos         map[string]Video `json:"videos,omitempty"`
	TimeAdded      string           `json:"time_added,omitempty"`
	TimeRead       string           `json:"time_read,omitempty"`
	TimeFavorited  string           `json:"time_favorited,omitempty"`
	DomainMeta     DomainMetadata   `json:"domain_metadata,omitempty"`
	SortId         *int             `json:"sort_id,omitempty"`
	Lang           string           `json:"lang,omitempty"`
	IsIndex        string           `json:"is_index,omitempty"`
	ListenEstimate *int             `json:"listen_duration_estimate,omitempty"`
}

type retrieveResponse struct {
	Status     int                   `json:"status"`
	List       map[string]PocketItem `json:"list"`
	Since      int64                 `json:"since"`
	Complete   int                   `json:"complete"`
	SearchMeta map[string]string     `json:"search_meta"`
	Error      map[string]string     `json:"error"`
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func checkJSON(err error) {
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Badly-formed JSON: Error at position %d", syntaxError.Offset)
			panic(msg)
		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := fmt.Sprintf("Badly-formed JSON: Unexpected EOF")
			panic(msg)
		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("JSON contains an invalid value for the %q field at position %d", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			panic(msg)
		case errors.Is(err, io.EOF):
			panic("JSON file cannot be empty")
		default:
			panic(err)
		}
	}
}

func main() {
	*tend = time.Now()
	flag.Var(&TimeValue{tstart}, "start", "dump bookmarks from this date and after, RFC 3339 format with optional time and time zone, default to local time (2017-11-01[T00:00:00[-07:00]]) (Default is beginning of file)")
	flag.Var(&TimeValue{tend}, "end", "dump bookmarks from this date and before, in RFC 3339 format with optional time and time zone, default to local time (2017-11-01[T00:00:00[-07:00]]) (Default is end of file)")
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
	checkJSON(err)

	items := make(map[int64]PocketItem)
	var keys []int64
	for _, v := range dump.List {
		if (v.Status != "0") {
			continue
		}

		stamp, errn := strconv.ParseInt(v.TimeAdded, 10, 64)
		check(errn)
		items[stamp] = v
		keys = append(keys, stamp)
	}

	var st int64
	var et int64
	if *rev {
		sort.Slice(keys, func(i, j int) bool { return keys[i] > keys[j] })
		st = keys[len(keys)-1]
		if !tstart.IsZero() {
			st = tstart.Unix()
		}

		et = keys[0]
		if tend.Before(time.Unix(et, 0)) {
			et = tend.Unix()
		}

	} else {
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		st = keys[0]
		if !tstart.IsZero() {
			st = tstart.Unix()
		}

		et = keys[len(keys)-1]
		if tend.Before(time.Unix(et, 0)) {
			et = tend.Unix()
		}
	}

	fmt.Fprintf(writer, "<!DOCTYPE html><html>\n<head><meta charset=\"utf-8\"><title>%s</title></head><body>\n", html.EscapeString(*title))
	if *drange {
		fmt.Fprintf(writer, "<h1>%s - %s</h1>\n", time.Unix(keys[0], 0), time.Unix(keys[len(keys)-1], 0))
	}

	fmt.Fprintf(writer, "<ol>\n")
	for _, key := range keys {
		if key < st {
			continue
		}

		if key > et {
			continue
		}

		v := items[key]

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
		var title string
		if (len(v.GivenTitle) == 0) && (len(v.ResolvedTitle) == 0) {
			title = v.GivenURL
		} else if len(v.GivenTitle) == 0 {
			title = v.ResolvedTitle
		} else {
			title = v.GivenTitle
		}

		if ret != nil {
			if !ret.Match([]byte(title)) {
				continue
			}
		}

		fmt.Fprintf(writer, "<li>%s <a href=\"%s\">%s</a></li>\n", when.Format(time.UnixDate), v.GivenURL, html.EscapeString(title))
	}

	fmt.Fprintf(writer, "</ol>\n</body></html>")

	writer.Flush()
}
