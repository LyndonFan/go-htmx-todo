package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type Todo struct {
	ID           int       `json:"id"`
	Description  string    `json:"description"`
	CreatedDate  time.Time `json:"created_date"`
	DeadlineDate time.Time `json:"deadline_date"`
	Status       string    `json:"status"`
}

var (
	todoRowTemplate = parseTemplate("templates/todos.html", "todoRow")
	homeTemplate    = parseTemplate("templates/index.html", "home")
)

func parseTemplate(filename, templateName string) *template.Template {
	tmpl, err := template.New(templateName).ParseFiles(filename)
	if err != nil {
		log.Fatal(err)
	}
	return tmpl
}

func main() {
	var err error
	db, err = sql.Open("sqlite3", "todo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	r := mux.NewRouter()
	r.HandleFunc("/", GetHomePageHTML).Methods("GET")
	r.HandleFunc("/todos", GetAllTodosHTML).Methods("GET")
	r.HandleFunc("/todos", CreateTodo).Methods("POST")
	r.HandleFunc("/todos/{id}", GetTodoHTML).Methods("GET")
	r.HandleFunc("/todos/{id}", UpdateTodo).Methods("PUT")
	r.HandleFunc("/todos/{id}", DeleteTodo).Methods("DELETE")

	http.Handle("/", r)

	fmt.Println("Server is running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func GetHomePageHTML(w http.ResponseWriter, r *http.Request) {
	log.Default().Println("GetHomePageHTML")
	w.Header().Set("Content-Type", "text/html")
	err := homeTemplate.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func GetAllTodosHTML(w http.ResponseWriter, r *http.Request) {
	log.Default().Println("GetAllTodosHTML")
	todos := []Todo{}
	rows, err := db.Query("SELECT * FROM todos")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var todo Todo
		err := rows.Scan(&todo.ID, &todo.Description, &todo.CreatedDate, &todo.DeadlineDate, &todo.Status)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		todos = append(todos, todo)
	}

	w.Header().Set("Content-Type", "text/html")

	for _, todo := range todos {
		err := todoRowTemplate.Execute(w, todo)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func GetTodoHTML(w http.ResponseWriter, r *http.Request) {
	log.Default().Println("GetTodoHTML")
	vars := mux.Vars(r)
	id := vars["id"]

	var todo Todo
	err := db.QueryRow("SELECT * FROM todos WHERE id = ?", id).Scan(&todo.ID, &todo.Description, &todo.CreatedDate, &todo.DeadlineDate, &todo.Status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = todoRowTemplate.Execute(w, todo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func CreateTodo(w http.ResponseWriter, r *http.Request) {
	log.Default().Println("CreateTodo")
	var todo Todo
	err := json.NewDecoder(r.Body).Decode(&todo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Insert the new todo into the database
	_, err = db.Exec("INSERT INTO todos (description, created_date, deadline_date, status) VALUES (?, ?, ?, ?)",
		todo.Description, time.Now(), todo.DeadlineDate, todo.Status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func UpdateTodo(w http.ResponseWriter, r *http.Request) {
	log.Default().Println("UpdateTodo")
	vars := mux.Vars(r)
	id := vars["id"]
	var todo Todo

	err := json.NewDecoder(r.Body).Decode(&todo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = db.Exec("UPDATE todos SET description = ?, deadline_date = ?, status = ? WHERE id = ?", todo.Description, todo.DeadlineDate, todo.Status, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteTodo(w http.ResponseWriter, r *http.Request) {
	log.Default().Println("DeleteTodo")
	vars := mux.Vars(r)
	id := vars["id"]

	_, err := db.Exec("DELETE FROM todos WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
