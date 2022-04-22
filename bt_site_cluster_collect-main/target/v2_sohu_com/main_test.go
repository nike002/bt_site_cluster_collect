package v2_sohu_com

import (
	"fmt"
	"github.com/cgghui/bt_site_cluster_collect/collect"
	"testing"
)

func TestTechsir(t *testing.T) {
	obj := collect.GetStandard(Name)
	list, err := obj.ArticleList(collect.TagFashion, 1)
	if err != nil {
		t.Fatalf("error:%v", err)
	}
	if len(list) == 0 {
		t.Fatal("article list == 0")
	}
	for i := range list {
		_ = obj.ArticleDetail(&list[i])
	}

	fmt.Println()
}
