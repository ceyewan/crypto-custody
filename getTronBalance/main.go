package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	apiKey := "15ccd21a-38dd-4abe-aa34-73e011285ff3"
	endpoint := "https://apilist.tronscanapi.com/api/accountv2?address=TKv4CVZsqeVXkoRmP8mfLYawsH785uERKz"

	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("TRON-PRO-API-KEY", apiKey)

	client := http.Client{}
	resp, _ := client.Do(req)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Println(string(body))
}
