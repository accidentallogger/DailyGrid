package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type Task struct {
	Title  string `json:"title"`
	Status string `json:"status"` // "todo", "done", "backlog"
}

type Goal struct {
	Title string `json:"title"`
	Days  int    `json:"days"`
	Tasks []Task `json:"tasks"`
}

// User structure for user authentication
type User struct {
	Username string
	Password string // Store hashed passwords
}

var (
	db           *sql.DB
	llm          *ollama.LLM
	sessionStore = make(map[string]string) // Simple session store
	loginAttempts = make(map[string]int)    // Track login attempts
)

// Main function to start the HTTP server
func main() {
	var err error
	llm, err = ollama.New(ollama.WithModel("llama3.2"))
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the database
	db, err = sql.Open("sqlite", "file:goals.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create users and goals tables
	createTables()

	http.HandleFunc("/register", handleRegister)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/", handleGoalsPage)
	http.HandleFunc("/add-goal", handleAddGoal)
	http.HandleFunc("/update-task", handleUpdateTask) // New endpoint for task updates
	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func createTables() {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL
	);`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS goals (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		days INTEGER NOT NULL,
		tasks TEXT NOT NULL
	);`)
	if err != nil {
		log.Fatal(err)
	}
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.ServeFile(w, r, "register2.html")
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form data", http.StatusInternalServerError)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", username, hashedPassword)
	if err != nil {
		http.Error(w, "Error saving user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/login?success=1", http.StatusSeeOther)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.ServeFile(w, r, "login2.html")
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form data", http.StatusInternalServerError)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	ip := r.RemoteAddr
	if loginAttempts[ip] >= 5 {
		http.Error(w, "Too many login attempts", http.StatusTooManyRequests)
		return
	}

	var user User
	err = db.QueryRow("SELECT username, password FROM users WHERE username = ?", username).Scan(&user.Username, &user.Password)
	if err != nil {
		loginAttempts[ip]++
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		loginAttempts[ip]++
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	loginAttempts[ip] = 0
	sessionID := fmt.Sprintf("%s-session", username)
	sessionStore[sessionID] = username
	http.SetCookie(w, &http.Cookie{Name: "session_id", Value: sessionID, Path: "/", MaxAge: 3600})

	http.Redirect(w, r, "/?success=1", http.StatusSeeOther)
}

func handleGoalsPage(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil || sessionStore[cookie.Value] == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles("goals.html")
	if err != nil {
		http.Error(w, "Error parsing template", http.StatusInternalServerError)
		return
	}

	goals, err := fetchGoals()
	if err != nil {
		http.Error(w, "Error fetching goals", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, goals)
	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}

func fetchGoals() ([]Goal, error) {
	rows, err := db.Query("SELECT title, days, tasks FROM goals")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []Goal
	for rows.Next() {
		var g Goal
		var tasksJSON string
		if err := rows.Scan(&g.Title, &g.Days, &tasksJSON); err != nil {
			return nil, err
		}
		var tasks []Task
		if err := json.Unmarshal([]byte(tasksJSON), &tasks); err != nil {
			return nil, err
		}
		g.Tasks = tasks
		goals = append(goals, g)
	}
	return goals, nil
}

func handleAddGoal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form data", http.StatusInternalServerError)
		return
	}

	goalTitle := r.FormValue("goalTitle")
	daysStr := r.FormValue("days")

	days, err := strconv.Atoi(daysStr)
	if err != nil {
		http.Error(w, "Invalid number of days", http.StatusBadRequest)
		return
	}

	tasks := generateTasksForGoal(goalTitle, days)

	var taskList []Task
	for _, task := range tasks {
		taskList = append(taskList, Task{Title: task, Status: "todo"})
	}

	tasksJSON, err := json.Marshal(taskList)
	if err != nil {
		http.Error(w, "Error saving tasks", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT INTO goals (title, days, tasks) VALUES (?, ?, ?)", goalTitle, days, tasksJSON)
	if err != nil {
		http.Error(w, "Error saving goal", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var updateData struct {
		Title  string `json:"title"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Logic to update task in the database goes here
	// This will require fetching the goal and modifying the task's status accordingly

	w.WriteHeader(http.StatusNoContent)
}

func generateTasksForGoal(goalTitle string, days int) []string {
	ctx := context.Background()
	var responseBuffer strings.Builder

	streamFunc := func(ctx context.Context, chunk []byte) error {
		responseBuffer.Write(chunk)
		return nil
	}

	prompt := fmt.Sprintf("Give me day-by-day tasks to achieve the goal: '%s' in '%d' days. Return the array of steps just titles and put them in one line for each day the list should be like day1:, day2:, day3:, etc. Skip any introduction line. just give me the array and nothing else as output ever again. every array must start with a good or bad or maybe to show if it is feasible or not to achieve the goal in that many days",
		goalTitle, days)

	_, err := llm.Call(ctx, prompt,
		llms.WithTemperature(0.8),
		llms.WithStreamingFunc(streamFunc),
	)
	if err != nil {
		log.Println("Error generating tasks:", err)
		return []string{"Error generating tasks."}
	}

	fullResponse := strings.TrimSpace(responseBuffer.String())
	fullResponse = strings.Trim(fullResponse, "[]")
	tasks := strings.Split(fullResponse, "day")

	var dailyTasks []string
	for _, task := range tasks {
		if trimmed := strings.TrimSpace(task); trimmed != "" {
			dailyTasks = append(dailyTasks, trimmed)
		}
	}

	return dailyTasks
}

