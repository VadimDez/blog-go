package main

import (
	"io/ioutil"
	"net/http"
	"html/template"
	"regexp"
	"time"
	"log"
	"fmt"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

var templates = template.Must(template.ParseFiles("./tmpl/index.html", "./tmpl/edit.html", "./tmpl/view.html"))
var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
const PORT = 8080
var DATABASE *sql.DB

type Page struct {
	Title string
	Body []byte
}

func (p *Page) save() error {
	filename := p.Title + ".txt"
	return ioutil.WriteFile("data/" + filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := ioutil.ReadFile("data/" + filename)

	if err != nil {
		return nil, err
	}

	return &Page{Title: title, Body: body}, nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl + ".html", p)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)

	if err != nil {
		http.Redirect(w, r, "/edit/" + title, http.StatusFound)
		return
	}

	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)

	if err != nil {
		p = &Page{Title: title}
	}

	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")

	p := &Page{ Title: title, Body: []byte(body) }
	err := p.save()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/view/" + title, http.StatusFound)
}

func makeHandler(fn func (http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)

		if m == nil {
			http.NotFound(w, r)
			return
		}

		fn(w, r, m[2])
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	params := struct {
		Name string
		Year int
	} { "Blog system", time.Now().Year() }

	err := templates.ExecuteTemplate(w, "index.html", params)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}


func db() {
	var err error
	DATABASE, err = sql.Open("sqlite3", "./blog.db")

	if err != nil {
		log.Fatal("No database")
	}

	stmt, err := DATABASE.Prepare("INSERT INTO pages(title, content) VALUES(?,?)")

	if err != nil {
		log.Fatal("Insert issue")
	}

	_, err = stmt.Exec("test", "cont")
}

func main() {
	db()

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	port := fmt.Sprintf(":%d", PORT)
	err := http.ListenAndServe(port, nil)

	if err != nil {
		log.Fatal("Filed starting server: ", err)
	}
}