

# DailyGrid

A web application designed to help users manage their daily schedules, track progress toward goals, and optimize time with a detailed timetable. The app integrates goal setting, daily task management, and timetable organization in one place.

## Features

- **Goal Tracking**: Define long-term goals and break them down into daily tasks.
- **Timetable Management**: Create, edit, and organize your daily timetable with time slots for various activities.
- **Task Generation**: Automatically generate daily tasks based on your goals.
- **Custom Tasks**: Add personal tasks to each day’s schedule.
- **Gap Management**: Automatically find gaps in your timetable and suggest activities.
- **File Upload**: Import timetables, goals, and tasks via CSV or Excel files.
- **Real-Time Updates**: Display current tasks or activities in real-time using HTMX.
- **Progress Visualization**: Track progress using visual graphs and a calendar view.
- **Alerts & Backlog**: Receive alerts for uncompleted tasks and manage them through a backlog system.
- **User Authentication**: Register and log in to manage your personalized timetable and goals.

## Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/your-username/goals-timetable-manager.git
   cd goals-timetable-manager
   ```

2. Install dependencies:

   - **Backend (Go)**:
     Ensure Go is installed. Run:
     ```bash
     go mod download
     ```

3. Set up environment variables:
   ```bash
   export LLM_API_KEY=your_llm_api_key
   export DATABASE_URL=your_database_url
   ```

4. Run the app:
   ```bash
   go run main.go
   ```

## Usage

1. **Create Goals & Tasks**: Define your goals and assign daily tasks.
2. **Timetable Creation**: Organize your daily schedule with time slots.
3. **Track Progress**: View progress using graphs and calendars.
4. **Upload Files**: Import goals or timetables using CSV or Excel.

THIS IS A WORK IN PROGRESS, THE ABOVE MENTIONED FEATURES WILL BE IMPLEMENTED WITH TIME.
