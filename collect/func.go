package collect

import (
	"errors"
	"github.com/cgghui/bt_site_cluster/bt"
	"github.com/cgghui/cgghui"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

var ErrNotFile = errors.New("not file")
var ErrNotScheme = errors.New("not scheme")
var ErrUndefinedSite = errors.New("undefined site")

const ImgRootPath = "./upload_temp"
const UploadTimeout = 10 * time.Minute

// DownloadImage 下载图片
func DownloadImage(imgURL string) (string, error) {
	imgURL = strings.Trim(imgURL, " ")
	if strings.HasPrefix(imgURL, "//") {
		imgURL = "http:" + imgURL
	}
	var err error
	var link *url.URL
	if link, err = url.Parse(imgURL); err != nil {
		return "", err
	}
	if !strings.Contains(link.Scheme, "http") {
		return "", ErrNotScheme
	}
	//
	storePath := ImgRootPath + link.Path
	//
	if strings.Contains(link.Host, ".aliyuncs.com") {
		if strings.Contains(imgURL, "?x-oss-process") {
			imgURL = strings.SplitN(imgURL, "?x-oss-process", 2)[0]
		}
	}
	if link.Host == "upload-images.jianshu.io" {
		if strings.Contains(imgURL, "?imageMogr2") {
			imgURL = strings.SplitN(imgURL, "?imageMogr2", 2)[0]
		}
	}
	if strings.Contains(link.Host, ".toutiaoimg.com") {
		storePath = strings.SplitN(storePath, "~", 2)[0] + ".jpg"
	}
	if strings.Contains(link.Host, ".byteimg.com") {
		storePath = strings.ReplaceAll(storePath, "~", "_") + ".jpg"
		storePath = strings.ReplaceAll(storePath, ":", "_")
	}
	if strings.Contains(link.Host, ".ws.126.net") {
		q := link.Query()
		if q.Has("type") {
			storePath = ImgRootPath + "/ws126net/" + cgghui.MD5(imgURL) + "." + q.Get("type")
		}
	}
	if strings.Contains(link.Host, "inews.gtimg.com") {
		storePath = ImgRootPath + "/inews_gtimg_com/" + cgghui.MD5(imgURL) + ".jpg"
	}
	if strings.Contains(link.Host, ".qpic.cn") {
		q := link.Query()
		if q.Has("wx_fmt") {
			storePath = ImgRootPath + "/qpic_cn/" + cgghui.MD5(imgURL) + "." + q.Get("wx_fmt")
		} else {
			storePath = ImgRootPath + "/qpic_cn/" + cgghui.MD5(imgURL) + ".jpg"
		}
	}
	if strings.Contains(link.Host, ".meipian.me") {
		storePath = strings.SplitN(storePath, "-mobile", 2)[0]
	}
	if storePath == ImgRootPath+"/" {
		return "", ErrNotFile
	}
	var req *http.Request
	req, err = http.NewRequest(http.MethodGet, imgURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.102 Safari/537.36")
	var resp *http.Response
	if resp, err = http.DefaultClient.Do(req); err != nil {
		return "", err
	}
	if err = os.MkdirAll("./"+path.Dir(storePath), 0755); err != nil {
		return "", err
	}
	var save *os.File
	if save, err = os.Create(storePath); err != nil {
		return "", err
	}
	if _, err = io.Copy(save, resp.Body); err != nil {
		return "", err
	}
	return link.Path, nil
}

type Site struct {
	bt.Site
	Session *bt.Session
}

// GetSite 获取站点
func GetSite(id uint, s *bt.Session) (*Site, error) {
	siteData, err := s.GetSiteListWithTimeout(6 * time.Second)
	if err != nil {
		return nil, err
	}
	for _, site := range siteData.Data {
		if site.ID == id {
			return &Site{Site: site, Session: s}, nil
		}
	}
	return nil, ErrUndefinedSite
}

// UploadImage 往宝塔上传文件
func UploadImage(site *Site, imgPath string) {
	imgRootPath := ImgRootPath + imgPath
	var fp *os.File
	var err error
	if fp, err = os.Open(imgRootPath); err != nil {
		return
	}
	serverPath := path.Dir(site.Path + imgPath)
	if err = site.Session.UploadWithTimeout(UploadTimeout, imgRootPath, serverPath, fp, true); err != nil {
		log.Printf("图片上传失败，请手动完成，%s。Error: %v", imgRootPath, err)
		return
	}
}
