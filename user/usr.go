package user

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	_ "github.com/lib/pq"
	"github.com/tokopedia/sqlt"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
	"html/template"
)

type User struct {
	UserID     int64     `json:"user_id" db:"user_id"`
	FullName   string    `json:"full_name" db:"full_name"`
	Email      string    `json:"email"db:"user_email"`
	BirthDate  time.Time `json:"birth_date"db:"birth_date"`
	CreateTime time.Time `json:"create_time"db:"create_time"`
	UpdateTime time.Time `json:"update_time"db:"update_time"`
	MSISDN     string    `json:"msisdn"db:"msisdn"`
	Age        int       `json:"age"`
}

type ResponseHeader struct {
	Data      interface{} `json:"data"`
	TotalData int         `json:"total_data"`
}

type Counter struct {
	Count int `json:"counter"`
}

var queryGetUser = `SELECT 
		user_id,
		user_email, 
		full_name, 
		msisdn, 
		birth_date, 
		create_time, 
		update_time 
	FROM ws_user %search% ORDER BY user_id DESC LIMIT $1 OFFSET $2`

func (r *ResponseHeader) Render(w http.ResponseWriter) {
	s := reflect.ValueOf(r.Data)
	if s.Kind() == reflect.Slice {
		r.TotalData = s.Len()
	}

	js, err := json.Marshal(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(js)
}

func HandlerUserHtmlPage(w http.ResponseWriter, r *http.Request) {
	var t = template.New("index.html").Delims("{[{", "}]}")
	err := template.Must(t.ParseFiles("index.html")).ExecuteTemplate(w, "index.html", r)
	if err != nil {
		log.Println("execute failed : ", err)
	}

}

func HandlerGetUserList(w http.ResponseWriter, r *http.Request) {
	var res ResponseHeader

	search := r.FormValue("search")
	limit := r.FormValue("limit")
	offset := r.FormValue("offset")

	intLimit, err := strconv.ParseInt(limit, 10, 64)
	if err != nil {
		intLimit = 10
	}
	intOffset, err := strconv.ParseInt(offset, 10, 64)
	if err != nil {
		intOffset = 0
	}

	data, _ := getUserList(search, int(intLimit), int(intOffset))

	res.Data = data

	res.Render(w)
}


func HandlerPingCounter(w http.ResponseWriter, r *http.Request) {
	var res ResponseHeader
	handler := r.FormValue("handler")
	action := r.FormValue("c")

	if action == "1" {
		c := AddCounter(handler)
		res.Data = c
	} else {
		c := ReadCounter(handler)
		res.Data = c
	}

	res.Render(w)
}

func AddCounter(handler string) Counter {
	red := redis.NewClient(&redis.Options{
		Addr:     "devel-redis.tkpd:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	defer red.Close()

	counter := Counter{}

	readCounter, err := red.HGet("counter:"+handler, "check").Result()
	if err != nil {
		counter = Counter{
			Count: 0,
		}
	} else {
		_ = json.Unmarshal([]byte(readCounter), &counter)
	}

	counter.Count += 1
	cJson, err := json.Marshal(counter)
	err = red.HSet("counter:" + handler, "check", cJson).Err()
	if err != nil  {
		log.Println("Error Set Redis : ", err)
	}

	return counter
}

func ReadCounter(handler string) Counter {
	red := redis.NewClient(&redis.Options{
		Addr:     "devel-redis.tkpd:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	defer red.Close()

	counter := Counter{}

	readCounter, err := red.HGet("counter:"+handler, "check").Result()
	if err != nil {
		counter = Counter{
			Count: 0,
		}
	} else {
		_ = json.Unmarshal([]byte(readCounter), &counter)
	}

	return counter
}


func getUserList(search string, limit int, offset int) (users []User, err error) {
	Master := "postgres://ep161101:k4ngDadangToped@devel-postgre.tkpd/tokopedia-dev-db?sslmode=disable"
	Slave := "postgres://ep161101:k4ngDadangToped@devel-postgre.tkpd/tokopedia-dev-db?sslmode=disable"
	Conn := Master + ";" + Slave

	db, err := sqlt.Open("postgres", Conn)
	if err != nil {
		log.Println("Error Open DB : ", err)
	}

	query := ""
	if search != "" {
		query += fmt.Sprintf("WHERE LOWER(full_name) like '%%%s%%' ", search)
	}

	searchQuery := strings.Replace(queryGetUser, "%search%", query, -1)
	stmt, err := db.Preparex(searchQuery)
	if err != nil {
		log.Println("Error Prepare DB : ", err)
		return
	}

	rows, err := stmt.Queryx(limit, offset)
	if err != nil {
		log.Println("Error Querying : ", err)
		return
	}

	for rows.Next() {
		u := User{}
		err := rows.StructScan(&u)
		if err != nil {
			log.Println("Error Struct Scan : ", err)
		}
		u.Age = countAge(u.BirthDate, time.Now())
		users = append(users, u)
	}
	return
}

func countAge(birth, now time.Time) int {
	year, _, _, _, _, _ := diff(birth, time.Now())

	return year
}

func diff(a, b time.Time) (year, month, day, hour, min, sec int) {
	if a.Location() != b.Location() {
		b = b.In(a.Location())
	}
	if a.After(b) {
		a, b = b, a
	}
	y1, M1, d1 := a.Date()
	y2, M2, d2 := b.Date()

	h1, m1, s1 := a.Clock()
	h2, m2, s2 := b.Clock()

	year = int(y2 - y1)
	month = int(M2 - M1)
	day = int(d2 - d1)
	hour = int(h2 - h1)
	min = int(m2 - m1)
	sec = int(s2 - s1)

	// Normalize negative values
	if sec < 0 {
		sec += 60
		min--
	}
	if min < 0 {
		min += 60
		hour--
	}
	if hour < 0 {
		hour += 24
		day--
	}
	if day < 0 {
		// days in month:
		t := time.Date(y1, M1, 32, 0, 0, 0, 0, time.UTC)
		day += 32 - t.Day()
		month--
	}
	if month < 0 {
		month += 12
		year--
	}

	return
}
