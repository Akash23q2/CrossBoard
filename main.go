// imports
package main

import (
	"fmt"
	"net/http"
)

func main() {

	//register routes

	http.HandleFunc("/", root)
	http.HandleFunc("/health", getHealth)
	http.HandleFunc("/clipboard", getClipboard)
	http.HandleFunc("/clipboard/send", sendClipboard)

	//start the server
	fmt.Println("starting server!")
	url := handleTunneling()
	generateQr(url)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Println(err)
	}

}
