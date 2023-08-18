package main

import (
	"embed"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

//go:embed templates/*.html
var templateFiles embed.FS

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Load the template from the embedded file
	templateData, err := templateFiles.ReadFile("templates/upload.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Parse the template
	tmpl, err := template.New("home").Parse(string(templateData))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Method == "GET" {
		// Display the form
		err = tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if r.Method == "POST" {
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			log.Println("Error parsing the form:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// get value of workdir
		var workdir = r.FormValue("workdir")
		// Process the form submission
		file, handler, err := r.FormFile("file")
		if err != nil {
			log.Println("Error retrieving the file:", err)
			return
		}
		defer file.Close()

		// Get the current date
		currentTime := time.Now()

		// Format the date as "20130523"
		date := currentTime.Format("20060102")
		outputDir := filepath.Join("public", date)
		err = os.MkdirAll(outputDir, 0755)
		if err != nil {
			log.Println("Error mkdir:", workdir, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		outputDir, err = filepath.Abs(outputDir)
		if err != nil {
			log.Println("Error abs:", workdir, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Save the uploaded file
		uploadFile := filepath.Join(workdir, handler.Filename)
		f, err := os.OpenFile(uploadFile, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Println("Error saving the file:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()

		_, err = io.Copy(f, file)
		if err != nil {
			log.Println("Error copying the file:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var cmd = exec.Command(
			"util/SeqAnalysis/SeqAnalysis.exe", "-i", uploadFile, "-w", workdir, "-o", outputDir, "-zip",
		)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		log.Println(cmd)
		err = cmd.Run()
		if err != nil {
			log.Println("Error run SeqAnalysis:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, date)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	}
}
