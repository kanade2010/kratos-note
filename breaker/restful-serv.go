package main

import(
	"net/http"
	"fmt"
	"log"
	"html"
	"io/ioutil"
	"sync/atomic"
)

func main() {
	var num int32 = 0

	//http.Handle("/foo", fooHandler)
	http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(atomic.AddInt32(&num, 1))
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	http.HandleFunc("/post", func(w http.ResponseWriter, r *http.Request) {
		robots, err := ioutil.ReadAll(r.Body)
		r.Body.Close()

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			fmt.Printf("POST : %s\n", robots)
			w.Write([]byte("successful"))
		}
	})

	http.HandleFunc("/put", func(w http.ResponseWriter, r *http.Request) {
		robots, err := ioutil.ReadAll(r.Body)
		r.Body.Close()

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			fmt.Printf("PUT : %s\n", robots)
			w.Write([]byte("successful"))
		}
	})

	http.HandleFunc("/del", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "do del, %q", html.EscapeString(r.URL.Path))
	})

	log.Fatal(http.ListenAndServe(":8089", nil))

}