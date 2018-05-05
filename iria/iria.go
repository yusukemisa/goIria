package iria

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/net/context/ctxhttp"
)

//New create Downloader
func New(args []string) (*Downloader, error) {
	if len(args) != 2 {
		return nil, errors.New("取得対象とするURLを１つ指定してください")
	}
	//取得対象ファイルと同名のファイルが既にある場合を許さない
	targetFileName := filepath.Base(args[1])
	if exists(targetFileName) {
		return nil, fmt.Errorf("取得対象のファイルが既に存在しています:%v", targetFileName)
	}
	//取得対象ファイルサイズを確認
	url := args[1]
	cl, err := getContentLength(url)
	if err != nil {
		return nil, err
	}
	splitNum := runtime.NumCPU()
	return &Downloader{
		URL:           url,
		SplitNum:      splitNum, //CPUコア数だけダウンロードを分割する
		ContentLength: cl,
		ChunkLength:   cl / int64(splitNum),
	}, nil
}

//ダウンロード分割サイズの決定
func getContentLength(url string) (int64, error) {
	//ネットワーク越しにファイルのサイズを取得
	//Content-TypeとContent-Length
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := ctxhttp.Head(ctx, http.DefaultClient, url)
	if err != nil {
		return 0, err
	}
	if "bytes" != res.Header.Get("Accept-Ranges") {
		return 0, fmt.Errorf("取得対象ファイルがrange request未対応でした:%v", url)
	}
	//分割サイズの決定
	return res.ContentLength, nil
}

//rangeヘッダに指定する値を算出する
//@return []string	rangeヘッダ指定値	{"0-N","N+1-M",..."M-contentLength"}
func getByteRange(contentLength int64, chunkLength int64, splitNum int) []string {
	var rangeArr = []string{}
	var from, to int64
	for i := 0; i < splitNum; i++ {
		switch i {
		case 0:
			from = 0
			to = chunkLength
		case splitNum - 1:
			from = to + 1
			to = contentLength
		default:
			from = to + 1
			to += chunkLength
		}
		rangeArr = append(rangeArr, fmt.Sprintf("%v-%v", from, to))
	}
	//log.Println(rangeArr)
	return rangeArr
}

//ファイル存在チェック
func exists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}
