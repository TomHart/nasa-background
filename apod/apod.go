package apod

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ApodImage struct {
	Image     string `json:"hdurl"`
	Date      string `json:"date"`
	MediaType string `json:"media_type"`
}

// GetApodImageURL fetches the APOD image URL using the provided API key.
func GetApodImageURL(apiKey string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	url := fmt.Sprintf("https://api.nasa.gov/planetary/apod?api_key=%s", apiKey)
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data ApodImage
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	return data.Image, nil
}
