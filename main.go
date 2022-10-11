package main

import (
	"bytes"
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

type Data struct {
	RightNow                    time.Time
	EventsToday, EventsTomorrow []gocal.Event
	W                           *BasicWeather
	FirstLabel, SecondLabel     string
}

func handleFront(w http.ResponseWriter, r *http.Request) {
	err := frontTempl.Execute(w, frontTempl)
	if err != nil {
		log.Println(err)
	}
}

func handleInside(w http.ResponseWriter, r *http.Request) {
	var rolledOver = false
	rightNow := time.Now()
	endOfToday := time.Date(rightNow.Year(), rightNow.Month(), rightNow.Day(), 0, 0, 0, 0, rightNow.Location()).AddDate(0, 0, 1)

	//get all events today
	eventsToday := GetEventsInTimeFrame(rightNow, endOfToday, calendarReader)
	if len(eventsToday) == 0 {
		//dont show
		rolledOver = true
		rightNow = endOfToday.Add(time.Second)
		endOfToday = time.Date(rightNow.Year(), rightNow.Month(), rightNow.Day(), 0, 0, 0, 0, rightNow.Location()).AddDate(0, 0, 1)

	}

	endOfTomorrow := endOfToday.AddDate(0, 0, 1)

	//get all events today
	eventsToday = GetEventsInTimeFrame(rightNow, endOfToday, calendarReader)
	if len(eventsToday) == 0 {
		log.Println("SOMETHING VERY BAD HAS HAPPENED")
	}
	//get all events tomorrow
	eventsTomorrow := GetEventsInTimeFrame(endOfToday, endOfTomorrow, calendarReader)

	//all the data to be shown
	data := Data{
		RightNow:       time.Now(),
		EventsToday:    eventsToday,
		EventsTomorrow: eventsTomorrow,
		W:              &myw,
		FirstLabel:     "Today",
		SecondLabel:    "Tomorrow",
	}
	if rolledOver {
		data.FirstLabel = "Tomorrow"
		data.SecondLabel = "Overmorrow"
	}

	//write to asker
	err := mainTempl.Execute(w, data)
	if err != nil {
		log.Println(err)
	}
	//we done
}

var calendarReader io.ReadSeeker
var mainTempl *template.Template
var frontTempl *template.Template

func main() {
	//open calendar
	f, err := os.Open(calendar_filename)
	check(err)

	//read calendar
	bs, _ := io.ReadAll(f)
	var rs io.ReadSeeker = bytes.NewReader(bs)
	defer f.Close()
	calendarReader = rs

	//read template 1
	f2, err := os.Open("main.html")
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

	//handle requests
	http.HandleFunc("/inside", http.HandlerFunc(handleInside))
	http.HandleFunc("/", http.HandlerFunc(handleFront))
	http.ListenAndServe(":8080", nil)
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
