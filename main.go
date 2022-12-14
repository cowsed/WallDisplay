package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/apognu/gocal"
)

var calendar_filename = "cal.ics"
var current_weather Weather

var today_events []gocal.Event
var tomorrow_events []gocal.Event
var overmorrow_events []gocal.Event

type Data struct {
	RightNow                  time.Time
	FirstEvents, SecondEvents []gocal.Event
	W                         *BasicWeather
	FirstLabel, SecondLabel   string
}

func handleFront(w http.ResponseWriter, r *http.Request) {
	err := frontTempl.Execute(w, frontTempl)
	if err != nil {
		log.Println(err)
	}
}
func doCalendar() {
	for {
		// open calendar
		f, err := os.Open(calendar_filename)
		check(err)

		// read calendar
		bs, _ := io.ReadAll(f)
		var calendarReader io.ReadSeeker = bytes.NewReader(bs)
		f.Close()

		rightNow := time.Now()
		startOfToday := time.Date(rightNow.Year(), rightNow.Month(), rightNow.Day(), 0, 0, 0, 0, rightNow.Location())
		endOfToday := startOfToday.AddDate(0, 0, 1)
		endOfTomorrow := endOfToday.AddDate(0, 0, 1)
		endOfOvermorrow := endOfTomorrow.AddDate(0, 0, 1)

		today_events = GetEventsInTimeFrame(rightNow, endOfToday, calendarReader)
		tomorrow_events = GetEventsInTimeFrame(endOfToday, endOfTomorrow, calendarReader)
		overmorrow_events = GetEventsInTimeFrame(endOfTomorrow, endOfOvermorrow, calendarReader)

		log.Println("Updated Calendar")
		time.Sleep(5 * time.Minute)
	}
}
func handleInside(w http.ResponseWriter, r *http.Request) {
	var rolledOver = false
	rightNow := time.Now()
	var trimmed_events_today []gocal.Event
	for i := range today_events {
		if today_events[i].Start.After(rightNow) {
			trimmed_events_today = append(trimmed_events_today, today_events[i])
		}
	}

	if len(trimmed_events_today) == 0 {
		rolledOver = true

	}

	//all the data to be shown
	data := Data{
		RightNow:     time.Now(),
		FirstEvents:  trimmed_events_today,
		SecondEvents: tomorrow_events,
		W:            &myw,
		FirstLabel:   "Today",
		SecondLabel:  "Tomorrow",
	}
	if rolledOver {
		data.FirstLabel = "Tomorrow"
		data.SecondLabel = "Overmorrow"
		data.FirstEvents = tomorrow_events
		data.SecondEvents = overmorrow_events
	}

	//write to asker
	err := mainTempl.Execute(w, data)
	if err != nil {
		log.Println(err)
	}
	//we done
}

var mainTempl *template.Template
var frontTempl *template.Template

const portNum = 8080

func main() {
	time.Local = time.FixedZone("America/New_York", -4*60*60)

	//read template 1
	f2, err := os.Open("main.html")
	check(err)
	bs2, _ := io.ReadAll(f2)
	tmpl := template.New("walldisplay")
	tmpl, _ = tmpl.Parse(string(bs2))
	mainTempl = tmpl

	//read template 2
	f3, err := os.Open("front.html")
	check(err)
	bs3, _ := io.ReadAll(f3)
	tmpl2 := template.New("wallfront")
	tmpl2, _ = tmpl2.Parse(string(bs3))
	frontTempl = tmpl2

	//take care of weather
	go doWeather()
	go doCalendar()

	//handle requests
	http.HandleFunc("/inside", http.HandlerFunc(handleInside))
	http.HandleFunc("/", http.HandlerFunc(handleFront))

	log.Printf("Starting server at localhost:%d\n", portNum)
	http.ListenAndServe(fmt.Sprintf(":%d", portNum), nil)
}

func GetEventsInTimeFrame(start, end time.Time, rs io.ReadSeeker) []gocal.Event {
	calendar := gocal.NewParser(rs)
	calendar.Start, calendar.End = &start, &end
	calendar.Parse()
	rs.Seek(0, io.SeekStart)

	events := []gocal.Event{}
	for _, e := range calendar.Events {

		events = append(events, e)
	}
	return events
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
