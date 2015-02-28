package main

import (
	"fmt"
	"strconv"
	"flag"
	"strings"
	"bytes"
	"io"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/golang/groupcache"
	"html/template"
	"github.com/disintegration/imaging"
)


var (
	cacheAddr     = flag.String("cache", "127.0.0.1:8000", "Address for groupcache")
	listenAddr    = flag.String("listen", "localhost:4000", "Address to listen on")
	imageSrc      = flag.String("imageSrc", "img/", "Directory with images to serve")
	imgHandlePath = "/images/{imgFile}"

	// This is our groupcache stuff.
	pool     *groupcache.HTTPPool
	imgCache *groupcache.Group

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

	// Setup the cache.
	pool = groupcache.NewHTTPPool("http://" + *cacheAddr)
	imgCache = groupcache.NewGroup("img", 64<<20, groupcache.GetterFunc(
			func (ctx groupcache.Context, key string, dest groupcache.Sink) error {
				img, err := query(key)
				if err != nil {
					err = fmt.Errorf("querying image: %v", err)
					log.Println(err)
					return err
				}

				log.Println("retrieved image for", key)
				dest.SetBytes(img)
				return nil
			}))

	/**
	 * Neue CahceGroup 'grayscaleImgs' wenn als param ?action=grayscale angehaengt wird
	 * in der GetterFunc könnte das Bild mit Höhe und Breite aus der 'img'-CacheGroup abgefragt werden.
	 * so können die Bilder mit angepasster Größe für verscheidene Actions aus dem Cache geladen werden.
	 *
	 * Weitere CacheGroup 'blurImgs', wenn als param?action=blur angehaengt wird.... usw.. ?!
	 *
	 */

	// setup the router
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
	log.Println("received request:", r.Method, r.URL.Path)

	reqVars := mux.Vars(r)
	imgFile := reqVars["imgFile"]

	width := r.URL.Query().Get("width")
	if len(width) == 0 {
		width = "0"
	}

	widthAsInt, wErr := strconv.Atoi(width)
	if wErr != nil || widthAsInt < 0 {
		log.Println("Invalid Width specified ", widthAsInt, " - ", wErr)
		widthAsInt = 0;
	}

	height := r.URL.Query().Get("height")
	if len(height) == 0 {
		height = "0"
	}

	heightAsInt, hErr := strconv.Atoi(height)
	if hErr != nil || heightAsInt < 0 {
		log.Println("Invalid Width specified ", heightAsInt, " - ", hErr)
		heightAsInt = 0;
	}

	// img key generation
	imgKey := imgFile + "?width=" + strconv.Itoa(widthAsInt) + "&height=" + strconv.Itoa(heightAsInt)
	log.Println("Try retreiving image from cache. Key:", imgKey)

	// Get the image from groupcache and write it out.
	var data []byte
	err := imgCache.Get(nil, imgKey, groupcache.AllocatingByteSliceSink(&data))
	if err != nil {
		log.Println("error while retreiving image for", imgKey, "-", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO: set some nice HTTP Headers ?
	io.Copy(w, bytes.NewReader(data))
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	rootTemplate.Execute(w, listenAddr)
}

func query(key string) ([]byte, error) {
	// split key
	splitted := strings.Split(key, "?")
	log.Println("Reading image from disk: ", splitted[0])

	// split into array of params
	params := strings.Split(splitted[1], "&")

	var height int
	var width  int

	for _, v := range params {

		keyValueParam := strings.Split(v, "=")

		switch {
		case strings.EqualFold("width", keyValueParam[0]):
			width, _ = strconv.Atoi(keyValueParam[1])
		case strings.EqualFold("height", keyValueParam[0]):
			height, _ = strconv.Atoi(keyValueParam[1])
		default:
			log.Println("Unsupported Param for Image: ", keyValueParam[0])
		}
	}

	// read image
	dstimg, err := imaging.Open(*imageSrc + splitted[0])
	if err != nil {
		return nil, err
	}

	// manipulate image
	if width > 0 || height > 0 {
		dstimg = imaging.Resize(dstimg, width, height, imaging.Box)
	}

	// write image
	buf := new(bytes.Buffer)
	encodeErr := imaging.Encode(buf, dstimg, imaging.PNG)

	return buf.Bytes(), encodeErr
}
