package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type Todo struct {
	ID           int
	Description  string
	CreatedDate  time.Time
	DeadlineDate time.Time
	Status       string
}

type TodoDisplay struct {
	ID           int
	Description  string
	CreatedDate  string
	DeadlineDate string
	Status       string
}

func (t Todo) ToDisplay() TodoDisplay {
	return TodoDisplay{
		ID:           t.ID,
		Description:  t.Description,
		CreatedDate:  t.CreatedDate.Format("2006-01-02"),
		DeadlineDate: t.DeadlineDate.Format("2006-01-02"),
		Status:       t.Status,
	}
}

func (t TodoDisplay) FromDisplay() (Todo, error) {
	createdDate, err := time.Parse("2006-01-02", t.CreatedDate)
	if err != nil {
		return Todo{}, err
	}
	deadlineDate, err := time.Parse("2006-01-02", t.DeadlineDate)
	if err != nil {
		return Todo{}, err
	}
	res := Todo{
		ID:           t.ID,
		Description:  t.Description,
		CreatedDate:  createdDate,
		DeadlineDate: deadlineDate,
		Status:       t.Status,
	}
	return res, nil
}

var (
	todoRowTemplate  = parseTemplate("templates/todoRow.html", "todoRow")
	todoEditTemplate = parseTemplate("templates/todoEdit.html", "todoEdit")
	homeTemplate     = parseTemplate("templates/index.html", "home")
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
	r.HandleFunc("/todos/edit/{id}", EditTodo).Methods("GET")
	r.HandleFunc("/todos/{id}", UpdateTodo).Methods("PUT")
	r.HandleFunc("/todos/{id}", DeleteTodoHTML).Methods("DELETE")

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
		todoForDisplay := todo.ToDisplay()
		err := todoRowTemplate.Execute(w, todoForDisplay)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	fmt.Fprint(w, "<tr id='last'></tr>")
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
	todoForDisplay := todo.ToDisplay()
	err = todoRowTemplate.Execute(w, todoForDisplay)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func EditTodo(w http.ResponseWriter, r *http.Request) {
	log.Default().Println("EditTodo")
	vars := mux.Vars(r)
	id := vars["id"]

	var todo Todo
	err := db.QueryRow("SELECT * FROM todos WHERE id = ?", id).Scan(&todo.ID, &todo.Description, &todo.CreatedDate, &todo.DeadlineDate, &todo.Status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = todoEditTemplate.Execute(w, todo.ToDisplay())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func CreateTodo(w http.ResponseWriter, r *http.Request) {
	log.Default().Println("CreateTodo")

	todo := Todo{
		Description:  "",
		CreatedDate:  time.Now(),
		DeadlineDate: time.Now().Add(24 * time.Hour),
		Status:       "Waiting",
	}

	// Insert the new todo into the database
	res, err := db.Exec(`
    	INSERT INTO todos (description, created_date, deadline_date, status)
    	VALUES (?, ?, ?, ?)
		RETURNING id  
	   `,
		todo.Description, time.Now(), todo.DeadlineDate, todo.Status,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n, err := res.LastInsertId()
	fmt.Printf("res: %d %v\n", n, err)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	todo.ID = int(n)
	todoForDisplay, err := todo.ToDisplay().FromDisplay()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	err = todoEditTemplate.Execute(w, todoForDisplay)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	fmt.Fprint(w, "<tr id='last'></tr>")
}

func UpdateTodo(w http.ResponseWriter, r *http.Request) {
	log.Default().Println("UpdateTodo")
	vars := mux.Vars(r)
	id := vars["id"]
	var todoFromDisplay TodoDisplay
	defer r.Body.Close()

	bodyBytes, err := io.ReadAll(r.Body)
	bodyString := string(bodyBytes)
	log.Default().Printf("bodyBytes: %s\n", bodyString)
	queryParameters, err := url.ParseQuery(bodyString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("queryParameters: %v\n", queryParameters)

	for key, value := range queryParameters {
		if len(value) != 1 {
			http.Error(w, fmt.Sprintf("Not exactly 1 argument for %s", key), http.StatusBadRequest)
			return
		}
		switch key {
		case "id":
			n, err := strconv.Atoi(value[0])
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			todoFromDisplay.ID = n
		case "description":
			todoFromDisplay.Description = value[0]
		case "created_date":
			todoFromDisplay.CreatedDate = value[0]
		case "deadline_date":
			todoFromDisplay.DeadlineDate = value[0]
		case "status":
			todoFromDisplay.Status = value[0]
		default:
			http.Error(w, "Can't find any fields", http.StatusBadRequest)
			return
		}
	}

	todo, err := todoFromDisplay.FromDisplay()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("UPDATE todos SET description = ?, deadline_date = ?, status = ? WHERE id = ?", todo.Description, todo.DeadlineDate, todo.Status, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = todoRowTemplate.Execute(w, todoFromDisplay)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func DeleteTodoHTML(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	log.Default().Printf("DeleteTodo: %s\n", id)

	_, err := db.Exec("DELETE FROM todos WHERE id = ?", id)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK) // 200, NOT 204, as to trigger HTMX
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(""))
}
