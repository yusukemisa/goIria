## goIria
Goによる分割ダウンロード実装

## Features
- [ ] Rangeアクセスを用いる
- [ ] いくつかのゴルーチンでダウンロードしてマージする
- [ ] エラー処理を工夫する
- [ ] golang.org/x/sync/errgourpパッケージなどを使ってみる
- [ ] キャンセルが発生した場合の実装を行う

## How to use
```
$ go get github.com/yusukemisa/goIria

$ go install github.com/yusukemisa/goIria

$ goIria https://dl.google.com/go/go1.10.1.src.tar.gz
```

## 分割ダウンロード方針
- [x] Headリクエストでファイルサイズを調べる
- [x] 取得ファイルがrangeに対応してない場合は終了
- [x] ひとまず固定で４分割でrangeヘッダを付与するときに指定するサイズを計算する
- [ ] リクエストヘッダにrangeを付加してgoルーチンでリクエスト→取得した塊をファイルサイズのチャネルに突っ込む
- [ ] チャネルがいっぱいになったら中身をファイルにして出力する




### curlでやる場合
```
$ curl -I -r 0-50 https://beauty.hotpepper.jp/CSP/c_common/ALL/IMG/cam_cm_327_98.jpg
HTTP/1.1 206 Partial Content
Date: Sun, 29 Apr 2018 08:33:45 GMT
Server: Apache
Set-Cookie: GalileoCookie=WuWDaawaLscAAGyE-x8AAADl; path=/; expires=Thu, 26-Apr-29 08:33:45 GMT
Last-Modified: Fri, 20 Apr 2018 02:26:42 GMT
ETag: "d1a9074-13eb0-56a3e6b2a3c80"
Accept-Ranges: bytes
Content-Length: 51
P3P: CP="NON DSP COR CURa ADMa DEVa TAIa PSDo OUR BUS UNI COM NAV STA"
Content-Range: bytes 0-50/81584
Content-Type: image/jpeg
```
