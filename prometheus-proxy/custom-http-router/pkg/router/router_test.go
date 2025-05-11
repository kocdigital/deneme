package router

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestRouter(t *testing.T) {
    r := NewRouter()

    r.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("Test successful"))
    })

    req, err := http.NewRequest("GET", "/test", nil)
    if err != nil {
        t.Fatal(err)
    }

    rr := httptest.NewRecorder()
    r.ServeHTTP(rr, req)

    if status := rr.Code; status != http.StatusOK {
        t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
    }

    expected := "Test successful"
    if rr.Body.String() != expected {
        t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
    }
}