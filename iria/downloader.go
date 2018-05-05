package iria

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"

	"golang.org/x/sync/errgroup"
)

//ParallelDownloader is interface
type ParallelDownloader interface {
	Execute() error
	SplitDownload(part int, rangeString string) error
	MargeChunk() error
}

//Downloader implemants ParallelDownloader
type Downloader struct {
	URL           string //取得対象URL
	SplitNum      int    //ダウンロード分割数
	ContentLength int64  //取得対象ファイルサイズ
	ChunkLength   int64  //ダウンロード分割サイズ
	//Eg            errgroup.Group
}

//ダウンロード用一時ファイル part1~part{splitNum}
const tmpFile = "part"

//Execute はDownloaderメイン処理
func (d *Downloader) Execute() error {
	//分割用一時ファイル存在チェック
	for i := 1; i <= d.SplitNum; i++ {
		if exists(fmt.Sprintf("%v%v", tmpFile, i)) {
			return fmt.Errorf("分割用一時ファイルが存在するためダウンロードを開始できません:%v%v", tmpFile, i)
		}
	}
	eg, _ := errgroup.WithContext(context.Background())
	//gorutineで分割ダウンロード
	for i, v := range getByteRange(d.ContentLength, d.ChunkLength, d.SplitNum) {
		part := i + 1
		rangeString := v
		log.Printf("splitDownload part%v start %v\n", i+1, v)
		//goルーチンで動かす関数や処理はforループが回りきってから動き始める(引数も回りきった後の状態)ので
		//goルーチン内でAdd(1)するとWaitされない場合がある
		eg.Go(func() error {
			return d.SplitDownload(part, rangeString)
		})
	}
	//分割ダウンロードが終わるまでブロック
	if err := eg.Wait(); err != nil {
		return err
	}
	//分割ダウンロードしたファイル合体
	margeFile, err := os.Create(filepath.Base(d.URL))
	if err != nil {
		return err
	}
	defer margeFile.Close()

	return d.MargeChunk(margeFile)
}

//SplitDownload gorutineで並列ダウンロード
func (d *Downloader) SplitDownload(part int, rangeString string) error {
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
func partialRequest(url string, part int, rangeString string, w io.Writer) error {
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
	if err != nil {
		log.Printf("http.DefaultClient.Do(req) err:%v", err.Error())
		return err
	}
	log.Println("Do done")
	//デバッグ用レスポンスヘッダ出力
	dumpResp, _ := httputil.DumpResponse(res, false)
	fmt.Println(string(dumpResp))

	if _, err := io.Copy(w, res.Body); err != nil {
		return err
	}
	log.Printf("partialRequest %v done", part)
	return nil
}

//MargeChunk 分割ダウンロードしたファイルを合体して復元する
func (d *Downloader) MargeChunk(w io.Writer) error {
	for i := 0; i < d.SplitNum; i++ {
		file, err := os.Open(fmt.Sprintf("%v%v", tmpFile, i+1))
		if err != nil {
			return err
		}
		//ファイルに追記
		if _, err = io.Copy(w, file); err != nil {
			return err
		}
		if err = file.Close(); err != nil {
			return err
		}
		//削除
		if err = os.Remove(fmt.Sprintf("%v%v", tmpFile, i+1)); err != nil {
			return err
		}
	}
	return nil
}
