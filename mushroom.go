package main

import (
	"fmt"
	"strconv"
	"flag"
	"strings"
	"bytes"
	"io"
	"path"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	_"github.com/gorilla/mux"
	"github.com/golang/groupcache"
	"github.com/ha/doozer"
	"html/template"
	"github.com/disintegration/imaging"
)


var (
	// cacheAddr     = flag.String("cache", "127.0.0.1:8000", "Address for groupcache")
	listenAddr    = flag.String("listen", "localhost:4000", "Address to listen on")
	doozerAddr    = flag.String("doozer", "127.0.0.1:8046", "Doozerd Config Server")
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

	// Setup the doozer connection.
	d, err := doozer.Dial(*doozerAddr)
	if err != nil {
		log.Fatalf("connecting to doozer: %v\n", err)
	}
	defer d.Close()

	// Setup the cache.
	pool = groupcache.NewHTTPPool("http://" + *listenAddr)
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

	// Start watching for changes and signals.
	go watch(d)

	// Add the handler for definition requests and then start the
	// server.
	http.Handle("/images/", http.HandlerFunc(imgHandler))
	log.Println(http.ListenAndServe(*listenAddr, nil))

	// setup the router
	/*router := mux.NewRouter()
	router.HandleFunc("/", rootHandler)
	router.HandleFunc(imgHandlePath, imgHandler)

	http.Handle("/", router)
	httpErr := http.ListenAndServe(*listenAddr, nil)
	if httpErr != nil {
		log.Fatal(httpErr)
	}
*/
}

// watch updates the peer list of servers based on changes to the
// doozer configuration or signals from the OS.
func watch(d *doozer.Conn) {
	peerFile := "/peers"
	var peers []string
	var rev int64

	// Run the initial get.
	data, rev, err := d.Get(peerFile, nil)
	if err != nil {
		log.Println("initial peer list get:", err)
		log.Println("using empty set to start")
		peers = []string{}
	} else {
		peers = strings.Split(string(data), " ")
	}

	// Add myself to the list.
	peers = append(peers, "http://" + *listenAddr)
	rev, err = d.Set(peerFile, rev,
		[]byte(strings.Join(peers, " ")))
	if err != nil {
		log.Println("unable to add myself to the peer list (no longer watching).")
		return
	}
	pool.Set(peers...)
	log.Println("added myself to the peer list, Current Peers: ", peers)

	// Setup signal handling to deal with ^C and others.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, os.Kill)

	// Get the channel that's listening for changes.
	updates := wait(d, peerFile, &rev)

	for {
		select {
		case <- sigs:
			// Remove ourselves from the peer list and exit.
			for i, peer := range peers {
				if peer == "http://" + *listenAddr {
					peers = append(peers[:i], peers[i+1:]...)
					d.Set(peerFile, rev, []byte(strings.Join(peers, " ")))
					log.Println("removed myself from peer list before exiting.")
				}
			}
			os.Exit(1)
		case update, ok := <-updates:
			// If the channel was closed, we should stop selecting on it.
			if !ok {
				updates = nil
				continue
			}

			// Otherwise, update the peer list.
			peers = update
			log.Println("got new peer list:", strings.Join(peers, " "))
			pool.Set(peers...)
		}
	}
}

func wait(d *doozer.Conn, file string, rev *int64) chan []string {
	c := make(chan []string, 1)
	cur := *rev
	go func() {
		for {
			// Wait for the change.
			e, err := d.Wait(file, cur+1)
			if err != nil {
				log.Println("waiting failed (no longer watching):", err)
				close(c)
				return
			}
			// Update the revision and send the change on the channel.
			atomic.CompareAndSwapInt64(rev, cur, e.Rev)
			cur = e.Rev
			c <- strings.Split(string(e.Body), " ")
		}
	}()

	return c
}

func imgHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("received request:", r.Method, r.URL.Path)
	imgFile := strings.Trim(path.Base(r.URL.Path), "/")

	/*
	reqVars := mux.Vars(r)
	imgFile := reqVars["imgFile"]
	*/

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
