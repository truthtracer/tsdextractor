package tsdextractor

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

type textDensity struct {
	density float64
	tiText  string
	ti      int
	lti     int
	tgi     int
	ltgi    int
	pi      int
}

type nodeInfo struct {
	density      textDensity
	node         *goquery.Selection
	sbdi         float64
	textTagCount int
	score        float64
}

type infos []*nodeInfo

var (
	DebugFlag = false
)

func (in infos) Len() int {
	return len(in)
}

func (in infos) Swap(i, j int) {
	in[i], in[j] = in[j], in[i]
}

func (in infos) Less(i, j int) bool {
	return in[i].score > in[j].score
}

func contentExtract(body *goquery.Selection) *nodeInfo {
	var info infos
	for _, v := range iterator(body) {
		density := calcTextDensity(v)
		textTagCount := countTextTag(v)
		sbdi := calcSbdi(density)
		info = append(info, &nodeInfo{
			density:      density,
			node:         v,
			sbdi:         sbdi,
			textTagCount: textTagCount,
		})
	}
	std := calcDensityStd(info)
	calcNewScore(info, std)
	sort.Sort(info)

	if DebugFlag {
		fmt.Println("[tsdextractor] debug begin")
		for _, item := range info {
			fmt.Printf("score:%f, density:%f, ti:%d, lti:%d, tgi:%d, ltgi:%d, sbdi:%f, tags:%d, titext:%s\n", item.score,
				item.density.density,
				item.density.ti,
				item.density.lti,
				item.density.tgi,
				item.density.ltgi,
				item.sbdi,
				item.textTagCount,
				item.density.tiText)
		}
		fmt.Println("[tsdextractor] debug end")
	}

	return info[0]
}

func needSkipLtgi(ti int, lti int) bool {
	if lti == 0 {
		return false
	}
	return ti/lti > 10
}

func countTag(elements *goquery.Selection, excludeEmpty bool) int {
	tagCount := 0
	if excludeEmpty {
		for _, v := range iterator(elements) {
			if goquery.NodeName(v) == "p" || goquery.NodeName(v) == "span" {
				if len(v.Text()) == 0 {
					continue
				}
			}
			tagCount++
		}
	} else {
		tagCount = elements.Length()
	}
	return tagCount
}

//	Ti - LTi
//
// TDi = -----------
//
//	TGi - LTGi
//
// Ti: char count of node i
// LTi：char count of links in node i
// TGi：tag count of node i
// LTGi：link tag count of node i
func calcTextDensity(s *goquery.Selection) textDensity {
	tiText := strings.Join(getAllTiText(s), "\n")
	ti := utf8.RuneCountInString(tiText)
	lti := utf8.RuneCountInString((strings.Join(getAllLtiText(s), "\n")))
	tgi := countTag(s.Find(".//*"), true)
	ltgi := s.Find("a").Length()
	pi := countTextTag(s)

	result := textDensity{
		density: 0,
		tiText:  tiText,
		ti:      ti - lti,
		lti:     lti,
		tgi:     tgi,
		ltgi:    ltgi,
		pi:      pi,
	}

	if (tgi - ltgi) == 0 {
		if !needSkipLtgi(ti, pi) {
			return result
		} else {
			ltgi = 0
		}
	}
	result.density = float64(ti-lti) / float64(tgi-ltgi)
	result.density = math.Abs(result.density)
	return result
}

var (
	ctnRx1 = regexp.MustCompile(`/[\n\r]/g`)
	ctnRx2 = regexp.MustCompile(`/\s{1,}/g`)
)

func getAllTiText(s *goquery.Selection) []string {
	var result []string
	for _, v := range iterator(s) {
		if len(v.Nodes) > 0 && v.Get(0).Type == html.TextNode {
			text := v.Text()
			text = ctnRx1.ReplaceAllString(text, " ")
			text = ctnRx2.ReplaceAllString(text, " ")
			text = strings.TrimSpace(text)
			if len(text) > 0 {
				result = append(result, text)
			}
		}
	}
	return result
}

func getAllLtiText(s *goquery.Selection) []string {
	var result []string
	for _, v := range iterator(s) {
		if goquery.NodeName(v) == "a" {
			text := v.Text()
			text = ctnRx1.ReplaceAllString(text, " ")
			text = ctnRx2.ReplaceAllString(text, " ")
			text = strings.TrimSpace(text)
			if len(text) > 0 {
				result = append(result, text)
			}
		}
	}
	return result
}

func countTextTag(s *goquery.Selection) int {
	return s.Find("p").Length()
}

//	Ti - LTi
//
// SbDi = --------------
//
//	Sbi + 1
//
// SbDi: symbol density
// Sbi：symbols count
func calcSbdi(density textDensity) float64 {
	sbi := countPunctuation(density.tiText)
	sbdi := float64(density.ti-density.lti) / float64(sbi+1)
	if sbdi == 0 {
		sbdi = 1
	}
	return sbdi
}

func countPunctuation(text string) int {
	result := 0
	punctuation := `！，。？、；：“”‘’《》%（）,.?:;'"!%()`
	for _, v := range strings.Split(text, "") {
		if strings.ContainsAny(v, punctuation) {
			result++
		}
	}
	return result
}

func calcDensityStd(info infos) float64 {
	var score []float64
	for _, v := range info {
		score = append(score, v.density.density)
	}
	std := std(score)
	return std
}

// score = log(std) * ndi * log10(textTagCount + 2) * log(sbdi)
// std：standard deviation of node-i's text density
// ndi：node-i's text density
// textTagCount: text content tag count, content in <p></p> include span/div
// sbdi：node-i's symbol density
func calcNewScore(info infos, std float64) {
	for _, v := range info {
		//v.score = math.Log(std) * v.density.density * math.Log10(float64(v.textTagCount+2)) * math.Log(v.sbdi)
		v.score = v.density.density * math.Log10(float64(v.textTagCount+2)) * math.Log(v.sbdi)
		if math.IsNaN(v.score) || math.IsInf(v.score, 1) || math.IsInf(v.score, -1) {
			v.score = 0
		}
	}
}
