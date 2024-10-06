
---

# Timetable Management App

A web application designed to help users manage their daily schedules efficiently. The app provides functionality to add, edit, and organize periods, extracurricular activities, and manage gaps between events. Additionally, users can upload timetables as images, which are parsed into structured data.

## Features

- **Dynamic Timetable Creation**: Add, edit, and manage periods with overlapping detection.
- **Gap Management**: Automatically find and fill gaps between scheduled activities.
- **Time Validation**: Enforce valid start and end times, with a 24-hour format for precise scheduling.
- **Image Upload and OCR**: Upload timetable images, which are parsed into CSV format using Optical Character Recognition (OCR).
- **Excel/CSV Parsing**: Import timetables from Excel or CSV files, with periods formatted as `HH:MM to HH:MM`.
- **Real-Time Updates**: Display the current period dynamically using HTMX for a seamless user experience.
- **Wake-up and Sleep Time Management**: Customize daily schedules with wake-up and sleep time tracking.
- **User Authentication**: Register and log in to manage personal schedules.

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [API Endpoints](#api-endpoints)
- [File Upload Support](#file-upload-support)
- [Excel/CSV Parsing](#excelcsv-parsing)
- [OCR Integration](#ocr-integration)
- [Contributing](#contributing)
- [License](#license)

## Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/your-username/timetable-manager.git
   cd timetable-manager
   ```

2. Install dependencies:

   - **Backend (Go)**:
   
     Ensure Go is installed. Then, from the root of the project, run:

     ```bash
     go mod download
     ```

   - **Frontend**:
   
     The front end uses basic HTML, CSS, and HTMX for dynamic content loading, so no additional dependencies are required.

3. Set up the environment variables:

   ```bash
   export OCR_API_KEY=your_ocr_api_key
   export DATABASE_URL=your_database_url
   ```

4. Run the app:

   ```bash
   go run main.go
   ```

## Usage

1. **Timetable Creation**: 
   - Add periods through the UI. Enter the start and end times using the dropdown (in 24-hour format).
   - Ensure no overlap between periods. An error will be shown if overlapping times are detected.

2. **Upload Timetable as Image**(Experimental): 
   - Use the "Upload Timetable" button to upload an image file (JPEG/PNG). The app will parse the image into structured data using OCR.

3. **View Current Period**: 
   - The current period will be displayed dynamically on the top right of the screen, updated in real-time as the day progresses.

4. **Import from CSV/Excel**:
   - Upload a CSV or Excel file with the timetable in the format `HH:MM to HH:MM` for each day. The periods will automatically be imported into the app.

5. **Sleep and Wake Time Settings**:
   - Adjust your wake-up and sleep times to customize the available time for activities.

## API Endpoints

- `POST /api/timetable/upload` – Upload a timetable image.
- `GET /api/timetable/gaps` – Retrieve gaps between scheduled periods.
- `POST /api/user/login` – User login.
- `POST /api/user/register` – User registration.

## File Upload Support

- **Timetable Images**: Users can upload timetable images, which are processed using OCR to extract data and update the timetable.
- **Accepted formats**: JPEG, PNG.

## Excel/CSV Parsing

To upload timetables via CSV or Excel:

- **Format**:
  - Top row contains days of the week.
  - Left column contains timings (formatted as `HH:MM to HH:MM`).
  - Example:

    | Time         | Monday        | Tuesday      | ... |
    |--------------|---------------|--------------|-----|
    | 08:00 to 09:00 | Math          | Physics      | ... |
    | 09:00 to 10:00 | Chemistry     | Math         | ... |

## OCR Integration

This app uses OCR (Optical Character Recognition) to parse timetable images and convert them into structured periods. To enable this feature, ensure that the `OCR_API_KEY` is set in your environment variables.

- **API Key Setup**: Set up your OCR API key in the `.env` file or as an environment variable:

   ```bash
   export OCR_API_KEY=your_ocr_api_key
   ```

## Contributing

We welcome contributions from the community! If you'd like to contribute:

1. Fork the repository.
2. Create a feature branch (`git checkout -b feature-branch`).
3. Commit your changes (`git commit -m 'Add new feature'`).
4. Push to the branch (`git push origin feature-branch`).
5. Open a pull request.

Please ensure your code follows the existing style and conventions.

---

