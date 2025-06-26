package main

import (
	"GeeCache/geecache"
	"fmt"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	geecache.NewGroup("score", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("search key:", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("key %s not exist", key)
		},
	))

	host := "localhost:8001"
	peers := geecache.NewHTTPPool(host)
	log.Println("geecache is running at", host)
	log.Fatal(http.ListenAndServe(host, peers))

}
