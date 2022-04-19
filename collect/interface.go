package collect

var standardMap = make(map[string]func() Standard)

func RegisterStandard(name string, new func() Standard) {
	standardMap[name] = new
}

func GetStandard(name string) Standard {
	if _, ok := standardMap[name]; ok {
		return standardMap[name]()
	}
	return nil
}

type Standard interface {

	// ArticleList 获取文章列表
	// tag 标签，如：TagCommerce，如果 tag 未定义则应返回 ErrUndefinedTag
	// page 页码
	ArticleList(Tag, int) ([]Article, error)

	// ArticleDetail 获取文章的详细内容
	// 如果 art.Href 为空， 则应返回 ErrUndefinedArticleHref
	ArticleDetail(*Article, *Site) error
}
