package main

import (
	"log"
	"net/http"

	"github.com/dgraph-io/badger"
	"github.com/go-chi"
	"github.com/go-chi/middleware"
	"github.com/go-chi/render"
)

var db *badger.DB

const (
	dbPath = "./tmp/postoffice"
)

func Routes() *chi.Mux {
	router := chi.NewRouter()
	router.Use(
		render.SetContentType(render.ContentTypeJSON),
		middleware.Logger,
		middleware.DefaultCompress,
		middleware.RedirectSlashes,
		middleware.Recoverer,
	)

	router.Route("v1", func(r chi.Router)) {
		r.Mount()
	}
}

func postLDN(w http.ResponseWriter, r *http.Request) {
	err := db.Update(func(txn *badger.Txn) error {
		// Your code here…
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

func getLDN(w http.ResponseWriter, r *http.Request) {
	err := db.View(func(txn *badger.Txn) error {
		// Your code here…
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	router := Routes()
	db, err := badger.Open(badger.DefaultOptions(dbPath))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Fatal(http.ListenAndServe(":8080", router))
}
