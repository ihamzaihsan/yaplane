package Error

import (
	"html/template"
	"log"
	"net/http"
)

func ServeError(w http.ResponseWriter, r *http.Request, statusCode int) {
	var errorMessage string
	switch statusCode {
	case http.StatusNotFound:
		errorMessage = "Page not found"
	case http.StatusBadRequest:
		errorMessage = "Bad request"
	case http.StatusInternalServerError:
		errorMessage = "Internal server error"
	case http.StatusMethodNotAllowed:
		errorMessage = "Method not allowed"
	default:
		errorMessage = "Unexpected error"
	}

	data := struct {
		ErrorCode    int
		ErrorMessage string
	}{
		ErrorCode:    statusCode,
		ErrorMessage: errorMessage,
	}

	w.WriteHeader(statusCode)
	tmpl, err := template.ParseFiles("static/templates/error.html")
	if err != nil {
		log.Printf("Template parse error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Template execute error: %v", err)
	}
}