package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const viaCepURL = "https://viacep.com.br/ws/"
const weatherAPIURL = "https://api.weatherapi.com/v1/current.json?key=87022f0c0e0d4335a1d182957242207&q="

func getCity(cep string) (string, error) {
	resp, err := http.Get(viaCepURL + cep + "/json/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("invalid zipcode")
	}

	var result struct {
		Localidade string `json:"localidade"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Localidade, nil
}

func getTemperature(city string) (float64, error) {
	resp, err := http.Get(weatherAPIURL + city)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("can not find zipcode")
	}

	var result struct {
		Current struct {
			TempC float64 `json:"temp_c"`
		} `json:"current"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result.Current.TempC, nil
}

func temperatureHandler(w http.ResponseWriter, r *http.Request) {
	cep := r.URL.Path[len("/cep/"):]

	city, err := getCity(cep)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	tempC, err := getTemperature(city)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	tempF := tempC*1.8 + 32
	tempK := tempC + 273.15

	response := map[string]interface{}{
		"city":   city,
		"temp_C": tempC,
		"temp_F": tempF,
		"temp_K": tempK,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	shutdown := initTracer()
	defer shutdown()

	http.HandleFunc("/cep/", temperatureHandler)
	http.ListenAndServe(":8081", nil)
}
