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

const (
	dbPath = "./.db"
)

func routes() *chi.Mux {
	router := chi.NewRouter()
	router.Use(
		render.SetContentType(render.ContentTypeJSON),
		middleware.Logger,
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
	h := NewHermes(
		"{{.Protocol}}//{{.Host}}/v1/api/mailbox/{{.InboxID}}",
		"{{.Protocol}}//{{.Host}}/v1/api/mailbox/{{.InboxID}}/{{.NotificationID}}")

	router.Get("/{inboxID}", func(w http.ResponseWriter, r *http.Request) {
		inboxID := chi.URLParam(r, "inboxID")
		h.GetInbox(inboxID, w, r)
	})

	router.Post("/{inboxID}", func(w http.ResponseWriter, r *http.Request) {
		inboxID := chi.URLParam(r, "inboxID")
		h.CreateNotification(inboxID, w, r)
	})

	router.Get("/{inboxID}/{notificationID}", func(w http.ResponseWriter, r *http.Request) {
		inboxID := chi.URLParam(r, "inboxID")
		notificationID := chi.URLParam(r, "notificationID")
		h.GetNotification(inboxID, notificationID, w, r)
	})

	return router
}

func main() {
	router := routes()
	var err error
	db, err = badger.Open(badger.DefaultOptions(dbPath))
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
