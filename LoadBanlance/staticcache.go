package LoadBanlance

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
)

const cacheDir string = "./LoadBanlance/cache"

var SupportMimeType []string = []string{
	"text/html",
	"text/plain",
	"text/xml",
	"text/css",
	"text/javascript",
	"text/json",
	"image/jpeg",
	"image/png",
	"image/gif",
	"image/bmp",
	"image/webp",
}

func CheckType(url string) bool {
	ext := path.Ext(url)
	if ext == "js" {
		return true
	}
	for _, mimeType := range SupportMimeType {
		if strings.HasSuffix(mimeType, ext) {
			return true
		}
	}
	return false
}

func CheckCache(url string) (bool, error) {
	if !CheckType(url) {
		return false, nil
	}
	filename := fmt.Sprintf("%x", md5.Sum([]byte(url)))
	file, err := os.Open(cacheDir + "/" + filename)
	if errors.Is(err, os.ErrNotExist) {
		return false, errors.New(url + " not exist")
	} else if err != nil {
		return false, err
	}
	defer file.Close()
	return true, nil
}

func SendCache(w http.ResponseWriter, url string) error {
	filename := fmt.Sprintf("%x", md5.Sum([]byte(url)))
	data, err := os.ReadFile(cacheDir + "/" + filename)
	if errors.Is(err, os.ErrNotExist) {
		return errors.New(filename + " not exist")
	} else if err != nil {
		return err
	}
	w.Header().Set("Cached", "true")
	fmt.Println(url, "Content-Length:", len(data))
	_, err = w.Write(data)
	return err
}

func StoreCache(d io.Reader, url string) error {
	filename := fmt.Sprintf("%x", md5.Sum([]byte(url)))
	data, err := io.ReadAll(d)
	if err != nil {
		return err
	}
	err = os.WriteFile(cacheDir+"/"+filename, data, 0644)
	return err
}
