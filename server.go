package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func pdServer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		var requestData map[string]string

		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil {
			http.Error(w, `{"error": "Invalid JSON data"}`, http.StatusBadRequest)
			return
		}

		fileName := fmt.Sprintf("%s.json", requestData["date"])
		file, err := os.Open(fileName)
		if err != nil {
			http.Error(w, `{"error": "Data unavailable for this date."}`, http.StatusBadRequest)
			return
		}
		defer file.Close()

		var data map[string]map[string]interface{}
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&data); err != nil {
			http.Error(w, `{"error": "Error parsing associated data."}`, http.StatusBadRequest)
			return
		}

		data2 := make(map[string]map[string]interface{})

		stock := requestData["stock"]
		pdi := requestData["pdi"]

		data2["stock_price"] = data[stock]["stock_price"].(map[string]interface{})
		pdData := data[stock]["PD"].(map[string]interface{})
		pdvData := data[stock]["PDV"].(map[string]interface{})
		data2["PD"] = pdData[pdi].(map[string]interface{})
		data2["PDV"] = pdvData[pdi].(map[string]interface{})

		json.NewEncoder(w).Encode(data2)
		return
	}
}

func StartServer() *http.Server{
	//http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./dashboard"))))
	http.HandleFunc("/pd", pdServer)
	server := &http.Server{Addr: ":8080"}

	go func() {
		fmt.Println("Server listening on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("Server error:", err)
		}
	}()

	return server
}
