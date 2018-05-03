package iria

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/net/context/ctxhttp"
	"golang.org/x/sync/errgroup"
)

//Downloader is setting
type Downloader struct {
	URL           string //取得対象URL
	SplitNum      int    //ダウンロード分割数
	contentLength int64  //取得対象ファイルサイズ
	chunkLength   int64  //ダウンロード分割サイズ
	eg            errgroup.Group
}

//ダウンロード用一時ファイル part1~part{splitNum}
const tmpFile = "part"

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
	//CPUコア数だけダウンロードを分割する
	splitNum := runtime.NumCPU()
	//分割用一時ファイル存在チェック
	for i := 1; i <= splitNum; i++ {
		if exists(fmt.Sprintf("%v%v", tmpFile, i)) {
			return nil, fmt.Errorf("分割用一時ファイルが存在するためダウンロードを開始できません:%v%v", tmpFile, i)
		}
	}
	return &Downloader{
		URL:      args[1],
		SplitNum: splitNum,
	}, nil
}

//Execute はDownloaderメイン処理
func (d *Downloader) Execute() error {
	//ダウンロード分割サイズの決定
	if err := d.setChunkLength(); err != nil {
		return err
	}
	//gorutineで分割ダウンロード
	for i, v := range d.getByteRange() {
		part := i + 1
		rangeString := v
		log.Printf("splitDownload part%v start %v\n", i+1, v)
		//goルーチンで動かす関数はforループが回りきってから動き始めるので
		//goルーチン内でAdd(1)するとWaitされない場合がある
		d.eg.Go(func() error {
			return d.splitDownload(part, rangeString)
		})
	}

	//分割ダウンロードが終わるまでブロック
	if err := d.eg.Wait(); err != nil {
		return err
	}

	//分割ダウンロードしたファイル合体
	return d.margeChunk()
}

//ダウンロード分割サイズの決定
func (d *Downloader) setChunkLength() error {
	//ネットワーク越しにファイルのサイズを取得
	//Content-TypeとContent-Length
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := ctxhttp.Head(ctx, http.DefaultClient, d.URL)
	if err != nil {
		return err
	}
	if "bytes" != res.Header.Get("Accept-Ranges") {
		return fmt.Errorf("取得対象ファイルがrange request未対応でした:%v", d.URL)
	}
	//分割サイズの決定
	d.contentLength = res.ContentLength
	d.chunkLength = res.ContentLength / int64(d.SplitNum)
	log.Printf("Accept-Ranges:%v,ContentLength:%v,chunkSize:%v\n", res.Header.Get("Accept-Ranges"), res.ContentLength, d.chunkLength)
	return nil
}

//rangeヘッダに指定する値を算出する
//@return []string	rangeヘッダ指定値	{"0-N","N+1-M",..."M-contentLength"}
func (d *Downloader) getByteRange() []string {
	var rangeArr = []string{}
	var from, to int64
	for i := 0; i < d.SplitNum; i++ {
		switch i {
		case 0:
			from = 0
			to = d.chunkLength
		case d.SplitNum - 1:
			from = to + 1
			to = d.contentLength
		default:
			from = to + 1
			to += d.chunkLength
		}
		rangeArr = append(rangeArr, fmt.Sprintf("%v-%v", from, to))
	}
	//log.Println(rangeArr)
	return rangeArr
}

//gorutineで並列ダウンロード
func (d *Downloader) splitDownload(part int, rangeString string) error {
	//ファイル作成
	file, err := os.Create(fmt.Sprintf("part%v", part))
	if err != nil {
		return err
	}
	defer file.Close()
	//部分ダウンロードして外部ファイルに保存
	return partialRequest(d.URL, part, rangeString, file)
}

//分割ダウンロード
func partialRequest(url string, part int, rangeString string, file *os.File) error {
	//リクエスト作成
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range",
		fmt.Sprintf("bytes=%v", rangeString))

	//デバッグ用リクエストヘッダ出力
	dump, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		return err
	}
	fmt.Printf("%s", dump)

	res, err := http.DefaultClient.Do(req)

	//デバッグ用レスポンスヘッダ出力
	dumpResp, _ := httputil.DumpResponse(res, false)
	fmt.Println(string(dumpResp))

	if _, err := io.Copy(file, res.Body); err != nil {
		return err
	}
	return nil
}

//分割ダウンロードしたファイルを合体して復元する
func (d *Downloader) margeChunk() error {
	margeFile, err := os.Create(filepath.Base(d.URL))
	if err != nil {
		return err
	}
	defer margeFile.Close()

	for i := 0; i < d.SplitNum; i++ {
		file, err := os.Open(fmt.Sprintf("part%v", i+1))
		if err != nil {
			return err
		}
		//ファイルに追記
		if _, err = io.Copy(margeFile, file); err != nil {
			return err
		}
		if err = file.Close(); err != nil {
			return err
		}
		//削除
		if err = os.Remove(fmt.Sprintf("part%v", i+1)); err != nil {
			return err
		}
	}
	return nil
}

//ファイル存在チェック
func exists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}
