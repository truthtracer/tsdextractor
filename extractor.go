package tsdextractor

import (
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
)

var ignoreTag []string = []string{"style", "script", "link", "video", "iframe", "source", "picture", "header", "noscript"}

var ignoreClass []string = []string{"share", "contribution", "copyright", "copy-right", "disclaimer", "recommend", "related", "footer", "comment", "social", "submeta", "report-infor"}

var canBeRemoveIfEmpty []string = []string{"section", "h1", "h2", "h3", "h4", "h5", "h6", "span"}

type Article struct {
	Title       string   `json:"title,omitempty"`
	Images      []string `json:"images,omitempty"`
	Author      string   `json:"author,omitempty"`
	PublishTime string   `json:"publish_time,omitempty"`
	Content     string   `json:"content,omitempty"`
	ContentHTML string   `json:"content_html,omitempty"`
}

type headEntry struct {
	key string
	val string
}

func Extract(source string) (*Article, error) {
	dom, err := goquery.NewDocumentFromReader(strings.NewReader(source))
	if err != nil {
		return nil, err
	}
	body := dom.Find("body")
	normalize(body)
	result := &Article{}
	headText := headTextExtract(dom)
	wg := &sync.WaitGroup{}
	wg.Add(3)
	go func() {
		result.PublishTime = timeExtract(headText, body)
		wg.Done()
	}()
	go func() {
		result.Author = authorExtract(headText, body)
		wg.Done()
	}()
	go func() {
		content := contentExtract(body)
		result.Title = titleExtract(headText, dom.Selection, content.node)
		result.Content = content.density.tiText
		result.ContentHTML, _ = content.node.Html()
		var imgs []string
		content.node.Find("img").Each(func(i int, s *goquery.Selection) {
			if src, ok := s.Attr("src"); ok {
				imgs = append(imgs, src)
			}
		})
		result.Images = imgs
		wg.Done()
	}()
	wg.Wait()
	return result, nil
}

func headTextExtract(dom *goquery.Document) []*headEntry {
	var (
		rs       = []*headEntry{}
		head     = dom.Find("head")
		metaSkip = map[string]bool{
			"charset":    true,
			"http-equiv": true,
		}
	)
	for _, v := range iterator(head) {
		if goquery.NodeName(v) != "meta" {
			continue
		}
		for _, v := range v.Nodes {
			key := ""
			val := ""
			for _, v2 := range v.Attr {
				if metaSkip[v2.Key] {
					key = ""
					break
				}
				if v2.Key == "name" || v2.Key == "property" {
					key = strings.ToLower(v2.Val)
				} else if v2.Key == "content" {
					val = v2.Val
				}
			}
			if key != "" && val != "" {
				length := utf8.RuneCountInString(strings.TrimSpace(val))
				if length >= 2 && length <= 50 {
					rs = append(rs, &headEntry{
						key: key,
						val: val,
					})
				}
			}
		}
	}
	sort.Slice(rs, func(i, j int) bool {
		return len(rs[i].key) > len(rs[j].key)
	})
	return rs
}

func normalize(element *goquery.Selection) {
	for _, v := range ignoreTag {
		element.Find(v).Remove()
	}
	for _, v := range iterator(element) {
		tagName := goquery.NodeName(v)
		if tagName == "#comment" {
			v.Remove()
			continue
		}
		if canBeRemove(v) {
			v.Remove()
			continue
		}
		if val, ok := v.Attr("class"); ok {
			for _, class := range ignoreClass {
				if strings.Contains(val, class) {
					v.Remove()
					continue
				}
			}
		}
		if tagName == "p" {
			v.Find("span,strong,em,b").Each(func(i int, child *goquery.Selection) {
				text := child.Text()
				child.ReplaceWithHtml(text)
			})
		}
		//if tagName == "div" && v.Children().Length() <= 0 {
		//	v.Get(0).Data = "p"
		//}
		if goquery.NodeName(v) == "p" {
			if v.Children().Length() <= 0 && len(strings.TrimSpace(v.Text())) == 0 {
				v.Remove()
			}
		}

	}
}

func iterator(s *goquery.Selection) []*goquery.Selection {
	var result []*goquery.Selection
	iteratorNode(s, func(child *goquery.Selection) {
		result = append(result, child)
	})
	return result
}

func iteratorNode(s *goquery.Selection, fn func(*goquery.Selection)) {
	if s == nil {
		return
	}
	fn(s)
	s.Contents().Each(func(i int, c *goquery.Selection) {
		iteratorNode(c, fn)
	})
}

func canBeRemove(s *goquery.Selection) bool {
	for _, v := range canBeRemoveIfEmpty {
		if strings.ToLower(goquery.NodeName(s)) == v {
			if s.Children().Length() <= 0 && strings.TrimSpace(s.Text()) == "" {
				return true
			}
		}
	}
	return false
}
