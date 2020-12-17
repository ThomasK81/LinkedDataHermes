package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/badger"
	uuid "github.com/google/uuid"
	"net/http"
	"text/template"
)

type Hermes struct {
	inboxScheme        *template.Template
	notificationScheme *template.Template
}

type Scheme struct {
	Protocol       string
	Host           string
	InboxID        string
	NotificationID string
}

type Notification struct {
	ID      string `json:"id"`
	Actor   string `json:"actor"`
	Object  string `json:"object"`
	Target  string `json:"target"`
	Updated string `json:"updated"`
}

type LDNInbox struct {
	Context  string   `json:"@context"`
	ID       string   `json:"@id"`
	Contains []string `json:"contains"`
}

type LDNotification struct {
	Context string `json:"@context"`
	ID      string `json:"@id"`
	Type    string `json:"@type"`
	Actor   string `json:"actor"`
	Object  string `json:"object"`
	Target  string `json:"target"`
	Updated string `json:"updated"`
}

const maxResponseSize = 128

func getInbox(inboxID string) ([]Notification, error) {
	notifications := make([]Notification, 0, maxResponseSize)
	err := db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := []byte(fmt.Sprintf("%s/", inboxID))
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				var notification Notification
				err := json.Unmarshal(v, &notification)
				if err != nil {
					return err
				}
				notifications = append(notifications, notification)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return notifications, nil
}

func getNotification(inboxID string, notificationID string) (Notification, error) {
	var notification Notification
	err := db.View(func(txn *badger.Txn) error {
		id := fmt.Sprintf("%s/%s", inboxID, notificationID)
		item, err := txn.Get([]byte(id))
		if err != nil {
			return err
		}

		err = item.Value(func(v []byte) error {
			return json.Unmarshal(v, &notification)
		})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return Notification{}, nil
	}

	return notification, nil
}

func createNotification(inboxID string, notification *Notification) error {
	notification.ID = uuid.New().String()

	n, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	err = db.Update(func(txn *badger.Txn) error {
		id := fmt.Sprintf("%s/%s", inboxID, notification.ID)
		e := badger.NewEntry([]byte(id), n)
		err := txn.SetEntry(e)
		return err
	})
	if err != nil {
		return err
	}
	return nil
}

type stringTransform = func(string) string

func mapNotificationsID(ns []Notification, transform stringTransform) []string {
	ids := make([]string, len(ns))
	for i, n := range ns {
		ids[i] = transform(n.ID)
	}
	return ids
}

func makeLDNInbox(inboxID string, makeID stringTransform, ns []Notification) LDNInbox {
	return LDNInbox{
		Context:  "http://www.w3.org/ns/ldp",
		ID:       inboxID,
		Contains: mapNotificationsID(ns, makeID),
	}
}

func makeLDNotification(makeID stringTransform, n Notification) LDNotification {
	return LDNotification{
		Context: "https://www.w3.org/ns/activitystreams",
		ID:      makeID(n.ID),
		Type:    "Announce",
		Actor:   n.Actor,
		Object:  n.Object,
		Target:  n.Target,
		Updated: n.Updated,
	}
}

func NewHermes(inboxScheme string, notificationScheme string) *Hermes {
	h := Hermes{
		inboxScheme:        template.Must(template.New("inboxScheme").Parse(inboxScheme)),
		notificationScheme: template.Must(template.New("notificationScheme").Parse(notificationScheme)),
	}
	return &h
}

func (h *Hermes) makeInboxID(inboxID string, r *http.Request) (string, error) {
	buf := new(bytes.Buffer)
	s := Scheme{
		// TODO: determine protocol (http/https)
		Protocol: "http",
		Host:     r.Host,
		InboxID:  inboxID,
	}
	err := h.inboxScheme.Execute(buf, s)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func mustID(id string, err error) string {
	if err != nil {
		panic(err)
	}
	return id
}

func (h *Hermes) makeNotificationID(inboxID string, notificationID string, r *http.Request) (string, error) {
	buf := new(bytes.Buffer)
	s := Scheme{
		// TODO: determine protocol (http/https)
		Protocol:       "http:",
		Host:           r.Host,
		InboxID:        inboxID,
		NotificationID: notificationID,
	}
	err := h.notificationScheme.Execute(buf, s)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (h *Hermes) CreateNotification(inboxID string, w http.ResponseWriter, r *http.Request) {
	var n Notification

	err := json.NewDecoder(r.Body).Decode(&n)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = createNotification(inboxID, &n)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/ld+json")
	w.Header().Set("location", mustID(h.makeNotificationID(inboxID, n.ID, r)))
	w.WriteHeader(201)
}

func (h *Hermes) GetInbox(inboxID string, w http.ResponseWriter, r *http.Request) {
	notifications, err := getInbox(inboxID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := mustID(h.makeInboxID(inboxID, r))
	inbox := makeLDNInbox(id, func(notificationID string) string {
		return mustID(h.makeNotificationID(inboxID, notificationID, r))
	}, notifications)
	w.Header().Set("content-type", "application/ld+json")
	json.NewEncoder(w).Encode(inbox)
}

func (h *Hermes) GetNotification(inboxID string, notificationID string, w http.ResponseWriter, r *http.Request) {
	notification, err := getNotification(inboxID, notificationID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	n := makeLDNotification(func(notificationID string) string {
		return mustID(h.makeNotificationID(inboxID, notificationID, r))
	}, notification)
	w.Header().Set("content-type", "application/ld+json")
	json.NewEncoder(w).Encode(n)
}
