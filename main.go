package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	cookieName = "_qca"
)

type Server struct {
	db *pgxpool.Pool
}

type Entry struct {
	ClientId      uuid.UUID `json:"clientId"`
	Time          time.Time `json:"time"`
	RemoteAddress string
}

func SetNewClientID(w http.ResponseWriter, r *http.Request) uuid.UUID {
	userIdentifier := uuid.New()
	log.Printf("generated new client id = %s", userIdentifier.String())
	w.Header().Add("Set-Cookie", fmt.Sprintf("%s=%s", cookieName, userIdentifier.String()))
	return userIdentifier
}

func GetClientID(w http.ResponseWriter, r *http.Request) (userIdentifier uuid.UUID) {
	cookies := r.Cookies()
	for _, c := range cookies {
		if c.Name == cookieName {
			var err error
			userIdentifier, err = uuid.Parse(c.Value)
			if err != nil {
				log.Printf("received invalid err=%s", err)
				userIdentifier = SetNewClientID(w, r)
			} else {
				return userIdentifier
			}
		}
	}

	if userIdentifier == uuid.Nil {
		userIdentifier = SetNewClientID(w, r)
	}
	return
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clientID := GetClientID(w, r)
	log.Printf("new request from %s", clientID)
	s.db.Exec(context.Background(), "INSERT INTO entries (client_id, time, remote_address) VALUES ($1, $2, $3)", clientID, time.Now(), r.RemoteAddr)
}

func main() {
	dsn := "host=postgres user=postgres password=some-password dbname=qca port=5432"
	db, err := pgxpool.Connect(context.Background(), dsn)
	if err != nil {
		panic("failed to connect database")
	}
	defer db.Close()

	db.Exec(context.Background(), "CREATE TABLE IF NOT EXISTS entries (client_id uuid, time timestamp, remote_address text)")

	s := &Server{
		db,
	}

	http.Handle("/", s)
	log.Fatal(http.ListenAndServe(":80", nil))
}
