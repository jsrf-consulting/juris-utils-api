package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/oklog/ulid/v2"
)

func convertDocxToPdf(inputFile, outputFile string) error {
	cmd := exec.Command("libreoffice", "--headless", "--convert-to", "pdf", inputFile, "--outdir", filepath.Dir(outputFile))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // Limit upload size to 10MB
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Unable to retrieve file from form-data", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tempFilaName := ulid.Make()
	tempFile, err := os.Create(tempFilaName.String() + ".docx")
	if err != nil {
		http.Error(w, "Unable to create temp file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())

	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, "Unable to copy file to temp location", http.StatusInternalServerError)
		return
	}

	outputFile := tempFilaName.String() + ".pdf"
	err = convertDocxToPdf(tempFile.Name(), outputFile)
	if err != nil {
		http.Error(w, "Error converting file to PDF", http.StatusInternalServerError)
		return
	}
	defer os.Remove(outputFile)

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", handler.Filename+".pdf"))
	w.Header().Set("Content-Type", "application/pdf")

	output, err := os.Open(outputFile)
	if err != nil {
		errorExtra := "Unable to open converted PDF file output" + outputFile
		http.Error(w, errorExtra, http.StatusInternalServerError)
		return
	}
	defer output.Close()

	_, err = io.Copy(w, output)
	if err != nil {
		http.Error(w, "Unable to send converted PDF file", http.StatusInternalServerError)
		return
	}
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/convert", uploadHandler).Methods("POST")

	http.Handle("/", r)
	fmt.Println("Server started at :8080")
	http.ListenAndServe(":8080", nil)
}
