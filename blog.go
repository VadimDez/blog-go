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
	"strconv"
)

var templates = template.Must(template.ParseFiles("./tmpl/index.html", "./tmpl/edit.html", "./tmpl/view.html"))
var validPath = regexp.MustCompile("^/(new/?|(edit|save|view)/([0-9]+))$")
const PORT = 8080
var DATABASE *sql.DB

type Page struct {
	Id int
	Title string
	Content []byte
}

func (p *Page) save() error {
	filename := p.Title + ".txt"
	return ioutil.WriteFile("data/" + filename, p.Content, 0600)
}

func loadPage(id int64) (*Page, error) {
	var page Page
	err := DATABASE.QueryRow("SELECT * FROM pages WHERE id = ?", id).Scan(&page.Id, &page.Title, &page.Content)

	if err != nil {
		return nil, err
	}

	return &page, nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl + ".html", p)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, id int64) {
	p, err := loadPage(id)

	if err != nil {
		log.Fatalf("View: Load error: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, id int64) {
	p, err := loadPage(id)

	if err != nil {
		p = &Page{Title: "not found"}
	}

	renderTemplate(w, "edit", p)
}

func newHandler(w http.ResponseWriter, r *http.Request, id int64) {
	renderTemplate(w, "edit", &Page{})
}

func saveHandler(w http.ResponseWriter, r *http.Request, id int64) {
	title := r.FormValue("title")
	content := r.FormValue("content")

	stmt, err := DATABASE.Prepare("INSERT INTO pages(title, content) VALUES(?,?)")

	if err != nil {
		log.Fatal("Insert issue")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	res, err := stmt.Exec(title, content)

	if err != nil {
		log.Fatal("Insert issue")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newId, err := res.LastInsertId()

	if err != nil {
		log.Fatal("Insert issue")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/view/%d", newId), http.StatusFound)
}

func makeHandler(fn func (http.ResponseWriter, *http.Request, int64)) http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)

		if m == nil {
			http.NotFound(w, r)
			return
		}

		id, _ := strconv.ParseInt(m[3], 10, 32)

		fn(w, r, id)
	}
}

func getPages() []Page {
	pages := []Page{}

	rows, err := DATABASE.Query("SELECT * FROM pages")

	if err != nil {
		log.Fatalf("Select error: %s", err)
		return pages
	}

	for rows.Next() {
		var page Page

		err = rows.Scan(&page.Id, &page.Title, &page.Content)

		if err != nil {
			log.Fatalf("scan error: %s", err)
		}

		pages = append(pages, page)
	}

	return pages
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	pages := getPages()

	params := struct {
		Name string
		Year int
		Pages []Page
	} { "Blog system", time.Now().Year(), pages }

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
}

func main() {
	db()

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/new/", makeHandler(newHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	port := fmt.Sprintf(":%d", PORT)
	err := http.ListenAndServe(port, nil)

	if err != nil {
		log.Fatal("Filed starting server: ", err)
	}
}