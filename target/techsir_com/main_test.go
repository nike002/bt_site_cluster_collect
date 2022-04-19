package techsir_com

import (
	"context"
	"fmt"
	"github.com/cgghui/bt_site_cluster/bt"
	"github.com/cgghui/bt_site_cluster_collect/collect"
	"testing"
)

func TestTechsir(t *testing.T) {
	obj := collect.GetStandard(Name)
	list, err := obj.ArticleList(collect.TagCommerce, 1)
	if err != nil {
		t.Fatalf("error:%v", err)
	}
	if len(list) == 0 {
		t.Fatal("article list == 0")
	}
	opt := bt.Option{
		Link:     "http://208.87.200.95:8888/",
		Username: "xdzdki0a",
		Password: "81cea3d6",
		Code:     "e44f7e5d",
	}
	bts, err := opt.Login(context.Background())
	if err != nil {
		t.Fatalf("error:%v", err)
	}

	site, err := collect.GetSite(423, bts)
	_ = obj.ArticleDetail(&list[0], site)
	fmt.Println(list)
}
