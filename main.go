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
	"time"
)

const apiKey = "" // Replace with your real API key

// ==== Mars Rover ====

type MarsPhoto struct {
	ImgSrc string `json:"img_src"`
}

type ErrorResponse struct {
	Msg string `json:"msg"`
}

type MarsResponse struct {
	Photos []MarsPhoto `json:"photos"`
}

func fetchAPIResponse(url string, result interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "image/") {
		// Handle image response
		_, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return nil
	}

	// Handle JSON response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var errorResponse ErrorResponse
	if err := json.Unmarshal(body, &errorResponse); err == nil && errorResponse.Msg != "" {
		return fmt.Errorf("API error: %s", errorResponse.Msg)
	}

	return json.Unmarshal(body, result)
}

func getImageURLWithRollingDate(baseURL string, processResponse func(interface{}) (string, error), daysToRoll int, resultType interface{}) (string, error) {
	date := time.Now()
	for i := 0; i < daysToRoll; i++ {
		dateStr := date.Format("2006-01-02")
		url := fmt.Sprintf(baseURL, dateStr)
		fmt.Println("Checking URL:", url)

		if err := fetchAPIResponse(url, resultType); err != nil {
			date = date.AddDate(0, 0, -1) // Try the previous day
			continue
		}

		if imgURL, err := processResponse(resultType); err == nil {
			if imgURL != "" {
				return imgURL, nil
			}
			return url, nil
		}

		date = date.AddDate(0, 0, -1)
	}
	return "", fmt.Errorf("no images found")
}

func getMarsImageURL() (string, error) {
	baseURL := "https://api.nasa.gov/mars-photos/api/v1/rovers/curiosity/photos?earth_date=%s&api_key=" + apiKey
	marsData := &MarsResponse{}
	return getImageURLWithRollingDate(baseURL, func(data interface{}) (string, error) {
		marsData, ok := data.(*MarsResponse)
		if !ok || len(marsData.Photos) == 0 {
			return "", fmt.Errorf("no Mars images found")
		}
		return marsData.Photos[rand.Intn(len(marsData.Photos))].ImgSrc, nil
	}, 30, marsData)
}

func getRandomEarthImageURL() (string, error) {
	baseURLTemplate := "https://api.nasa.gov/planetary/earth/imagery?lon=%f&lat=%f&date=%%s&dim=0.2&api_key=%s"

	for i := 0; i < 5; i++ {
		lat := -90 + rand.Float64()*180
		lon := -180 + rand.Float64()*360

		baseURL := fmt.Sprintf(baseURLTemplate, lon, lat, apiKey)

		imgURL, err := getImageURLWithRollingDate(baseURL, func(data interface{}) (string, error) {
			// Earth API directly provides the image URL
			return "", nil
		}, 3, nil)

		if err == nil {
			return imgURL, nil
		}
		fmt.Printf("Attempt %d failed: %v\n", i+1, err)
	}

	return "", fmt.Errorf("failed to get Earth image after 5 attempts")
}

// ==== EPIC ====

type EpicImage struct {
	Image string `json:"image"`
	Date  string `json:"date"`
}

func getEpicImageURL() (string, error) {
	url := fmt.Sprintf("https://api.nasa.gov/EPIC/api/natural?api_key=%s", apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data []EpicImage
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	if len(data) == 0 {
		return "", fmt.Errorf("no EPIC images available")
	}

	img := data[rand.Intn(len(data))]
	dateParts := strings.Split(img.Date, " ")[0]
	datePath := strings.ReplaceAll(dateParts, "-", "/")
	imageURL := fmt.Sprintf("https://epic.gsfc.nasa.gov/archive/natural/%s/png/%s.png", datePath, img.Image)

	return imageURL, nil
}

// ==== Shared Utils ====

func downloadImage(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

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

// ==== MAIN ====

func main() {
	// rand.Seed(time.Now().UnixNano())

	options := []func() (string, error){
		getMarsImageURL,
		getRandomEarthImageURL,
		getEpicImageURL,
	}

	chosen := options[rand.Intn(len(options))]
	imageURL, err := chosen()
	if err != nil {
		fmt.Println("Error getting image:", err)
		return
	}

	tmpFile, err := os.CreateTemp("", "nasa_wallpaper_*.jpg")
	if err != nil {
		log.Fatal(err)
	}
	defer tmpFile.Close()

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
