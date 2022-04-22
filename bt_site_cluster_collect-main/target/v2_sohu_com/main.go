package v2_sohu_com

import (
	"bytes"
	"crypto/aes"
	"encoding/base64"
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"github.com/cgghui/bt_site_cluster_collect/collect"
	"github.com/cgghui/cgghui"
	"github.com/mozillazg/go-pinyin"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	Name = "v2_sohu_com"
)

func init() {
	collect.RegisterStandard(Name, func() collect.Standard {
		return &CollectGo{HomeURL: "https://v2.sohu.com/"}
	})
}

var pyArg = pinyin.NewArgs()

var Column = map[collect.Tag]string{
	collect.TagCommerce: "public-api/feed?scene=TAG&sceneId=65777&page={page}&size=20",
	collect.TagMobile:   "public-api/feed?scene=TAG&sceneId=59740&page={page}&size=20",
	collect.TagIT:       "public-api/feed?scene=CATEGORY&sceneId=911&page={page}&size=20",
	collect.TagTX:       "public-api/feed?scene=CATEGORY&sceneId=934&page={page}&size=20",
	collect.TagSmart:    "public-api/feed?scene=CATEGORY&sceneId=882&page={page}&size=20",
	collect.TagLife:     "public-api/feed?scene=CATEGORY&sceneId=913&page={page}&size=20",
	collect.TagSAB:      "public-api/feed?scene=CATEGORY&sceneId=881&page={page}&size=20",
	collect.TagScience:  "public-api/feed?scene=CATEGORY&sceneId=880&page={page}&size=20",
	collect.TagDigital:  "public-api/feed?scene=CATEGORY&sceneId=936&page={page}&size=20",
	collect.TagFashion:  "public-api/feed?scene=CATEGORY&sceneId=1045&page={page}&size=20",
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

type Article struct {
	Id          int64        `json:"id"`
	AuthorId    int64        `json:"authorId"`
	AuthorName  string       `json:"authorName"`
	ContentType string       `json:"contentType"`
	MobileTitle string       `json:"mobileTitle"`
	PublicTime  int64        `json:"publicTime"`
	Tags        []ArticleTag `json:"tags"`
}

type ArticleTag struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
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
	var ret []Article
	if err = json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		return nil, err
	}
	articles := make([]collect.Article, 0, len(ret))
	for _, r := range ret {
		if r.ContentType != "article" {
			continue
		}
		if strings.Contains(r.AuthorName, "本地消息") {
			continue
		}
		art := collect.Article{
			Title:    strings.TrimSpace(r.MobileTitle),
			Href:     strconv.FormatInt(r.Id, 10) + "_" + strconv.FormatInt(r.AuthorId, 10),
			PostTime: time.Unix(r.PublicTime, 0).Local(),
			Tag:      make([]collect.ArticleTag, 0),
		}
		for _, tg := range r.Tags {
			py := strings.Join(pinyin.LazyPinyin(tg.Name, pyArg), "")
			if py == "" {
				py = tg.Name
			}
			art.Tag = append(art.Tag, collect.ArticleTag{Name: tg.Name, Tag: py})
		}

		articles = append(articles, art)
	}

	return articles, nil
}

func (c CollectGo) HasSnapshot(art *collect.Article) bool {
	if art.Href == "" {
		return false
	}
	dir := cgghui.MD5("https://www.sohu.com/a/" + art.Href)
	return collect.PathExists("./snapshot/" + Name + "/" + string(dir[0]) + "/" + dir + ".html")
}

func (c CollectGo) ArticleDetail(art *collect.Article) error {
	var err error
	if art.Href == "" {
		return collect.ErrUndefinedArticleHref
	}
	target := "https://www.sohu.com/a/" + art.Href
	var cache *os.File
	dir := cgghui.MD5(target)
	snapshotPath := "./snapshot/" + Name + "/" + string(dir[0]) + "/"
	_ = os.MkdirAll(snapshotPath, 0666)
	snapshotPath += dir + ".html"
	if cache, err = os.Open(snapshotPath); err != nil {
		var req *http.Request
		if req, err = http.NewRequest(http.MethodGet, target, nil); err != nil {
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
	if art.LocalImages == nil {
		art.LocalImages = make([]string, 0)
	}
	word := doc.Find("#mp-editor")
	word.Find(".backsohu").Parent().Remove()
	// 处理图片
	word.Find("img").Each(func(_ int, img *goquery.Selection) {
		dataSrc := img.AttrOr("data-src", "")
		if dataSrc == "" {
			return
		}
		var imgPath string
		if imgPath, err = collect.DownloadImage(string(AesDecryptECB(dataSrc))); err != nil {
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
		img.RemoveAttr("data-src")
		img.SetAttr("src", imgPath)
		art.LocalImages = append(art.LocalImages, imgPath)
	})
	// 处理<a>
	if art.Tag == nil {
		art.Tag = make([]collect.ArticleTag, 0)
	}
	word.Find("a").Each(func(_ int, a *goquery.Selection) {
		aHTML, _ := a.Html()
		a.BeforeHtml(aHTML)
		a.Remove()
	})
	// 处理<p>
	word.Find("p").Each(func(i int, p *goquery.Selection) {
		if _, has := p.Attr("data-role"); has {
			p.Remove()
			return
		}
		if has := p.HasClass("ql-align-center"); has {
			p.SetAttr("style", "text-align: center;")
			p.RemoveClass("ql-align-center")
		}
		if has := p.HasClass("ql-align-justify"); has {
			p.SetAttr("style", "text-align: justify;")
			p.RemoveClass("ql-align-justify")
		}
	})
	if text := word.Find("p").Last().Text(); text == "举报/反馈" {
		word.Find("p").Last().Remove()
	}
	if text := word.Find("p").Last().Text(); strings.HasPrefix(text, "来源：") {
		word.Find("p").Last().Remove()
	}
	art.Content, _ = word.Html()
	art.Content = string(matchNote.ReplaceAll([]byte(art.Content), []byte{}))
	art.Content = strings.TrimSpace(art.Content)
	if len(strings.TrimSpace(word.Text())) < 900 {
		return collect.ErrArticleTooShort
	}
	return nil
}

var AesEcbKey = []byte("www.sohu.com6666")

func AesDecryptECB(encrypted string) (decrypted []byte) {
	var dst = make([]byte, base64.StdEncoding.DecodedLen(len(encrypted)))
	_, _ = base64.StdEncoding.Decode(dst, []byte(encrypted))
	cipher, _ := aes.NewCipher(AesEcbKey)
	decrypted = make([]byte, len(dst))
	last := 0
	for bs, be := 0, cipher.BlockSize(); bs < len(dst); bs, be = bs+cipher.BlockSize(), be+cipher.BlockSize() {
		if last = bytes.Index(decrypted, []byte{2}); last != -1 {
			break
		}
		if last = bytes.Index(decrypted, []byte{3}); last != -1 {
			break
		}
		if last = bytes.Index(decrypted, []byte{7}); last != -1 {
			break
		}
		if last = bytes.Index(decrypted, []byte{8}); last != -1 {
			break
		}
		cipher.Decrypt(decrypted[bs:be], dst[bs:be])
	}
	return decrypted[:last]
}

var matchNote = regexp.MustCompile(`(?U)<!--.+-->`)
