package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/anaskhan96/soup"
	"github.com/reteps/gopowerschool"
	//"gopkg.in/Iwark/spreadsheet.v2"
	//"image"
	"image/jpeg"
	//"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	http.HandleFunc("/getPowerschoolPhoto", getPowerschoolPhoto)
	http.HandleFunc("/getPowerschoolInfo", getPowerschoolInfo)
	http.HandleFunc("/getMilesplitAthlete", getAthlete)
	http.Handle("/", http.FileServer(http.Dir("./home")))
	if err := http.ListenAndServe(":8000", nil); err != nil {
		panic(err)
	}
}

type Results struct {
	Name    string
	School  string
	Class   string
	City    string
	Records map[string]string
	Events  map[string][]Event
}

type Event struct {
	Time  string
	Place int
	Meet  string
	Date  string
}

func getAthlete(rw http.ResponseWriter, req *http.Request) {
	start := time.Now()
	if err := req.ParseForm(); err != nil {
		fmt.Fprintf(rw, "Hello, POT method. ParseForm() err: %v", err)
		return
	}
	resp, err := soup.Get(fmt.Sprintf("http://nc.milesplit.com/search?q=%s&category=athlete", req.FormValue("name")))
	if err != nil {
		fmt.Fprintf(rw, "error")
		return
	}
	first_result := soup.HTMLParse(resp).Find("ul", "class", "search-results")
	if first_result.Error != nil {
		fmt.Fprintf(rw, "Could not find that athlete")
		return
	}
	id := strings.Split(first_result.Find("li").Find("a").Attrs()["href"], "/")[2]
	fmt.Println("Time to find id: ", time.Since(start))
	resp2, err := soup.Get(fmt.Sprintf("http://milesplit.com/athletes/pro/%s/stats", id))
	if err != nil {
		fmt.Fprintf(rw, "error")
		return
	}
	document := soup.HTMLParse(resp2)
	result := &Results{}
	result.Records = map[string]string{}
	result.Events = map[string][]Event{}

	result.Name = document.Find("span", "class", "fname").Text() + " " + document.Find("span", "class", "lname").Text()
	result.School = strings.TrimSpace(document.Find("div", "class", "team").Find("a").Text())
	result.Class = strings.TrimSpace(strings.Split(document.Find("span", "class", "grade").Text(), " of ")[1])
	result.City = strings.TrimSpace(document.Find("span", "class", "city").Text())
	for _, pr := range document.Find("div", "class", "bests").Find("ul").FindAll("li") {
		record := strings.Split(strings.TrimSpace(pr.Text()), " - ")
		result.Records[record[0]] = record[1]

	}
	fmt.Println("Time to find all other info: ", time.Since(start))
	var currentEvent string
	for _, event := range document.FindAll("tr") {
		class := event.Attrs()["class"]
		if class == "thead" {
			currentEvent = strings.TrimSpace(event.Find("th", "class", "event").Text())
		}
		if class == "" || class == "pr" {
			resp_event, err := soup.Get("http://milesplit.com" + event.FindAll("a")[1].Attrs()["href"])
			if err != nil {
				fmt.Fprintf(rw, "error")
				return
			}
			event_doc := soup.HTMLParse(resp_event)
			time := strings.TrimSpace(event_doc.Find("div", "class", "field mark").Find("span").Text())
			place := strings.TrimSpace(event_doc.Find("div", "class", "field place").Find("span").Text())
			int_place, _ := strconv.Atoi(place[:len(place)-2])
			meet := strings.TrimSpace(event_doc.Find("div", "class", "field meet").Find("a").Text())
			date := strings.Replace(strings.TrimSpace(event_doc.Find("div", "class", "field date").Find("span").Text()), "                        ", " ", -1)
			result.Events[currentEvent] = append(result.Events[currentEvent], Event{time, int_place, meet, date})
		}
	}
	fmt.Println("Time to find events: ", time.Since(start))
	json, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(rw, "json error")
		return
	}
	fmt.Fprintf(rw, string(json))

}

/*
func handleName(rw http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" && req.Method != "GET" {
		fmt.Fprintf(rw, "Post or get plz")
		return
	}
	if err := req.ParseForm(); err != nil {
		fmt.Fprintf(rw, "Hello, POT method. ParseForm() err: %v", err)
		return
	}
	spreadsheetName := strings.Title(req.FormValue("lname")) + ", " + strings.Title(req.FormValue("fname"))
	service, err := spreadsheet.NewService()
	if err != nil {
		panic(err)
	}
	spreadsheet, err := service.FetchSpreadsheet("1vOErQNWFTNmo42IKYCw11w3Y4QbBKzLKXpZelqF1RDc")
	if err != nil {
		panic(err)
	}
	sheet, err := spreadsheet.SheetByIndex(0)
	if err != nil {
		panic(err)
	}
	for _, row := range sheet.Rows {
		if row[3].Value == spreadsheetName {
			fmt.Fprintf(rw, strings.Split(row[0].Value, "@")[0])
			return
		}
	}
}
*/
func getPowerschoolInfo(rw http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" && req.Method != "GET" {
		http.Error(rw, "Invalid request method only POST and GET allowed.", 405)
		return
	}
	if err := req.ParseForm(); err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	if req.FormValue("base_url") == "" || req.FormValue("username") == "" || req.FormValue("password") == "" {
		http.Error(rw, "{\"error\":\"Needs parameters base_url, username, and password\"}", 500)
		return
	}
	client := gopowerschool.Client(req.FormValue("base_url"))
	student, err := client.GetStudent(req.FormValue("username"), req.FormValue("password"))
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}
	json, err := json.Marshal(student)
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}
	rw.Write(json)
}
func getPowerschoolPhoto(rw http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" && req.Method != "GET" {
		http.Error(rw, "Invalid request method only POST and GET allowed.", 405)
		return
	}
	if err := req.ParseForm(); err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}
	if req.FormValue("base_url") == "" || req.FormValue("username") == "" || req.FormValue("password") == "" {
		http.Error(rw, "Needs parameters base_url, username, and password", 500)
		return
	}
	client := gopowerschool.Client(req.FormValue("base_url"))
	session, userID, err := client.CreateUserSessionAndStudent(req.FormValue("username"), req.FormValue("password"))
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}
	response, err := client.GetStudentPhoto(&gopowerschool.GetStudentPhoto{UserSessionVO: session, StudentID: userID})
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}
	decoded, err := base64.StdEncoding.DecodeString(string(response.Return_))
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}
	img, err := jpeg.Decode(bytes.NewReader(decoded))
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}
	buffer := new(bytes.Buffer)
	if err := jpeg.Encode(buffer, img, &jpeg.Options{100}); err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}

	rw.Header().Set("Content-Type", "image/jpeg")
	rw.Header().Set("Content-Length", strconv.Itoa(len(buffer.Bytes())))
	if _, err := rw.Write(buffer.Bytes()); err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}

}
