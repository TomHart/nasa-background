package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const keychainService = "NasaBG_APIKey"
const keychainAccount = "nasa_api_key"

// === APOD

type ApodImage struct {
	Image     string `json:"hdurl"`
	Date      string `json:"date"`
	MediaType string `json:"media_type"`
}

func getApodImageURL(apiKey string) (string, error) {

	url := fmt.Sprintf("https://api.nasa.gov/planetary/apod?api_key=%s", apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer func() {
		cerr := resp.Body.Close()
		if cerr != nil {
			log.Printf("error closing response body: %v", cerr)
		}
	}()

	var data ApodImage
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	return data.Image, nil
}

// === NASA Images API ===
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
		} `json:"items"`
	} `json:"collection"`
}

func getRandomNasaImageURL(_ string) (string, error) {
	url := "https://images-api.nasa.gov/search?q=shuttle&media_type=image&keywords=launch"
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer func() {
		cerr := resp.Body.Close()
		if cerr != nil {
			log.Printf("error closing response body: %v", cerr)
		}
	}()

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
	return largest, nil
}

// ==== Shared Utils ====

func downloadImage(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		cerr := resp.Body.Close()
		if cerr != nil {
			log.Printf("error closing response body: %v", cerr)
		}
	}()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if cerr != nil {
			log.Printf("error closing file: %v", cerr)
		}
	}()

	_, err = io.Copy(out, resp.Body)
	return err
}

func setWallpaper(imagePath string) error {
	fmt.Printf("OS: %s\n", runtime.GOOS)
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(`tell application "System Events"
	set picture of every desktop to "%s"
end tell`, imagePath)
		fmt.Println(script)
		return exec.Command("osascript", "-e", script).Run()

	case "windows":
		//absPath, err := filepath.Abs(imagePath)
		//if err != nil {
		//	return err
		//}
		//
		//// Convert to UTF-16
		//pathUTF16, err := windows.UTF16PtrFromString(absPath)
		//if err != nil {
		//	return err
		//}
		//
		//const SPI_SETDESKWALLPAPER = 0x0014
		//const SPIF_UPDATEINIFILE = 0x01
		//const SPIF_SENDCHANGE = 0x02
		//
		//// Call Windows API
		//ret, _, err := windows.NewLazySystemDLL("user32.dll").
		//	NewProc("SystemParametersInfoW").
		//	Call(uintptr(SPI_SETDESKWALLPAPER), 0, uintptr(unsafe.Pointer(pathUTF16)), uintptr(SPIF_UPDATEINIFILE|SPIF_SENDCHANGE))
		//
		//if ret == 0 {
		//	return fmt.Errorf("failed to set wallpaper: %v", err)
		//}
		return nil
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func getAPIKeyFromKeychain() (string, error) {
	cmd := exec.Command("security", "find-generic-password", "-s", keychainService, "-a", keychainAccount, "-w")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func storeAPIKeyInKeychain(apiKey string) error {
	cmd := exec.Command("security", "add-generic-password", "-U", "-s", keychainService, "-a", keychainAccount, "-w", apiKey)
	return cmd.Run()
}

func promptForAPIKey() (string, error) {
	fmt.Print("Enter your NASA API key: ")
	var apiKey string
	_, err := fmt.Scanln(&apiKey)
	return strings.TrimSpace(apiKey), err
}

func getAPIKey(reset bool) (string, error) {
	if reset {
		apiKey, err := promptForAPIKey()
		if err != nil {
			return "", err
		}
		if err := storeAPIKeyInKeychain(apiKey); err != nil {
			return "", err
		}
		return apiKey, nil
	}
	apiKey, err := getAPIKeyFromKeychain()
	if err == nil && apiKey != "" {
		return apiKey, nil
	}
	apiKey, err = promptForAPIKey()
	if err != nil {
		return "", err
	}
	if err := storeAPIKeyInKeychain(apiKey); err != nil {
		return "", err
	}
	return apiKey, nil
}

// ==== MAIN ====

func main() {
	reset := false
	for _, arg := range os.Args[1:] {
		if arg == "--reset" {
			reset = true
		}
	}

	apiKey, err := getAPIKey(reset)
	if err != nil {
		fmt.Println("Error retrieving API key:", err)
		return
	}

	options := []func(string) (string, error){
		getApodImageURL,
		getRandomNasaImageURL,
		//getMarsImageURL,
		//getRandomEarthImageURL,
		//getEpicImageURL,
	}

	chosen := options[rand.Intn(len(options))]
	imageURL, err := chosen(apiKey)
	if err != nil {
		fmt.Println("Error getting image:", err)
		return
	}

	tmpFile, err := os.CreateTemp("", "nasa_wallpaper_*.jpg")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		cerr := tmpFile.Close()
		if cerr != nil {
			log.Printf("error closing temp file: %v", cerr)
		}
	}()

	imagePath := tmpFile.Name()
	fmt.Println("Downloading from:", imageURL)
	if err := downloadImage(imageURL, imagePath); err != nil {
		fmt.Println("Download error:", err)
		return
	}

	if err := setWallpaper(imagePath); err != nil {
		fmt.Println("Failed to set wallpaper:", err)
	} else {
		fmt.Println("Wallpaper set successfully!")
	}
}
