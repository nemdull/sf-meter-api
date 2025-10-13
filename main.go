package main

import (
	"log"

	"github.com/nemdull/sf-meter-api/registry"
)

func main() {
	// HTTPサーバー起動（サーバーのインスタンスを生成）
	srv := registry.NewHTTPServer(":8080")

	// サーバーが起動したことをログに出力
	log.Println("Starting server on :8080")

	// ポート8080番で待ち受け
	log.Fatal(srv.ListenAndServe())
}
