package main

import (

	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"
	"github.com/gorilla/sessions"
	_ "modernc.org/sqlite"
	"crypto/rand"
	"encoding/hex"
	"encoding/csv"

	"github.com/xuri/excelize/v2"
	"io"
	"os"
	"path/filepath"
	"strings"
	
		"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	
		"context"


)

// Structure to store the goal and its daily tasks
type Goal struct {
	Title string
	Days  int
	Tasks []string
}

var goals []Goal // Store multiple goals
var llm *ollama.LLM



// Handle the request and serve the HTML page with goals and input form
func handleGoalsPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("goals").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Goals and Tasks</title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/5.15.4/css/all.min.css">
</head>
<body>
<style>
        body { 
            font-family: Arial, sans-serif; 
            margin: 20px; 
            background-color: #f4f4f4; 
            color: #333; 
            max-width: 800px; 
            margin: auto;
        }
        h1, h2 { 
            color: #1E90FF; /* Dodger Blue */
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
            background-color: #CCE0FF; /* Light Blue for Backlog */
            color: #1E90FF; /* Dodger Blue */
        }
        .task { 
            margin-bottom: 5px; 
            padding: 10px; 
            background: #d0e6f9; /* Light blue */
            color: #1E90FF; /* Dodger Blue */
            border-radius: 4px; 
            cursor: pointer; 
            transition: background 0.3s; 
        }
        .task:hover { 
            background: #c0dff9; /* Darker shade of light blue */
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
            background-color: #1E90FF; /* Dodger Blue for buttons */
            color: white; 
            border-radius: 8px;
            box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
            padding: 20px;
            width: 100%;
            max-width: 400px;
            margin: 20px;
            cursor: pointer;
            border: none;
        }
        button:hover { 
            background-color: #1C86EE; /* Slightly darker blue */
        }
        /* Modal styles */
        #loadingModal {
            display: none; 
            position: fixed; 
            top: 0; 
            left: 0; 
            width: 100%; 
            height: 100%; 
            background-color: rgba(0, 0, 0, 0.5); 
            justify-content: center; 
            align-items: center; 
        }
        .modal-content {
            background-color: white; 
            padding: 20px; 
            border-radius: 5px; 
            text-align: center; 
            font-size: 18px; 
        }
    </style>
    <h1>Goals and Tasks</h1>

    <!-- Form to input new goal -->
    <form id="goalForm" action="/add-goal" method="POST">
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

    <!-- Modal for loading state -->
    <div id="loadingModal">
        <div class="modal-content">
            <p><i class="fas fa-spinner fa-spin"></i> Generating...</p>
        </div>
    </div>

    <script>
        // Function to show modal
        function showModal() {
            document.getElementById("loadingModal").style.display = "flex";
        }

        // Function to hide modal
        function hideModal() {
            document.getElementById("loadingModal").style.display = "none";
        }

        // Event listener to display modal on form submission
        document.getElementById("goalForm").addEventListener("submit", function(event) {
            showModal();
        });

        // Drag and drop logic for tasks
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

	`)

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
	http.Redirect(w, r, "/achieve", http.StatusSeeOther)
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
	prompt := fmt.Sprintf("Give me day-by-day tasks to achieve the goal: '%s' in '%d' days. Return the array of steps just titles and put them in one line for each day the list should be like day1:, day2:, day3:, etc. Skip any introduction line. just give me the array and nothing else as output ever again. ", //every array must start with a good or bad or maybe to show if it is feasible or not to achieve the goal in that many days
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


type Holiday struct {
    Day string
}

var holidays = map[string]bool{
    "Monday":    false,
    "Tuesday":   false,
    "Wednesday": false,
    "Thursday":  false,
    "Friday":    false,
    "Saturday":  false,
    "Sunday":    false,
}

func markHolidayHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == "POST" {
        r.ParseForm()
        selectedDay := r.FormValue("holidayDay")
        userID := getUserIDFromSession(r)
        
        if _, ok := holidays[selectedDay]; ok {
            holidays[selectedDay] = true // Mark the selected day as a holiday

            // Delete the timetable for the selected day from the database for the current user
            _, err := db.Exec(`DELETE FROM timetables WHERE user_id = ? AND day = ?`, userID, selectedDay)
            if err != nil {
                http.Error(w, "Failed to clear timetable for the holiday", http.StatusInternalServerError)
                return
            }

            // Remove periods for the selected day from the in-memory timetable object
            timetable.Periods = filterOutPeriodsByDay(timetable.Periods, selectedDay)
            timetable.Gaps = filterOutPeriodsByDay(timetable.Gaps, selectedDay)

            fmt.Fprintf(w, "Successfully marked %s as a holiday and cleared its schedule!", selectedDay)
        } else {
            http.Error(w, "Invalid day selected", http.StatusBadRequest)
        }
    } else {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
    }
}

// Helper function to filter out periods by day from the timetable
func filterOutPeriodsByDay(periods []TimePeriod, day string) []TimePeriod {
    var filtered []TimePeriod
    for _, period := range periods {
        if period.Day != day {
            filtered = append(filtered, period)
        }
    }
    return filtered
}


func clearTimetableHandler(w http.ResponseWriter, r *http.Request) {
    userID := getUserIDFromSession(r)
    if userID == 0 {
        http.Error(w, "User not logged in", http.StatusUnauthorized)
        return
    }

    // Delete the timetable for the user
    _, err := db.Exec(`DELETE FROM timetables WHERE user_id = ?`, userID)
    if err != nil {
        http.Error(w, "Failed to clear timetable", http.StatusInternalServerError)
        return
    }

    // Clear the timetable object in memory
    timetable = &Timetable{}

    http.Redirect(w, r, "/", http.StatusSeeOther)
}


func generateSecretKey() string {
	bytes := make([]byte, 32) // 32 bytes = 256 bits
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

var (
secretKey = generateSecretKey()
store = sessions.NewCookieStore([]byte(secretKey)) // Replace with your secret key
	wakeupTime, _ = time.Parse("15:04", "06:00") // Default wakeup time: 6:00 AM
	sleepTime, _  = time.Parse("15:04", "23:00") // Default sleep time: 11:00 PM
	days          = [7]string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	db            *sql.DB
	timetable     = &Timetable{}
)
func getUserIDFromSession(r *http.Request) int {
	session, _ := store.Get(r, "session-name")
	userID, ok := session.Values["userID"].(int)
	if !ok {
		return 0 // or handle error appropriately
	}
	return userID
}
type TimePeriod struct {
	Name     string
	Day      string
	Start    time.Time
	End      time.Time
	Duration time.Duration
}

type Timetable struct {
	Periods []TimePeriod
	Gaps    []TimePeriod
}
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session-name")

		// Check if user is authenticated
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			// User is not authenticated, redirect to login page
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// User is authenticated, proceed to the next handler
		next.ServeHTTP(w, r)
	})
}

func init() {
	var err error
	db, err = sql.Open("sqlite", "./timetable.db")
	if err != nil {
		log.Fatal(err)
	}

	// Create users table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL
	)`)
	if err != nil {
		log.Fatal(err)
	}

	// Create timetables table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS timetables (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		day TEXT,
		name TEXT,
		start TEXT,
		end TEXT,
		duration TEXT,
		FOREIGN KEY (user_id) REFERENCES users(id)
	)`)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
http.HandleFunc("/markHoliday", markHolidayHandler)
	// Protected routes
	http.Handle("/", authMiddleware(http.HandlerFunc(renderForm)))
	http.Handle("/add", authMiddleware(http.HandlerFunc(addTimePeriod)))
	http.Handle("/fitActivities", authMiddleware(http.HandlerFunc(fitActivitiesHandler)))
	http.Handle("/uploadTimetable", authMiddleware(http.HandlerFunc(uploadTimetableHandler)))
    http.Handle("/cleardb", authMiddleware(http.HandlerFunc(clearTimetableHandler))) // Change to http.Handle

	http.Handle("/profile", authMiddleware(http.HandlerFunc(profileopen)))

var err error
	llm, err = ollama.New(ollama.WithModel("llama3.2"))
	if err != nil {
		log.Fatal(err)
	}

	// Start the HTTP server to handle user input and display goals
	http.HandleFunc("/achieve", handleGoalsPage)
	http.HandleFunc("/add-goal", handleAddGoal)
	fmt.Println("Server started at http://localhost:4080")
	log.Fatal(http.ListenAndServe(":4080", nil))
}



func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		_, err := db.Exec(`INSERT INTO users (username, password) VALUES (?, ?)`, username, password)
		if err != nil {
			http.Error(w, "Registration failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	http.ServeFile(w, r, "register.html")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		var userID int
		err := db.QueryRow(`SELECT id FROM users WHERE username = ? AND password = ?`, username, password).Scan(&userID)
		if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Set user as authenticated in session
		session, _ := store.Get(r, "session-name")
		session.Values["authenticated"] = true
		session.Values["userID"] = userID
		session.Save(r, w)

		// Load the timetable for the user
		loadTimetable(userID)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	http.ServeFile(w, r, "login.html")
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	session.Values["authenticated"] = false
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusFound)
}





func profileopen(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "index.html", http.StatusSeeOther)
}

func renderForm(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("index.html"))
	tmpl.Execute(w, timetable)
}

func addTimePeriod(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	day := r.FormValue("day")
	startStr := r.FormValue("start")
	endStr := r.FormValue("end")

	start, err := time.Parse("15:04", startStr)
	if err != nil {
		http.Error(w, "Invalid start time format", http.StatusBadRequest)
		return
	}
	end, err := time.Parse("15:04", endStr)
	if err != nil {
		http.Error(w, "Invalid end time format", http.StatusBadRequest)
		return
	}

	if start.After(end) {
		fmt.Fprintf(w, `<div class="error-message" hx-target="#error-dialog">Start time cannot be after end time</div>`)
		return
	}

	duration := end.Sub(start)

	newPeriod := TimePeriod{
		Name:     name,
		Day:      day,
		Start:    start,
		End:      end,
		Duration: duration,
	}

	// Check for overlap on the same day
	for _, period := range timetable.Periods {
		if period.Day == day && ((newPeriod.Start.Before(period.End) && newPeriod.End.After(period.Start)) ||
			(newPeriod.Start.Equal(period.Start) && newPeriod.End.Equal(period.End))) {
			fmt.Fprintf(w, `<div class="error-message" hx-target="#error-dialog">%s overlaps with %s on %s</div>`, newPeriod.Name, period.Name, day)
			return
		}
	}

	// Add the new time period
	timetable.Periods = append(timetable.Periods, newPeriod)

	// Sort the periods by day and then by start time
	sort.Slice(timetable.Periods, func(i, j int) bool {
		if dayOrder[timetable.Periods[i].Day] == dayOrder[timetable.Periods[j].Day] {
			return timetable.Periods[i].Start.Before(timetable.Periods[j].Start)
		}
		return dayOrder[timetable.Periods[i].Day] < dayOrder[timetable.Periods[j].Day]
	})

	// Calculate gaps between periods
	calculateGaps()

	// Assume userID is retrieved from the session after login
	userID := getUserIDFromSession(r)

	// Save the updated timetable to the database
	saveTimetable(userID)

	// Render the updated timetable and gaps
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func fitActivitiesHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("activityName")
	durationMinutes, err := strconv.Atoi(r.FormValue("duration"))
	if err != nil {
		http.Error(w, "Invalid duration", http.StatusBadRequest)
		return
	}
	duration := time.Duration(durationMinutes) * time.Minute
	day := r.FormValue("day")
	timePart := r.FormValue("partOfDay")

	activity := TimePeriod{
		Name:     name,
		Duration: duration,
	}

	if day == "Any" {
		// Try to fit the activity into the first available valid day
		for _, d := range days {
			activity.Day = d
			if fitActivityIntoGaps(activity, timePart) {
				break
			}
		}
	} else {
		activity.Day = day
		fitActivityIntoGaps(activity, timePart)
	}

	// Assume userID is retrieved from the session after login
	userID := getUserIDFromSession(r)

	// Save the updated timetable to the database
	saveTimetable(userID)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func fitActivityIntoGaps(activity TimePeriod, timePart string) bool {
	dayGaps := getDayGaps(activity.Day)

	// If there are no gaps for the day, initialize with wakeup to sleep gap
	if len(dayGaps) == 0 {
		initialGap := TimePeriod{
			Name:     "Gap",
			Day:      activity.Day,
			Start:    wakeupTime,
			End:      sleepTime,
			Duration: sleepTime.Sub(wakeupTime),
		}
		dayGaps = append(dayGaps, initialGap)
	}

	for i, gap := range dayGaps {
		if activity.Duration <= gap.Duration && (timePart == "Any" || matchTimePart(gap.Start, timePart)) {
			activity.Start = gap.Start
			activity.End = gap.Start.Add(activity.Duration)
			activity.Day = gap.Day

			// Add the activity to the timetable
			timetable.Periods = append(timetable.Periods, activity)

			// Update the gap start time and duration
			dayGaps[i].Start = activity.End
			dayGaps[i].Duration = dayGaps[i].End.Sub(dayGaps[i].Start)

			// Update the timetable with the remaining gaps for the day
			updateTimetableGaps(activity.Day, dayGaps)

			// Sort the timetable periods after adding the activity
			sort.Slice(timetable.Periods, func(i, j int) bool {
				if timetable.Periods[i].Day == timetable.Periods[j].Day {
					return timetable.Periods[i].Start.Before(timetable.Periods[j].Start)
				}
				return timetable.Periods[i].Day < timetable.Periods[j].Day
			})

			// Recalculate gaps after adding the activity
			calculateGaps()
			return true
		}
	}

	return false
}


func calculateGaps() {
	timetable.Gaps = []TimePeriod{}
	for _, day := range days {
		var previousEndTime = wakeupTime
		for _, period := range timetable.Periods {
			if period.Day == day {
				if period.Start.After(previousEndTime) {
					gap := TimePeriod{
						Name:     "Gap",
						Day:      day,
						Start:    previousEndTime,
						End:      period.Start,
						Duration: period.Start.Sub(previousEndTime),
					}
					timetable.Gaps = append(timetable.Gaps, gap)
				}
				previousEndTime = period.End
			}
		}

		// Add the gap from the last period to sleep time
		if previousEndTime.Before(sleepTime) {
			gap := TimePeriod{
				Name:     "Gap",
				Day:      day,
				Start:    previousEndTime,
				End:      sleepTime,
				Duration: sleepTime.Sub(previousEndTime),
			}
			timetable.Gaps = append(timetable.Gaps, gap)
		}
	}
}


func matchTimePart(t time.Time, part string) bool {
	switch part {
	case "Morning":
		return t.Hour() >= 6 && t.Hour() < 12
	case "Afternoon":
		return t.Hour() >= 12 && t.Hour() < 17
	case "Evening":
		return t.Hour() >= 17 && t.Hour() < 21
	case "Night":
		return t.Hour() >= 21 || t.Hour() < 6
	default:
		return true
	}
}

func updateTimetableGaps(day string, dayGaps []TimePeriod) {
	for _, gap := range dayGaps {
		if gap.Duration > 0 {
			timetable.Gaps = append(timetable.Gaps, gap)
		}
	}
}

func getDayGaps(day string) []TimePeriod {
	var dayGaps []TimePeriod
	for _, gap := range timetable.Gaps {
		if gap.Day == day {
			dayGaps = append(dayGaps, gap)
		}
	}
	return dayGaps
}

func saveTimetable(userID int) {
	// Delete existing timetable for the user
	_, err := db.Exec(`DELETE FROM timetables WHERE user_id = ?`, userID)
	if err != nil {
		log.Fatal(err)
	}

	// Insert the new timetable
	for _, period := range timetable.Periods {
		durationInSeconds := int(period.Duration.Seconds())
		_, err := db.Exec(`INSERT INTO timetables (user_id, day, name, start, end, duration) VALUES (?, ?, ?, ?, ?, ?)`,
			userID, period.Day, period.Name, period.Start.Format("15:04"), period.End.Format("15:04"), durationInSeconds)
		if err != nil {
			log.Fatal(err)
		}
	}
}


func loadTimetable(userID int) {
	rows, err := db.Query(`SELECT day, name, start, end, duration FROM timetables WHERE user_id = ?`, userID)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	timetable.Periods = []TimePeriod{}
	for rows.Next() {
		var day, name, startStr, endStr string
		var durationInSeconds int
		rows.Scan(&day, &name, &startStr, &endStr, &durationInSeconds)

		start, _ := time.Parse("15:04", startStr)
		end, _ := time.Parse("15:04", endStr)
		duration := time.Duration(durationInSeconds) * time.Second

		timetable.Periods = append(timetable.Periods, TimePeriod{
			Day:      day,
			Name:     name,
			Start:    start,
			End:      end,
			Duration: duration,
		})
	}

	calculateGaps() // Recalculate gaps after loading the timetable
}


func readTimetableFromCSV(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	return parseTimetableRecords(records)
}

func readTimetableFromExcel(filePath string) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return err
	}

	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil {
		return err
	}

	return parseTimetableRecords(rows)
}

func parseTimetableRecords(records [][]string) error {
	if len(records) < 2 {
		return fmt.Errorf("file does not contain enough data")
	}

	daysOfWeek := records[0][1:] // First row, ignoring the first cell (which is for time ranges)

	// Initialize lastEndTime for each day to the default wakeup time
	lastEndTime := make(map[string]time.Time)
	for _, day := range daysOfWeek {
		lastEndTime[day] = wakeupTime
	}

	for i, row := range records[1:] { // Skip the header row
		if len(row) < 2 {
			continue // Skip if the row doesn't have enough columns
		}

		timeRange := row[0] // First column is the time range
		timeParts := strings.Split(timeRange, " to ")
		if len(timeParts) != 2 {
			return fmt.Errorf("invalid time range format at row %d", i+2)
		}

		startTime, err := time.Parse("15:04", timeParts[0])
		if err != nil {
			return fmt.Errorf("invalid start time format at row %d: %v", i+2, err)
		}

		endTime, err := time.Parse("15:04", timeParts[1])
		if err != nil {
			return fmt.Errorf("invalid end time format at row %d: %v", i+2, err)
		}

		for j, day := range daysOfWeek {
			if j >= len(row)-1 {
				continue // Skip if there's no corresponding column for this day
			}

			name := row[j+1] // The period name for this day

			// Add a gap if the cell is empty
			if name == "" {
				if lastEndTime[day].Before(startTime) {
					gapDuration := startTime.Sub(lastEndTime[day])
					if gapDuration > 0 {
						gapPeriod := TimePeriod{
							Name:     "Gap",
							Day:      strings.Title(day),
							Start:    lastEndTime[day],
							End:      startTime,
							Duration: gapDuration,
						}
						timetable.Gaps = append(timetable.Gaps, gapPeriod)

						// Save the gap to the database
						_, err := db.Exec(`INSERT INTO timetables (user_id, day, name, start, end, duration) VALUES (?, ?, ?, ?, ?, ?)`,
							// Use a placeholder userID here, you'll need to determine how to get the userID
							1, // Placeholder userID, adjust this as necessary
							gapPeriod.Day,
							gapPeriod.Name,
							gapPeriod.Start.Format("15:04"),
							gapPeriod.End.Format("15:04"),
							int(gapPeriod.Duration.Seconds()),
						)
						if err != nil {
							return fmt.Errorf("error inserting gap into database: %v", err)
						}
					}
				}
				lastEndTime[day] = endTime
				continue // Skip adding an empty period, just handle the gap
			}

			// Add a gap if there's a period before the current time slot
			if lastEndTime[day].Before(startTime) {
				gapDuration := startTime.Sub(lastEndTime[day])
				if gapDuration > 0 {
					gapPeriod := TimePeriod{
						Name:     "Gap",
						Day:      strings.Title(day),
						Start:    lastEndTime[day],
						End:      startTime,
						Duration: gapDuration,
					}
					timetable.Gaps = append(timetable.Gaps, gapPeriod)

					// Save the gap to the database
					_, err := db.Exec(`INSERT INTO timetables (user_id, day, name, start, end, duration) VALUES (?, ?, ?, ?, ?, ?)`,
						// Use a placeholder userID here, you'll need to determine how to get the userID
						1, // Placeholder userID, adjust this as necessary
						gapPeriod.Day,
						gapPeriod.Name,
						gapPeriod.Start.Format("15:04"),
						gapPeriod.End.Format("15:04"),
						int(gapPeriod.Duration.Seconds()),
					)
					if err != nil {
						return fmt.Errorf("error inserting gap into database: %v", err)
					}
				}
			}

			newPeriod := TimePeriod{
				Name:     name,
				Day:      strings.Title(day),
				Start:    startTime,
				End:      endTime,
				Duration: endTime.Sub(startTime),
			}

			// Add the new period to the timetable
			timetable.Periods = append(timetable.Periods, newPeriod)

			// Save the period to the database
			_, err := db.Exec(`INSERT INTO timetables (user_id, day, name, start, end, duration) VALUES (?, ?, ?, ?, ?, ?)`,
				// Use a placeholder userID here, you'll need to determine how to get the userID
				1, // Placeholder userID, adjust this as necessary
				newPeriod.Day,
				newPeriod.Name,
				newPeriod.Start.Format("15:04"),
				newPeriod.End.Format("15:04"),
				int(newPeriod.Duration.Seconds()),
			)
			if err != nil {
				return fmt.Errorf("error inserting period into database: %v", err)
			}

			lastEndTime[day] = endTime
		}
	}

	// Add gaps after the last period of the day
	for _, day := range daysOfWeek {
		if lastEndTime[day].Before(sleepTime) {
			gapDuration := sleepTime.Sub(lastEndTime[day])
			if gapDuration > 0 {
				gapPeriod := TimePeriod{
					Name:     "Gap",
					Day:      day,
					Start:    lastEndTime[day],
					End:      sleepTime,
					Duration: gapDuration,
				}
				timetable.Gaps = append(timetable.Gaps, gapPeriod)

				// Save the gap to the database
				_, err := db.Exec(`INSERT INTO timetables (user_id, day, name, start, end, duration) VALUES (?, ?, ?, ?, ?, ?)`,
					// Use a placeholder userID here, you'll need to determine how to get the userID
					1, // Placeholder userID, adjust this as necessary
					gapPeriod.Day,
					gapPeriod.Name,
					gapPeriod.Start.Format("15:04"),
					gapPeriod.End.Format("15:04"),
					int(gapPeriod.Duration.Seconds()),
				)
				if err != nil {
					return fmt.Errorf("error inserting gap into database: %v", err)
				}
			}
		}
	}

	// Sort the periods and gaps by day of the week and start time
	sort.SliceStable(timetable.Periods, func(i, j int) bool {
		if dayOrder[timetable.Periods[i].Day] != dayOrder[timetable.Periods[j].Day] {
			return dayOrder[timetable.Periods[i].Day] < dayOrder[timetable.Periods[j].Day]
		}
		return timetable.Periods[i].Start.Before(timetable.Periods[j].Start)
	})

	sort.SliceStable(timetable.Gaps, func(i, j int) bool {
		if dayOrder[timetable.Gaps[i].Day] != dayOrder[timetable.Gaps[j].Day] {
			return dayOrder[timetable.Gaps[i].Day] < dayOrder[timetable.Gaps[j].Day]
		}
		return timetable.Gaps[i].Start.Before(timetable.Gaps[j].Start)
	})

	return nil
}

func uploadTimetableHandler(w http.ResponseWriter, r *http.Request) {
	// Assume the file is uploaded as a form file
	file, header, err := r.FormFile("timetableImage")
	if err != nil {
		http.Error(w, "Failed to upload file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Get the file extension
	ext := filepath.Ext(header.Filename)
	if ext != ".csv" && ext != ".xlsx" {
		http.Error(w, "Invalid file type", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join("./uploads", header.Filename)

	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	// Save the file locally
	out, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Process the file based on its extension
	if ext == ".csv" {
		if err := readTimetableFromCSV(filePath); err != nil {
			http.Error(w, "Failed to read CSV file", http.StatusInternalServerError)
			return
		}
	} else if ext == ".xlsx" {
		if err := readTimetableFromExcel(filePath); err != nil {
			http.Error(w, "Failed to read Excel file", http.StatusInternalServerError)
			return
		}
	}
	calculateGaps()

	// Redirect to the timetable page or wherever you want to display the result
	http.Redirect(w, r, "/", http.StatusSeeOther)
}



var dayOrder = map[string]int{
	"Monday":    0,
	"Tuesday":   1,
	"Wednesday": 2,
	"Thursday":  3,
	"Friday":    4,
	"Saturday":  5,
	"Sunday":    6,
}


