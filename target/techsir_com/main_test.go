package techsir_com

import (
	"fmt"
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
	_ = obj.ArticleDetail(&list[0])
	fmt.Println(list)
}
