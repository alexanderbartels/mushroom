package main

import (
	_"fmt"
	"strconv"
	"flag"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"html/template"
	"github.com/disintegration/imaging"
)


var (
	listenAddr    = flag.String("listen", "localhost:4000", "Address to listen on")
	imageSrc      = flag.String("imageSrc", "img/", "Directory with images to serve")
	imgHandlePath = "/images/{imgFile}"

	rootTemplate  = template.Must(template.New("root").Parse(`
		<!DOCTYPE html>
		<html>
		<head>
		<meta charset="utf-8" />
		</head>
		<body>
			<h1> Welcome To Mushroom Image-Resizer </h1>
		</body>
		</html>
	`))
)

func main() {
	// parse the command line flags
	flag.Parse()

	router := mux.NewRouter()
	router.HandleFunc("/", rootHandler)
	router.HandleFunc(imgHandlePath, imgHandler)

	http.Handle("/", router)
	err := http.ListenAndServe(*listenAddr, nil)
	if err != nil {
		log.Fatal(err)
	}

}

func imgHandler(w http.ResponseWriter, r *http.Request) {
	reqVars := mux.Vars(r)
	imgFile := reqVars["imgFile"]

	width := r.URL.Query().Get("width")
	if len(width) == 0 {
		width = "0"
	}
	widthAsInt, _ := strconv.Atoi(width)

	height := r.URL.Query().Get("height")
	if len(height) == 0 {
		height = "0"
	}
	heightAsInt, _ := strconv.Atoi(height)

	// read image
	dstimg, err := imaging.Open(*imageSrc + imgFile)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// manipulate image
	if widthAsInt > 0 || heightAsInt > 0 {
		dstimg = imaging.Resize(dstimg, widthAsInt, heightAsInt, imaging.Box)
	}

	// write image
	//w.Header().Set("Content-type", "image/png")
	imaging.Encode(w, dstimg, imaging.PNG)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	rootTemplate.Execute(w, listenAddr)
}
