package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/net/context/ctxhttp"
)

//分割数（４で固定。CPUコア数とかにするといいのかも）
const splitNum = 4

//RFC 7233 — HTTP/1.1: Range Requests
func main() {
	// サーバ起動
	//http.HandleFunc("/", Handler)
	//go http.ListenAndServe(":8080", nil)
	main5(os.Args[1])

}

func main5(url string) {
	//ネットワーク越しにファイルのサイズを取得
	//Content-TypeとContent-Length
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := ctxhttp.Head(ctx, http.DefaultClient, url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	if "bytes" != res.Header.Get("Accept-Ranges") {
		fmt.Fprintln(os.Stderr, "対象ファイルがrange未対応だす")
		os.Exit(1)
	}
	//分割サイズの決定
	chunkSize := res.ContentLength / splitNum
	log.Printf("Accept-Ranges:%v,ContentLength:%v,chunkSize:%v\n", res.Header.Get("Accept-Ranges"), res.ContentLength, chunkSize)

	//分割ダウンロード関数
	var wg sync.WaitGroup
	var splitDownload = func(part int, url string, rangeString string) {
		defer log.Println("gorutine終了")
		//部分ダウンロードして外部ファイルに保存
		err := partialDownload(part, url, rangeString)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		}
		wg.Done()

	}

	for i, v := range getByteRange(chunkSize, res.ContentLength, splitNum) {
		wg.Add(1)
		log.Printf("splitDownload part%v start %v\n", i+1, v)
		//goルーチンで動かす関数はforループが回りきってから動き始めるので
		//goルーチン内でAdd(1)するとWaitされない場合がある
		go splitDownload(i+1, url, v)
	}
	fmt.Println("wg.Wait()")
	//分割ダウンロードが終わるまでブロック
	wg.Wait()
	//分割保存したファイルを合体
	_, err = os.Stat(filepath.Base(url))
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	//margeFile := os.Create(filepath.Base(url))
	for i := 0; i < splitNum; i++ {
		file, err := os.Open(fmt.Sprintf("part%v", i))
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

	}
	fmt.Println("wg all done")
}

func _partialDownload(part int, url string, rangeString string) error {
	log.Printf("part:%vダウンロード中・・・", part)
	if part == 1 {
		time.Sleep(4 * time.Second)
	}

	log.Printf("part:%v,url:%v,rangeString:%v", part, url, rangeString)
	return nil
}

//分割ダウンロード
func partialDownload(part int, url string, rangeString string) error {
	//ファイル作成
	file, err := os.Create(fmt.Sprintf("part%v", part))
	if err != nil {
		return err
	}
	defer file.Close()

	//リクエスト作成
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Range",
		fmt.Sprintf("bytes=%v", rangeString))

	dump, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		return err
	}

	fmt.Printf("%s", dump)

	res, err := http.DefaultClient.Do(req)

	dumpResp, _ := httputil.DumpResponse(res, false)
	fmt.Println(string(dumpResp))

	//b, err := ioutil.ReadAll(res.Body)
	io.Copy(file, res.Body)
	//fmt.Fprintln(file, b)
	if err != nil {
		return err
	}

	return nil
}

//rangeヘッダに指定する値を算出するchunkSize int64, res.ContentLength int, splitNum int) string{
func getByteRange(chunkSize int64, contentLength int64, splitNum int) []string {
	var rangeArr = []string{}
	var from, to int64
	for i := 0; i < splitNum; i++ {
		switch i {
		case 0:
			from = 0
			to = chunkSize
		case splitNum - 1:
			from = to + 1
			to = contentLength
		default:
			from = to + 1
			to += chunkSize
		}
		rangeArr = append(rangeArr, fmt.Sprintf("%v-%v", from, to))
	}
	log.Println(rangeArr)
	return rangeArr
}

func main4() {
	//ネットワーク越しにファイルを部分取得
	url := "https://beauty.hotpepper.jp/CSP/c_common/ALL/IMG/cam_cm_327_98.jpg"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
	req.Header.Set("Range", "bytes=0-100")

	dump, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	fmt.Printf("%s", dump)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	dumpResp, _ := httputil.DumpResponse(resp, false)
	fmt.Println(string(dumpResp))
}

func main2() {
	url := "http://ascii.jp"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Range", "bytes=0-100")

	dump, _ := httputil.DumpRequestOut(req, false)
	client := new(http.Client)
	fmt.Printf("%s", dump)
	resp, _ := client.Do(req)
	dumpResp, _ := httputil.DumpResponse(resp, false)
	fmt.Println(string(dumpResp))
}

func main3() {
	fmt.Println("-----main3 start----")
	req, err := http.NewRequest("GET", "http://localhost:8080/", nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Range", "bytes=0-100")
	dump, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", dump)

	fmt.Println("-----main3 Do----")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	fmt.Println("-----main3 Do end----")
	dumpResp, _ := httputil.DumpResponse(res, false)
	fmt.Println(string(dumpResp))
	fmt.Println("-----main3 end----")
	// for {
	// 	//サイズを取得
	// 	sizeStr, err := reader.ReadBytes('\n')

	// 	if err == io.EOF {
	// 		break
	// 	}
	// 	//16進数のサイズをパース。サイズが0ならクローズ
	// 	size, err := strconv.ParseInt(
	// 		string(sizeStr[:len(sizeStr)-2]), 16, 64)
	// 	if size == 0 {
	// 		break
	// 	}
	// 	if err != nil {
	// 		fmt.Printf("err=%v", err.Error())
	// 		os.Exit(1)
	// 	}
	// 	//サイズ数分バッファを確保して読み込み
	// 	line := make([]byte, int(size))
	// 	reader.Read(line)
	// 	reader.Discard(2)
	// 	fmt.Printf(" %d betes: %s\n", size, string(line))
	// }

}

func main1() {
	//コネクション作成（RW）
	conn, err := net.Dial("tcp", "ascii.jp:80")
	if err != nil {
		panic(err)
	}
	request, err := http.NewRequest(
		"GET",
		"http://ascii.jp/",
		nil)
	if err != nil {
		panic(err)
	}

	dumpreq, _ := httputil.DumpRequestOut(request, false)
	fmt.Println(string(dumpreq))

	err = request.Write(conn)
	if err != nil {
		panic(err)
	}

	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, request)
	if err != nil {
		panic(err)
	}
	dump, err := httputil.DumpResponse(response, false)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(dump))
	if len(response.TransferEncoding) < 1 ||
		response.TransferEncoding[0] != "chunked" {
		panic("wrong transfer encoding")
	}

}
