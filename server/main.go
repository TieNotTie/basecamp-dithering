package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
)

func cleanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}
	err := os.RemoveAll("downloads")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	err = os.RemoveAll("images")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	// Maximum upload of 10 MB files
	r.ParseMultipartForm(10 << 20)

	// Get handler for filename, size and headers
	file, handler, err := r.FormFile("image")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return
	}

	defer file.Close()
	//fmt.Printf("Uploaded File: %+v\n", handler.Filename)
	//fmt.Printf("File Size: %+v\n", handler.Size)
	//fmt.Printf("MIME Header: %+v\n", handler.Header)

	folderName := "downloads/"
	createFolder(folderName)

	// Create file
	createFile(folderName, handler.Filename, file, w)

	//fmt.Fprintf(w, "Successfully Uploaded File\n")
	ditherImage(handler.Filename)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		uploadFile(w, r)
	}
}

func addHeaders(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	}
}

func createFolder(folderName string) {
	if _, err := os.Stat(folderName); os.IsNotExist(err) {
		os.Mkdir(folderName, 0755)
	}
}

func createFile(path, fileName string, file multipart.File, w http.ResponseWriter) {
	dst, err := os.Create(path + fileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy the uploaded file to the created file on the filesystem
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ditherImage(fileName string) {
	const resultFolderName string = "images/"
	const resourceFolderName string = "downloads/"

	var width, height int = getImageDimension(resourceFolderName + fileName)

	img := image.NewGray16(image.Rectangle{image.Point{0, 0}, image.Point{width, height}})

	file, _ := os.Open(resourceFolderName + fileName)

	//sourceImage, _, err := image.Decode(file)
	//if err != nil {

	//}
	file.Close()

	for y := 0; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			//r, g, b, _ :=
			oldColor := img.Gray16At(x, y).Y
			newColor := getClosestColor(oldColor)

			img.Set(x, y, color.Gray16{newColor})
			quantError := oldColor - newColor
			setNearPixels(x, y, quantError, img)
		}
	}

	createFolder(resultFolderName)
	f, _ := os.Create(resultFolderName + fileName)
	jpeg.Encode(f, img, nil)
	f.Close()
}

func setNearPixels(x, y int, quantError uint16, dst *image.Gray16) {
	oldGray := dst.Gray16At(x, y).Y
	//oldGray := rgbToGrayscale(r, g, b)
	dst.Set(x+1, y, color.Gray16{oldGray + uint16(float64(quantError)*(float64(7)/16))})
	dst.Set(x-1, y+1, color.Gray16{oldGray + uint16(float64(quantError)*(float64(3)/16))})
	dst.Set(x, y+1, color.Gray16{oldGray + uint16(float64(quantError)*(float64(5)/16))})
	dst.Set(x+1, y+1, color.Gray16{oldGray + uint16(float64(quantError)*(float64(1)/16))})
}

func rgbaToGrayscale(r, g, b, a uint32) uint16 {
	return uint16(r/3 + g/3 + b/3)
}

func getClosestColor(grayColor uint16) uint16 {
	return uint16(math.Round(float64(grayColor)/math.MaxUint16) * math.MaxUint16)
}

func getImageDimension(imagePath string) (int, int) {
	file, err := os.Open(imagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}

	image, _, err := image.DecodeConfig(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", imagePath, err)
	}
	defer file.Close()
	return image.Width, image.Height
}

func main() {

	const port string = "8080"
	//Upload route
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/clean", cleanHandler)

	http.Handle("/", addHeaders(http.FileServer(http.Dir(""))))

	//Listen on port 8080
	fmt.Printf("Starting server at port %s\n", port)
	http.ListenAndServe(":"+port, nil)
	ditherImage("")

}
