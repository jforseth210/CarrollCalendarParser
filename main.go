package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
	ics "github.com/arran4/golang-ical"
	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Author: Justin Forseth 
// Downloads html from Carroll College calendar and
// parses it into an ics file 

func main() {
	// Validate argument length
	if len(os.Args) != 3 {
		println("Please enter the start and end months in the format YYYY-MM")
		os.Exit(1)
	}

	start := os.Args[1]
	end := os.Args[2]
	// Make a list of month strings
	months, err := generateMonthList(start, end)

	if err != nil {
		println("Please enter the start and end months in the format YYYY-MM")
		os.Exit(1)
	}

	// Set up a calendar
	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodRequest)

	// Save calendar if process is interrupted
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		// Don't overwrite an existing complete calendar if it exists
		os.WriteFile("carroll.ics.part", []byte(cal.Serialize()), 0655)
		println("Program interrupted, saved incomplete calendar as carroll.ics.part")
		os.Exit(1)
	}()

	for _, month := range months {
		// Load the month's calendar page
		doc, err := loadPage("http://www.carroll.edu/news-events/events/" + month)
		if err != nil {
			log.Fatal(err)
		}
		// Find all the links to events
		links := getEventLinksFromHTML(doc)

		for _, link := range links {
			println("Loading " + link)
			// Load the event page
			doc, err := loadPage(link)
			if err != nil {
				log.Println("Error loading " + link)
				continue
			}

			// Find the title of the event
			title := parseTitle(doc)

			// Find the start time of the event
			startTime, err := parseStartTime(doc)
			if err != nil {
				log.Println("Failed to find a start time for " + title)
				continue
			}

			// Find the end time of the event
			endTime, err := parseEndTime(doc)
			if err != nil {
				log.Println("Failed to find an end time for " + title)
				continue
			}

			// Find the event location
			location := parseLocation(doc)

			// Find the event description
			description := parseDescription(doc)

			// Create an ics.VEvent with the data provided
			addEvent(cal, title, *startTime, *endTime, location, description, link)
		}
	}

	// Write out all the events to a file
	os.WriteFile("carroll.ics", []byte(cal.Serialize()), 0755)
}

// Loads a page from a URL and parses the HTML
func loadPage(url string) (*goquery.Document, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

// Get the links to events from a Carroll calendar page
func getEventLinksFromHTML(doc *goquery.Document) []string {
	// Find all the anchor tags in the month table
	anchorTags := doc.Find("table a")

	var links []string

	// Iterate through the event links
	anchorTags.Each(func(i int, anchorTag *goquery.Selection) {
		// Get the href attribute of the anchor tag
		relativeLink, exists := anchorTag.Attr("href")
		if !exists {
			return
		}

		// Make sure it's a link to a carroll event
		if !strings.Contains(relativeLink, "/news-events/events") {
			return
		}

		// Make sure it hasn't been added already
		// (There are multiple links to multi-day events)
		for _, existingLink := range links {
			if existingLink == "https://www.carroll.edu"+relativeLink {
				return
			}
		}

		// Add the link to the list
		links = append(links, "https://www.carroll.edu"+relativeLink)
	})
	// Return the list of links
	return links
}

// Get a title from from a Carroll event page
func parseTitle(doc *goquery.Document) string {
	// Get the title
	title := doc.Find(".hero__title").Text()
	// Get rid of all caps
	title = cases.Title(language.English).String(title)
	// Get rid of extra whitespace
	title = strings.TrimSpace(title)
	return title
}

// Get the start time from a Carroll event page
func parseStartTime(doc *goquery.Document) (*time.Time, error) {
	// Find the date area
	dates := doc.Find(".event__date").Find("time")

	// Find the start date
	var unixString string
	var exists bool
	dates.Each(func(i int, date *goquery.Selection) {
		if i == 0 {
			unixString, exists = date.Attr("datetime")
		}
	})
	// Parse
	if !exists {
		return nil, errors.New("No start time found")
	}
	dateInt, err := strconv.ParseInt(unixString, 10, 64)
	if err != nil {
		return nil, err
	}
	tm := time.Unix(dateInt, 0).Local()
	return &tm, nil
}
func parseEndTime(doc *goquery.Document) (*time.Time, error) {
	// Find the date area
	dates := doc.Find(".event__date").Find("time")

	//Find the end date
	var unixString string
	var exists bool
	dates.Each(func(i int, date *goquery.Selection) {
		unixString, exists = date.Attr("datetime")
	})

	if !exists {
		return nil, errors.New("No end time found")
	}
	dateInt, err := strconv.ParseInt(unixString, 10, 64)
	if err != nil {
		return nil, err
	}
	tm := time.Unix(dateInt, 0).Local()
	return &tm, nil
}
func parseLocation(doc *goquery.Document) string {
	location := doc.Find(".event__location").Text()
	location = strings.Replace(location, "Campus", "Campus\n", -1)
	location = strings.TrimSpace(location)
	return location
}
func parseDescription(doc *goquery.Document) string {
	description := doc.Find(".text-content").Children().First().Text()
	description = strings.TrimSpace(description)
	return description

}

// Create an ics.VEvent with the data provided
func addEvent(cal *ics.Calendar, summary string, start time.Time, end time.Time, location string, description string, url string) *ics.VEvent {
	event := cal.AddEvent(uuid.NewString())
	event.SetCreatedTime(time.Now())
	event.SetDtStampTime(time.Now())
	event.SetModifiedAt(time.Now())
	event.SetStartAt(start)
	event.SetEndAt(end)
	event.SetSummary(summary)
	event.SetLocation(location)
	event.SetDescription(description)
	event.SetURL(url)
	return event
}

// This function generates a list of month strings from the start and end dates
// It's AI generated and works by adding a month to the start month until it's
// greater than the end date.
func generateMonthList(start, end string) ([]string, error) {
	var monthList []string

	// Convert start and end dates to time.Time
	startDate, err := time.Parse("2006-01", start)
	if err != nil {
		return nil, err
	}
	endDate, err := time.Parse("2006-01", end)
	if err != nil {
		return nil, err
	}

	// Generate month list
	for !startDate.After(endDate) {
		monthList = append(monthList, startDate.Format("200601"))
		startDate = startDate.AddDate(0, 1, 0)
	}

	return monthList, nil
}
