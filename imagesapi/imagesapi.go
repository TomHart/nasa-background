package imagesapi

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
)

type NasaImagesResponse struct {
	Collection struct {
		Items []struct {
			Links []struct {
				Href   string `json:"href"`
				Rel    string `json:"rel"`
				Render string `json:"render"`
				Width  int    `json:"width,omitempty"`
				Height int    `json:"height,omitempty"`
				Size   int    `json:"size,omitempty"`
			} `json:"links"`
			Data []struct {
				Keywords []string `json:"keywords"`
			} `json:"data"`
		} `json:"items"`
	} `json:"collection"`
}

// GetRandomNasaImageURL fetches a random NASA image URL from the Images API.
func GetRandomNasaImageURL(_ string) (string, error) {
	url := "https://images-api.nasa.gov/search?media_type=image&keywords=artemis 2,SLS,Orion"
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	if resp.StatusCode == 403 {
		err := resp.Body.Close()
		if err != nil {
			log.Printf("error closing response body: %v", err)
		}
		return GetRandomNasaImageURL("")
	}
	defer resp.Body.Close()

	var data NasaImagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	items := data.Collection.Items
	if len(items) == 0 {
		return "", fmt.Errorf("no images found")
	}
	chosen := items[rand.Intn(len(items))]
	var largest string
	maxSize := -1
	for _, link := range chosen.Links {
		if strings.HasSuffix(link.Href, ".jpg") && link.Size > maxSize {
			largest = link.Href
			maxSize = link.Size
		}
	}
	if largest == "" {
		return "", fmt.Errorf("no JPG image found")
	}

	fmt.Printf("Keywords: '%s'\n", strings.Join(chosen.Data[0].Keywords, "', '"))
	return largest, nil
}
