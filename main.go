package main

import (
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type Message struct {
	Dose string
}

// Dirctory for static content to be served from
const dir = "./static"

func main() {
	// Handle server routes
	r := mux.NewRouter()
	r.HandleFunc("/", home)
	r.HandleFunc("/cal", cal)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(dir)))
	http.Handle("/", r)
	server := http.Server{
		Addr:    ":8001",
		Handler: r,
	}
	server.ListenAndServe()
}

const upperLimit = 8

func home(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles(dir + "/index.html")
	if err != nil {
		log.Println(err.Error())
	}
	m := Message{}
	m.Dose = r.URL.Query().Get("dose")

	t.Execute(w, m)
}

func cal(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	bs := r.PostFormValue("bs")
	carb := r.PostFormValue("carb")
	v := url.Values{}
	bsFl, err := strconv.ParseFloat(bs, 64)
	if err != nil {
		log.Fatal(err)
	}
	carbFl, err := strconv.ParseFloat(carb, 64)
	if err != nil {
		log.Fatal(err)
	}
	v.Add("dose", strconv.FormatFloat(
		calculate(
			bsFl,
			carbFl,
			workOutRatio(),
		),
		'f', -1, 64))
	http.Redirect(w, r, "/?"+v.Encode(), 302)
}

func calculate(bs float64, carbs float64, ratio int) float64 {
	diff := 0.0
	if bs > upperLimit {
		diff = bs - upperLimit
	}
	return Round(carbs/float64(ratio) + diff)
}

func workOutRatio() int {
	location, err := time.LoadLocation("GMT")
	if err != nil {
		fmt.Println(err)
	}
	t := time.Now().In(location)

	morning := map[string]time.Time{
		"start": time.Date(t.Year(), t.Month(), t.Day(), 5, 0, 0, 0, location),
		"end":   time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, location),
	}

	afternoon := map[string]time.Time{
		"start": time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, location),
		"end":   time.Date(t.Year(), t.Month(), t.Day(), 17, 0, 0, 0, location),
	}

	evening := map[string]time.Time{
		"start": time.Date(t.Year(), t.Month(), t.Day(), 17, 0, 0, 0, location),
		"end":   time.Date(t.Year(), t.Month(), t.Day(), 23, 0, 0, 0, location),
	}

	if t.After(morning["start"]) && t.Before(morning["end"]) {
		return 5
	}

	if t.After(afternoon["start"]) && t.Before(afternoon["end"]) {
		return 6
	}

	if t.After(evening["start"]) && t.Before(evening["end"]) {
		return 6
	}

	return 5
}

func Round(x float64) float64 {
	const (
		mask  = 0x7FF
		shift = 64 - 11 - 1
		bias  = 1023

		signMask = 1 << 63
		fracMask = (1 << shift) - 1
		halfMask = 1 << (shift - 1)
		one      = bias << shift
	)

	bits := math.Float64bits(x)
	e := uint(bits>>shift) & mask
	switch {
	case e < bias:
		// Round abs(x)<1 including denormals.
		bits &= signMask // +-0
		if e == bias-1 {
			bits |= one // +-1
		}
	case e < bias+shift:
		// Round any abs(x)>=1 containing a fractional component [0,1).
		e -= bias
		bits += halfMask >> e
		bits &^= fracMask >> e
	}
	return math.Float64frombits(bits)
}
