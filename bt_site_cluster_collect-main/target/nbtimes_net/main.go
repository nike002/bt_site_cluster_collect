package nbtimes_net

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
	Name = "nbtimes_net"
)

func init() {
	collect.RegisterStandard(Name, func() collect.Standard {
		return &CollectGo{HomeURL: "https://www.nbtimes.net/"}
	})
}

var Column = map[collect.Tag]string{
	collect.TagCommerce: "page/{page}?s=电商",
	collect.TagMobile:   "page/{page}?s=手机",
	collect.TagCar:      "page/{page}?s=汽车",
	collect.TagSmart:    "page/{page}?s=智能",
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
	target = strings.Replace(target, "{page}", strconv.Itoa(page), 1)
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
	doc.Find(".post-loop-default li").Each(func(_ int, li *goquery.Selection) {
		if !li.HasClass("item") {
			return
		}
		href := li.Find(".item-title a")
		articles = append(articles, collect.Article{
			Title: strings.TrimSpace(href.Text()),
			Href:  href.AttrOr("href", ""),
		})
	})
	return articles, nil
}

func (c CollectGo) HasSnapshot(art *collect.Article) bool {
	if art.Href == "" {
		return false
	}
	dir := cgghui.MD5(art.Href)
	return collect.PathExists("./snapshot/" + Name + "/" + string(dir[0]) + "/" + dir + ".html")
}

func (c CollectGo) ArticleDetail(art *collect.Article) error {
	var err error
	if art.Href == "" {
		return collect.ErrUndefinedArticleHref
	}
	var cache *os.File
	dir := cgghui.MD5(art.Href)
	snapshotPath := "./snapshot/" + Name + "/" + string(dir[0]) + "/"
	_ = os.MkdirAll(snapshotPath, 0666)
	snapshotPath += dir + ".html"
	if cache, err = os.Open(snapshotPath); err != nil {
		var req *http.Request
		if req, err = http.NewRequest(http.MethodGet, art.Href, nil); err != nil {
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
	art.Title = doc.Find(`meta[property="og:title"]`).AttrOr("content", "")
	art.Title = strings.TrimSpace(art.Title)
	art.PostTime, err = time.Parse(time.RFC3339, doc.Find(".entry-date").AttrOr("datetime", ""))
	if err == nil {
		art.PostTime = art.PostTime.Local()
	} else {
		art.PostTime = time.Now()
	}
	if art.LocalImages == nil {
		art.LocalImages = make([]string, 0)
	}
	word := doc.Find(".entry-content")
	//
	word.Find("div").Last().Remove()
	word.Find("p").Last().Remove()
	// 处理图片
	word.Find(".pgc-img").Each(func(_ int, div *goquery.Selection) {
		img := div.Find("img")
		src := img.AttrOr("src", "")
		if src == "" {
			return
		}
		var imgPath string
		if imgPath, err = collect.DownloadImage(src); err != nil {
			div.Remove()
			return
		}
		if alt := img.AttrOr("alt", ""); len(alt) == 0 {
			img.RemoveAttr("alt")
		} else {
			if strings.Contains(alt, "http://") {
				img.RemoveAttr("alt")
			}
		}
		img.RemoveAttr("data-ic")
		img.RemoveAttr("data-ic-uri")
		img.SetAttr("src", imgPath)
		imgHTML, _ := div.Html()
		div.BeforeHtml(imgHTML)
		div.Remove()
		art.LocalImages = append(art.LocalImages, imgPath)
	})
	// 处理<a>
	if art.Tag == nil {
		art.Tag = make([]collect.ArticleTag, 0)
	}
	word.Find("a").Each(func(_ int, a *goquery.Selection) {
		// 标签
		if span := a.Parent(); span.HasClass("wpcom_tag_link") {
			tag := a.AttrOr("href", "")
			if strings.Contains(tag, "/tag/") {
				tag = strings.SplitN(tag, "/tag/", 2)[1]
				tag = strings.TrimRight(tag, "/")
			} else {
				tag = ""
			}
			tg := collect.ArticleTag{Name: strings.TrimSpace(a.Text()), Tag: strings.TrimSpace(tag)}
			art.Tag = append(art.Tag, tg)
			a.RemoveAttr("href")
			a.RemoveAttr("target")
			a.AddClass(collect.TagClass[1:])
			a.SetAttr(collect.TagAttrName, tg.Name)
			a.SetAttr(collect.TagAttrValue, tg.Tag)
			tagHTML, _ := span.Html()
			span.BeforeHtml(tagHTML)
			span.Remove()
		} else {
			aHTML, _ := a.Html()
			a.BeforeHtml(aHTML)
			a.Remove()
		}
	})
	// 处理<p>
	word.Find("p").Each(func(i int, p *goquery.Selection) {
		p.RemoveAttr("data-track")
	})
	art.Content, _ = word.Html()
	art.Content = strings.ReplaceAll(art.Content, "【蓝科技综述】", "")
	art.Content = strings.ReplaceAll(art.Content, "【蓝科技观察】", "")
	art.Content = strings.TrimSpace(art.Content)
	return nil
}
