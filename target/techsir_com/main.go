package techsir_com

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/cgghui/bt_site_cluster_collect/collect"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	Name = "techsir_com"
)

func init() {
	collect.RegisterStandard(Name, func() collect.Standard {
		return &CollectGo{HomeURL: "https://www.techsir.com/"}
	})
}

var Column = map[collect.Tag]string{
	collect.TagCommerce: "ebiz/index{page}.html",
}

type CollectGo struct {
	HomeURL string
}

func (c CollectGo) GetTag() []collect.Tag {
	r := make([]collect.Tag, 0)
	for k := range Column {
		r = append(r, k)
	}
	return r
}

func (c CollectGo) ArticleList(tag collect.Tag, page int) ([]collect.Article, error) {
	if _, ok := Column[tag]; !ok {
		return nil, collect.ErrUndefinedTag
	}
	target := c.HomeURL + Column[tag]
	if page <= 1 {
		target = strings.ReplaceAll(target, "{page}", "")
	} else {
		target = strings.ReplaceAll(target, "{page}", "_"+strconv.Itoa(page))
	}
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	collect.RequestStructure(req, true)
	var resp *http.Response
	if resp, err = collect.HttpClient.Do(req); err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	var doc *goquery.Document
	if doc, err = goquery.NewDocumentFromReader(resp.Body); err != nil {
		return nil, err
	}
	articles := make([]collect.Article, 0)
	doc.Find(".title").Each(func(_ int, h2 *goquery.Selection) {
		if !h2.HasClass("h4") {
			return
		}
		href := h2.Find("a").AttrOr("href", "")
		if href == "" {
			return
		}
		articles = append(articles, collect.Article{
			Title: h2.Find("a").Text(),
			Href:  href,
		})
	})
	return articles, nil
}

func (c CollectGo) ArticleDetail(art *collect.Article, site *collect.Site) error {
	var err error
	if art.Href == "" {
		return collect.ErrUndefinedArticleHref
	}
	var req *http.Request
	if req, err = http.NewRequest(http.MethodGet, c.HomeURL+art.Href, nil); err != nil {
		return err
	}
	collect.RequestStructure(req, true)
	var resp *http.Response
	if resp, err = collect.HttpClient.Do(req); err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	var doc *goquery.Document
	if doc, err = goquery.NewDocumentFromReader(resp.Body); err != nil {
		return err
	}
	art.Title = doc.Find(".title").Text()
	art.PostTime, err = time.Parse("2006-01-02", doc.Find(".time").Text())
	if err == nil {
		art.PostTime = art.PostTime.Local()
	}
	if site != nil {
		doc.Find(".kg-card-markdown img").Each(func(_ int, img *goquery.Selection) {
			src := img.AttrOr("src", "")
			if src == "" {
				return
			}
			var imgPath string
			if imgPath, err = collect.DownloadImage(src); err != nil {
				return
			}
			if alt := img.AttrOr("alt", ""); len(alt) == 0 {
				img.RemoveAttr("alt")
			}
			img.RemoveAttr("data-original")
			img.RemoveAttr("data-link")
			img.SetAttr("src", imgPath)
		})
	}
	art.Content, _ = doc.Find(".kg-card-markdown").Html()
	return nil
}
