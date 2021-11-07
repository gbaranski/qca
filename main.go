package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/cors"
)

const (
	cookieName = "_qca"
)

type Server struct {
	db *pgxpool.Pool
}

type Entry struct {
	ClientID uuid.UUID
	Time     time.Time
	Host     string
}

func GetHost(r *http.Request) (host string) {
	host = r.Header.Get("CF-Connecting-IP")
	if host == "" {
		var err error
		host, _, err = net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			panic(err)
		}
	}

	return
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
	if r.Method == "POST" {
		clientID := GetClientID(w, r)
		host := GetHost(r)
		log.Printf("new request from %s host = %s", clientID, host)
		_, err := s.db.Exec(context.Background(), "INSERT INTO entries (client_id, time, host) VALUES ($1, $2, $3)", clientID, time.Now(), host)
		if err != nil {
			panic(err)
		}
		w.WriteHeader(http.StatusOK)
	} else if r.Method == "GET" {
		var count int64
		row := s.db.QueryRow(context.Background(), "SELECT COUNT(distinct client_id) FROM entries")
		row.Scan(&count)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("%d", count)))
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func main() {
	dsn := "host=postgres user=root password=some-password dbname=qca port=5432"
	db, err := pgxpool.Connect(context.Background(), dsn)
	if err != nil {
		panic("failed to connect database")
	}
	defer db.Close()

	db.Exec(context.Background(), "CREATE TABLE IF NOT EXISTS entries (client_id uuid, time timestamp, host text)")

	s := &Server{
		db,
	}

	mux := http.NewServeMux()
	mux.Handle("/", s)
	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"quizizz.com"},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).Handler(mux)
	log.Fatal(http.ListenAndServe(":80", handler))
}
