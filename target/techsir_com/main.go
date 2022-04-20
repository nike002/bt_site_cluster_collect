package techsir_com

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/cgghui/bt_site_cluster_collect/collect"
	"github.com/cgghui/cgghui"
	"io"
	"net/http"
	"os"
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

func (c CollectGo) ArticleDetail(art *collect.Article) error {
	var err error
	if art.Href == "" {
		return collect.ErrUndefinedArticleHref
	}
	var cache *os.File
	cacheFilePath := "./cache/" + cgghui.MD5(c.HomeURL+art.Href) + ".html"
	if cache, err = os.Open(cacheFilePath); err != nil {
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
		if cache, err = os.Create(cacheFilePath); err == nil {
			_, _ = io.Copy(cache, resp.Body)
		}
		_, _ = cache.Seek(0, io.SeekStart)
	}
	var doc *goquery.Document
	if doc, err = goquery.NewDocumentFromReader(cache); err != nil {
		return err
	}
	art.Title = doc.Find(".title").Text()
	art.PostTime, err = time.Parse("2006-01-02", doc.Find(".time").Text())
	if err == nil {
		art.PostTime = art.PostTime.Local()
	}
	if art.LocalImages == nil {
		art.LocalImages = make([]string, 0)
	}
	// 处理图片
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
		img.RemoveAttr("srcset")
		img.SetAttr("src", imgPath)
		art.LocalImages = append(art.LocalImages, imgPath)
	})
	if art.Tag == nil {
		art.Tag = make([]collect.ArticleTag, 0)
	}
	// 处理标签
	doc.Find(".kg-card-markdown .infotextkey").Each(func(_ int, k *goquery.Selection) {
		tag := k.AttrOr("href", "")
		if strings.Contains(tag, "/s/") {
			tag = strings.SplitN(tag, "/s/", 2)[1]
			tag = strings.TrimRight(tag, "/")
		} else {
			tag = ""
		}
		art.Tag = append(art.Tag, collect.ArticleTag{
			Name: k.Text(),
			Tag:  tag,
		})
		k.RemoveAttr("href")
		k.RemoveAttr("target")
		k.RemoveClass("infotextkey")
		k.AddClass("tag")
	})
	art.Content, _ = doc.Find(".kg-card-markdown").Html()
	art.Content = strings.TrimSpace(art.Content)
	return nil
}
