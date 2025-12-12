package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"NasaBG/apod"
	"NasaBG/imagesapi"
)

const keychainService = "NasaBG_APIKey"
const keychainAccount = "nasa_api_key"
const lastWallpaperPathFile = "~/.nasa_wallpaper_last"

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
	// First, check environment variable
	if envKey := os.Getenv("NASA_API_KEY"); envKey != "" {
		return envKey, nil
	}
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

func expandHome(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			return strings.Replace(path, "~", home, 1)
		}
	}
	return path
}

func getLastWallpaperPath() (string, error) {
	path := expandHome(lastWallpaperPathFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func setLastWallpaperPath(wallpaperPath string) error {
	path := expandHome(lastWallpaperPathFile)
	return os.WriteFile(path, []byte(wallpaperPath), 0644)
}

func removeOldWallpaper() {
	oldPath, err := getLastWallpaperPath()
	if err == nil && oldPath != "" {
		if _, err := os.Stat(oldPath); err == nil {
			_ = os.Remove(oldPath)
		}
	}
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
		apod.GetApodImageURL,
		imagesapi.GetRandomNasaImageURL,
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

	removeOldWallpaper()

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
		_ = setLastWallpaperPath(imagePath)
	}
}
