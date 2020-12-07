package main

import (
	"log"
	"net/http"

	"github.com/dgraph-io/badger"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

var db *badger.DB

type JSONLD struct {
	Text Text `json:"@context"`
}

type Text struct {
	Id   string `json:"id"`
	Text string `json:"text"`
}

const (
	dbPath = "./postoffice"
)

func routes() *chi.Mux {
	router := chi.NewRouter()
	router.Use(
		render.SetContentType(render.ContentTypeJSON),
		middleware.Logger,
		middleware.DefaultCompress,
		middleware.RedirectSlashes,
		middleware.Recoverer,
	)

	router.Route("/v1", func(r chi.Router) {
		r.Mount("/api/mailbox/", apiRoutes())
	})
	return router
}

func apiRoutes() *chi.Mux {
	router := chi.NewRouter()
	router.Get("/{mailbox}", getLDN)
	router.Post("/{mailbox}", postLDN)
	return router
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
	mailbox := chi.URLParam(r, "mailbox")
	// err := db.View(func(txn *badger.Txn) error {
	// 	// Your code here…
	// 	return nil
	// })
	// if err != nil {
	// 	log.Fatal(err)
	// }
	jsonld := JSONLD{Text: Text{
		Id:   mailbox,
		Text: "Hello world",
	},
	}
	render.JSON(w, r, jsonld)
}

func main() {
	router := routes()
	db, err := badger.Open(badger.DefaultOptions(dbPath))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		log.Printf("%s %s\n", method, route)
		return nil
	}
	if err := chi.Walk(router, walkFunc); err != nil {
		log.Panicf("Logging err: %s\n", err.Error())
	}
	log.Fatal(http.ListenAndServe(":8080", router))
}
