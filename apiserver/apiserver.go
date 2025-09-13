package apiserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func Test(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		return
	}

	symbolsList := 1

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(symbolsList)
}

func Run() {
	var port = "8888"

	r := mux.NewRouter()

	r.HandleFunc("/test", Test).Methods(http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodOptions)
	r.Use(mux.CORSMethodMiddleware(r))

	fmt.Println("Running server on port:", port, "...")
	http.ListenAndServe(":"+port, r)
}
