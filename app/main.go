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
    "os/exec"
	"github.com/xuri/excelize/v2"
	"io"
	"os"
	"path/filepath"
	"strings"
)
func runpdf(pdfPath, excelPath string) error {
	// Define the command to run the Python script with arguments
	cmd := exec.Command("python3", "pdf-to-excel.py", pdfPath, excelPath)

	// Run the command and capture its output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error running Python script: %v, output: %s", err, string(output))
	}
	return readTimetableFromExcel(excelPath)
	
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

	// Protected routes
	http.Handle("/", authMiddleware(http.HandlerFunc(renderForm)))
	http.Handle("/add", authMiddleware(http.HandlerFunc(addTimePeriod)))
	http.Handle("/fitActivities", authMiddleware(http.HandlerFunc(fitActivitiesHandler)))
	http.Handle("/uploadTimetable", authMiddleware(http.HandlerFunc(uploadTimetableHandler)))
    http.Handle("/cleardb", authMiddleware(http.HandlerFunc(clearTimetableHandler))) // Change to http.Handle

	http.Handle("/profile", authMiddleware(http.HandlerFunc(profileopen)))

	http.ListenAndServe(":8080", nil)
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



func showtimetable(w http.ResponseWriter, r *http.Request) {
	// Handle timetable upload
}

func profileopen(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "profile.html", http.StatusSeeOther)
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
	timePart := r.FormValue("timePart")

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
	}else if ext == ".pdf"{
		if err := runpdf(filePath,"demo.xlsx"); err!=nil{
			http.Error(w, "Failed to read pdf file", http.StatusInternalServerError)
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

