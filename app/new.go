package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

// Structure to store the goal and its daily tasks
type Goal struct {
	Title string
	Days  int
	Tasks []string
}

var goals []Goal // Store multiple goals
var llm *ollama.LLM

func main() {
	var err error
	llm, err = ollama.New(ollama.WithModel("llama3.2"))
	if err != nil {
		log.Fatal(err)
	}

	// Start the HTTP server to handle user input and display goals
	http.HandleFunc("/app2", handleGoalsPage)
	http.HandleFunc("/add-goal", handleAddGoal)
	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Handle the request and serve the HTML page with goals and input form
func handleGoalsPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("goals").Parse(
				<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Goals and Tasks</title>
			<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/5.15.4/css/all.min.css">
			<style>
				body { 
					font-family: Arial, sans-serif; 
					margin: 20px; 
					background-color: #f4f4f4; 
					color: #333; 
				}
				h1, h2 { 
					color: #4CAF50; 
				}
				.goal { 
					background-color: white; 
					border-radius: 8px; 
					box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1); 
					margin-bottom: 30px; 
					padding: 20px; 
				}
				.goal-title { 
					font-weight: bold; 
					font-size: 1.2em; 
				}
				.tasks-container { 
					display: flex; 
					justify-content: space-between; 
					margin-top: 10px; 
				}
				.todo, .done, .backlog { 
					border: 1px solid #ccc; 
					border-radius: 5px; 
					padding: 10px; 
					margin-right: 20px; 
					width: 30%; 
					background-color: #f9f9f9; 
				}
				.backlog { 
					background-color: #FFCCCC; 
					color: #C0392B; 
				}
				.task { 
					margin-bottom: 5px; 
					padding: 10px; 
					background: #e7f3fe; 
					color: #31708f; 
					border-radius: 4px; 
					cursor: pointer; 
					transition: background 0.3s; 
				}
				.task:hover { 
					background: #d0e6f9; 
				}
				.task.dragging { 
					opacity: 0.5; 
				}
				form { 
					margin-bottom: 40px; 
				}
				input[type="text"], input[type="number"] {
					width: 100%; 
					padding: 8px; 
					margin: 5px 0; 
					border: 1px solid #ccc; 
					border-radius: 4px; 
				}
				button { 
					background-color: #4CAF50; 
					color: white; 
					padding: 10px 15px; 
					border: none; 
					border-radius: 4px; 
					cursor: pointer; 
				}
				button:hover { 
					background-color: #45a049; 
				}
			</style>
		</head>
		<body>
			<h1>Goals and Tasks</h1>

			<!-- Form to input new goal -->
			<form action="/add-goal" method="POST">
				<label for="goalTitle">Goal Title:</label><br>
				<input type="text" id="goalTitle" name="goalTitle" required><br>
				<label for="days">Number of Days:</label><br>
				<input type="number" id="days" name="days" required><br>
				<button type="submit">Add Goal</button>
			</form>

			<!-- Display Goals -->
			{{ if . }}
				<h2>Your Goals</h2>
				{{ range . }}
				<div class="goal">
					<p class="goal-title">Goal: {{ .Title }} ({{ .Days }} days)</p>
					<div class="tasks-container">
						<div class="todo" id="todo-{{ .Title | urlquery }}">
							<h3>To Do</h3>
							{{ range .Tasks }}
							<div class="task" draggable="true">{{ . }}</div>
							{{ end }}
						</div>
						<div class="done" id="done-{{ .Title | urlquery }}">
							<h3>Done</h3>
						</div>
						<div class="backlog" id="backlog-{{ .Title | urlquery }}">
							<h3>Backlog</h3>
						</div>
					</div>
				</div>
				{{ end }}
			{{ else }}
				<p>No goals added yet. Use the form above to add a goal.</p>
			{{ end }}

			<script>
				const tasks = document.querySelectorAll('.task');
				const todoContainers = document.querySelectorAll('.todo');
				const doneContainers = document.querySelectorAll('.done');
				const backlogContainers = document.querySelectorAll('.backlog');

				tasks.forEach(task => {
					task.addEventListener('dragstart', () => {
						task.classList.add('dragging');
					});

					task.addEventListener('dragend', () => {
						task.classList.remove('dragging');
					});
				});

				todoContainers.forEach(container => {
					container.addEventListener('dragover', (e) => {
						e.preventDefault();
					});
					container.addEventListener('drop', (e) => {
						const draggingTask = document.querySelector('.dragging');
						if (draggingTask) {
							container.appendChild(draggingTask);
						}
					});
				});

				doneContainers.forEach(container => {
					container.addEventListener('dragover', (e) => {
						e.preventDefault();
					});
					container.addEventListener('drop', (e) => {
						const draggingTask = document.querySelector('.dragging');
						if (draggingTask) {
							container.appendChild(draggingTask);
						}
					});
				});

				backlogContainers.forEach(container => {
					container.addEventListener('dragover', (e) => {
						e.preventDefault();
					});
					container.addEventListener('drop', (e) => {
						const draggingTask = document.querySelector('.dragging');
						if (draggingTask) {
							container.appendChild(draggingTask);
						}
					});
				});
			</script>

		</body>
		</html>

	)

	if err != nil {
		http.Error(w, "Error parsing template", http.StatusInternalServerError)
		return
	}

	// Render the template with the current goals
	err = tmpl.Execute(w, goals)
	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}

