package anilist

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchManga_NotFoundIsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors":[{"message":"Not Found.","status":404}],"data":{"Media":null}}`))
	}))
	defer srv.Close()

	previous := graphqlURL
	graphqlURL = srv.URL
	t.Cleanup(func() { graphqlURL = previous })

	media, err := SearchManga("Missing Title")
	if err != nil {
		t.Fatal(err)
	}
	if media != nil {
		t.Fatalf("got %+v", media)
	}
}

func TestSearchManga_OtherStatusIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusBadGateway)
	}))
	defer srv.Close()

	previous := graphqlURL
	graphqlURL = srv.URL
	t.Cleanup(func() { graphqlURL = previous })

	if _, err := SearchManga("Missing Title"); err == nil {
		t.Fatal("expected error")
	}
}
