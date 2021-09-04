package main

import (
	"fmt"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/truthtracer/tsdextractor"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("tsdextractor is a text extractor based on text and symbol density")
		fmt.Println("Usage:")
		fmt.Println("\ttsdextractor [url]")
		return
	}
	url := os.Args[1]
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	source, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	result, err := tsdextractor.Extract(string(source))
	if err != nil {
		log.Fatal(err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		log.Fatal(err)
	}
	var str bytes.Buffer
	_ = json.Indent(&str, data, "", "    ")
	fmt.Println("result:")
	fmt.Println(str.String())
}