// Handle form submission to add a new goal
func handleAddGoal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form data", http.StatusInternalServerError)
		return
	}

	goalTitle := r.FormValue("goalTitle")
	daysStr := r.FormValue("days")

	// Convert days to integer
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		http.Error(w, "Invalid number of days", http.StatusBadRequest)
		return
	}

	// Generate tasks for the goal
	tasks := generateTasksForGoal(goalTitle, days)

	// Append the new goal and its tasks to the goals slice
	goals = append(goals, Goal{
		Title: goalTitle,
		Days:  days,
		Tasks: tasks,
	})

	// Redirect to the goals page to display the updated list of goals
	http.Redirect(w, r, "/app2", http.StatusSeeOther)
}

// Function to generate tasks for a given goal
func generateTasksForGoal(goalTitle string, days int) []string {
	ctx := context.Background()

	// Buffer to collect the streamed response chunks
	var responseBuffer strings.Builder

	// Define a streaming function to collect the output
	streamFunc := func(ctx context.Context, chunk []byte) error {
		responseBuffer.Write(chunk) // Collecting the response chunks
		return nil
	}

	// Create the prompt using the user's goal
	prompt := fmt.Sprintf("Give me day-by-day tasks to achieve the goal: '%s' in '%d' days. Return the array of steps just titles and put them in one line for each day the list should be like day1:, day2:, day3:, etc. Skip any introduction line. just give me the array and nothing else as output ever again. every array must start with a good or bad or maybe to show if it is feasible or not to achieve the goal in that many days",
		goalTitle, days)

	_, err := llm.Call(ctx, prompt,
		llms.WithTemperature(0.8),
		llms.WithStreamingFunc(streamFunc),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Once streaming is done, process the response
	fullResponse := responseBuffer.String()

	// Remove any leading or trailing whitespace
	fullResponse = strings.TrimSpace(fullResponse)

	// Remove brackets and split the response into daily tasks
	fullResponse = strings.Trim(fullResponse, "[]") // Remove square brackets
	tasks := strings.Split(fullResponse, "day")      // Split by commas

	// Clean up any empty strings in the tasks
	var dailyTasks []string
	for _, task := range tasks {
		if trimmed := strings.TrimSpace(task); trimmed != "" {
			dailyTasks = append(dailyTasks, trimmed)
		}
	}

	return dailyTasks
}

