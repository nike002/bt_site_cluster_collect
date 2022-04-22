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
	collect.TagMobile:   "shuma/index{page}.html",
	collect.TagCar:      "chanye/car/index{page}.html",
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
			Title: strings.TrimSpace(h2.Find("a").Text()),
			Href:  href,
		})
	})
	return articles, nil
}

func (c CollectGo) HasSnapshot(art *collect.Article) bool {
	if art.Href == "" {
		return false
	}
	dir := cgghui.MD5(c.HomeURL + art.Href)
	return collect.PathExists("./snapshot/" + Name + "/" + string(dir[0]) + "/" + dir + ".html")
}

func (c CollectGo) ArticleDetail(art *collect.Article) error {
	var err error
	if art.Href == "" {
		return collect.ErrUndefinedArticleHref
	}
	var cache *os.File
	dir := cgghui.MD5(c.HomeURL + art.Href)
	snapshotPath := "./snapshot/" + Name + "/" + string(dir[0]) + "/"
	_ = os.MkdirAll(snapshotPath, 0666)
	snapshotPath += dir + ".html"
	if cache, err = os.Open(snapshotPath); err != nil {
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
		if cache, err = os.Create(snapshotPath); err == nil {
			_, _ = io.Copy(cache, resp.Body)
		}
		_, _ = cache.Seek(0, io.SeekStart)
	}
	var doc *goquery.Document
	if doc, err = goquery.NewDocumentFromReader(cache); err != nil {
		return err
	}
	art.Title = doc.Find(".title").Text()
	art.Title = strings.TrimSpace(art.Title)
	art.PostTime, err = time.Parse("2006-01-02", doc.Find(".time").Text())
	if err == nil {
		art.PostTime = art.PostTime.Local()
	} else {
		art.PostTime = time.Now()
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
			img.Remove()
			return
		}
		if alt := img.AttrOr("alt", ""); len(alt) == 0 {
			img.RemoveAttr("alt")
		} else {
			if strings.Contains(alt, "http://") {
				img.RemoveAttr("alt")
			}
		}
		img.RemoveAttr("data-original")
		img.RemoveAttr("data-link")
		img.RemoveAttr("srcset")
		img.RemoveAttr("sizes")
		img.RemoveAttr("title")
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
		tg := collect.ArticleTag{Name: strings.TrimSpace(k.Text()), Tag: strings.TrimSpace(tag)}
		art.Tag = append(art.Tag, tg)
		k.RemoveAttr("href")
		k.RemoveAttr("target")
		k.RemoveClass("infotextkey")
		k.AddClass(collect.TagClass[1:])
		k.SetAttr(collect.TagAttrName, tg.Name)
		k.SetAttr(collect.TagAttrValue, tg.Tag)
	})
	// 处理<a>
	doc.Find(".kg-card-markdown a").Each(func(_ int, a *goquery.Selection) {
		if _, ok := a.Attr(collect.TagAttrName); ok {
			return
		}
		href := a.AttrOr("href", "")
		if strings.Contains(href, "/tag/") {
			tg := collect.ArticleTag{Name: strings.TrimSpace(a.Text()), Tag: ""}
			if tg.Name == "" {
				a.Remove()
			} else {
				art.Tag = append(art.Tag, tg)
				a.AddClass(collect.TagClass[1:])
				a.SetAttr(collect.TagAttrName, tg.Name)
				a.SetAttr(collect.TagAttrValue, tg.Tag)
			}
		}
		a.RemoveAttr("href")
		a.RemoveAttr("title")
		a.RemoveAttr("data-group")
		a.RemoveAttr("data-id")
		a.RemoveAttr("data-index")
		aParent := a.Parent()
		if aParent.Is("figure") {
			html, _ := a.Html()
			aParent.SetHtml(html)
		}
	})
	// 处理<p>
	doc.Find(".kg-card-markdown p").Each(func(i int, p *goquery.Selection) {
		p.RemoveAttr("data-track")
	})
	art.Content, _ = doc.Find(".kg-card-markdown").Html()
	art.Content = strings.TrimSpace(art.Content)
	return nil
}
