package main

import (
	"encoding/json"
	"net/http"
)

type ApiResponse struct {
	Error    string      `json:"error"`
	Response interface{} `json:"response,omitempty"`
}

func writeResponse(w http.ResponseWriter, errStr string, res interface{}) {
	data, err := json.Marshal(ApiResponse{errStr, res})
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}
