package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/julienschmidt/httprouter"
	"github.com/quanhengzhuang/gtm-console/pkg/storage"
	"github.com/spf13/viper"
)

var (
	s storage.ConsoleStorage
)

var (
	configFile = flag.String("c", "configs/default.toml", "config file")
)

func main() {
	flag.Parse()

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.SetConfigFile(*configFile)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("read in config failed: %v", err)
	}

	db, err := gorm.Open(viper.GetString("db_storage.type"), viper.GetString("db_storage.dsn"))
	if err != nil {
		log.Fatalf("db open failed: %v", err)
	}
	db.LogMode(true)

	s = storage.NewDBConsoleStorage(db)

	router := httprouter.New()
	router.GET("/", makeHandler(transactions))
	router.GET("/transaction/:id", makeHandler(transaction))

	port := viper.GetString("http.port")
	log.Printf("http server starting. port:%v", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func makeHandler(f func(r *http.Request, ps httprouter.Params) ([]byte, error)) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		if content, err := f(r, ps); err != nil {
			w.Write([]byte(fmt.Sprintf("internal error: %v", err)))
		} else {
			w.Write(content)
		}
	}
}

var (
	templateFunc = template.FuncMap{
		"toSeconds": toSeconds,
	}
)

func transactions(r *http.Request, _ httprouter.Params) (content []byte, err error) {
	page := r.FormValue("page")
	pageNum, _ := strconv.Atoi(page)
	pageSize := 50

	transactions, err := s.GetTransactions(pageNum, pageSize)
	if err != nil {
		return nil, fmt.Errorf("get transactions failed: %v", err)
	}

	pageNext := pageNum + 1
	if len(transactions) < pageSize {
		pageNext = 0
	}

	values := map[string]interface{}{
		"transactions": transactions,
		"page":         pageNum,
		"pagePrevious": pageNum - 1,
		"pageNext":     pageNext,
	}

	return render("web/transactions.html", values)
}

func transaction(r *http.Request, ps httprouter.Params) (content []byte, err error) {
	id := ps.ByName("id")
	if id == "" {
		return nil, fmt.Errorf("id parse failed: %v", err)
	}

	transaction, err := s.GetTransaction(id)
	if err != nil {
		return nil, fmt.Errorf("get transaction failed: %v", err)
	}

	partnerResults, err := s.GetPartnerResults(transaction.ID)
	if err != nil {
		return nil, fmt.Errorf("get partner results failed: %v", err)
	}

	values := map[string]interface{}{
		"transaction":    transaction,
		"partnerResults": partnerResults,
	}

	return render("web/transaction.html", values)
}

func render(templateFile string, values interface{}) (content []byte, err error) {
	tpl := template.New("tpl")
	tpl.Funcs(sprig.FuncMap())
	tpl.Funcs(templateFunc)
	if _, err := tpl.ParseFiles("web/layout.html", templateFile); err != nil {
		return nil, fmt.Errorf("template parse failed: %v", err)
	}

	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, "layout.html", values); err != nil {
		return nil, fmt.Errorf("template execute failed: %v", err)
	}

	return buf.Bytes(), nil
}

func toSeconds(duration time.Duration) string {
	return fmt.Sprintf("%.3f", duration.Seconds())
}
