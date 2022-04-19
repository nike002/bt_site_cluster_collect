package collect

import "sync"

var standardMap = make(map[string]func() Standard)
var smm = &sync.Mutex{}

func RegisterStandard(name string, new func() Standard) {
	smm.Lock()
	defer smm.Unlock()
	standardMap[name] = new
}

func GetStandard(name string) Standard {
	smm.Lock()
	defer smm.Unlock()
	if _, ok := standardMap[name]; ok {
		return standardMap[name]()
	}
	return nil
}

func GetStandardName() []string {
	smm.Lock()
	defer smm.Unlock()
	r := make([]string, 0)
	for name := range standardMap {
		r = append(r, name)
	}
	return r
}

type Standard interface {

	// GetTag 获取标签列表
	GetTag() []Tag

	// ArticleList 获取文章列表
	// tag 标签，如：TagCommerce，如果 tag 未定义则应返回 ErrUndefinedTag
	// page 页码
	ArticleList(Tag, int) ([]Article, error)

	// ArticleDetail 获取文章的详细内容
	// 如果 art.Href 为空， 则应返回 ErrUndefinedArticleHref
	ArticleDetail(*Article, *Site) error
}
