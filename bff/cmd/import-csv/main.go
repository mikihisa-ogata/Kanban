// import-csv は CSV 保存時代のデータ（todos.csv・epics.csv・spaces.csv）を MySQL に取り込む。
// CSV はカレントディレクトリ（-dir で変更可）から読み、変更しない。取り込み先のテーブルが空でなければ何もしない。
//
//	cd bff && go run ./cmd/import-csv
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"todo-api/internal/database"
	"todo-api/internal/repository"
)

func main() {
	dir := flag.String("dir", ".", "todos.csv・epics.csv・spaces.csv があるディレクトリ")
	flag.Parse()

	db, err := database.Open(database.DSNFromEnv(), 30*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		log.Fatal(err)
	}

	result, err := repository.ImportCSV(db, *dir)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("imported: todos=%d epics=%d spaces=%d\n", result.Todos, result.Epics, result.Spaces)
}
