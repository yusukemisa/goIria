package iria_test

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/yusukemisa/goIria/iria"
)

type TestCase struct {
	name string      //ケース名
	in   []string    //os.Args[]
	out  interface{} //合格条件
}

var splitNum = runtime.NumCPU()

/*
	iria.New
	正常系ケース定義
*/
var nomalCases = []TestCase{
	{
		name: "正常系_有効URL",
		in:   []string{"goIria", "http://test.com"},
		out: &iria.Downloader{
			URL:      "http://test.com",
			SplitNum: splitNum,
		},
	},
}

/*
	Test Suite Run
*/
func TestAll(t *testing.T) {
	t.Run("New正常系", func(t *testing.T) {
		for _, target := range nomalCases {
			testNewNormal(t, target)
		}
	})
}

//正常系テストコード
func testNewNormal(t *testing.T, target TestCase) {
	t.Helper()
	actual, err := iria.New(target.in)
	if err != nil {
		t.Errorf("err expected nil: %v", err.Error())
	}
	if actual == nil {
		t.Error("New expected Nonnil")
	}
	//構造体の中身ごと一致するか比較
	if !reflect.DeepEqual(actual, target.out) {
		t.Errorf("case:%v => %q, want %v ,actual %v", target.name, target.in, target.out, actual)
	}
}
