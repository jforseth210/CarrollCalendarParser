# Carroll Calendar Parser  
Parse an ICS file from [https://www.carroll.edu/news-events/events](https://www.carroll.edu/news-events/events)

## Installation
You must have git and [go](https://go.dev/doc/install) installed
```bash
git clone https://github.com/jforseth210/CarrollCalendarParser
cd CarrollCalendarParser
```

## Usage
```
go run ./main.go 2024-03 2024-03
```
Open the generated ics file in the calendar program of your choosing. 

Please run this program sparingly, as it makes a lot of http requests, especially if loading several months. 
