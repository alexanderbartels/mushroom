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
	_"sync/atomic"
	_"github.com/gorilla/mux"
	"github.com/golang/groupcache"
	_"github.com/ha/doozer"
	"html/template"
	"github.com/disintegration/imaging"
	"github.com/alexanderbartels/mushroom/distributed"
	"github.com/alexanderbartels/mushroom/keys"
)


var (
	// cacheAddr     = flag.String("cache", "127.0.0.1:8000", "Address for groupcache")
	listenAddr    = flag.String("listen", "localhost:4000", "Address to listen on")
	doozerAddr    = flag.String("doozer", "127.0.0.1:8046", "Doozerd Config Server")
	imageSrc      = flag.String("imageSrc", "img/", "Directory with images to serve")

	// This is our groupcache stuff.
	pool     *groupcache.HTTPPool
	imgCache *groupcache.Group

	// save the current active list of caching peers
	cachingPeers    []string

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

	// setup distributed configuration
	dc, err := distributed.NewConfig(*doozerAddr)
	if err != nil {
		log.Fatalf("connecting to distributed config (doozer): %v\n", err)
	}
	defer dc.Close()


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

	// create channels to listen on for new updates
	cachingPeerUpdates := handleCachingPeers(&dc)
	signalingUpdates   := handleSignaling()

	// handle the different channels
	go handleUpdates(&dc, cachingPeerUpdates, signalingUpdates)

	// write initial caching peers to the update channel, so that the update func will setup the initial CachePool
	cachingPeerUpdates <- cachingPeers

	// Add the handler for definition requests and then start the
	// server.
	http.Handle("/images/", http.HandlerFunc(imgHandler))
	log.Println(http.ListenAndServe(*listenAddr, nil))
}

// Setup signal handling to deal with ^C and others.
func handleSignaling() chan os.Signal {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, os.Kill)

	return sigs
}

func handleCachingPeers(dc *distributed.Config) chan []string {
	// fetch the initial configuration.
	cachingPeers = distributed.FetchConfigItem(dc, "/cachingPeers")
	log.Println("Initial caching peers fetched: ", strings.Join(cachingPeers, " "))

	// add myself to the list
	cachingPeers = append(cachingPeers, "http://" + *listenAddr)
	distributed.SetConfigItem(dc, "/cachingPeers", cachingPeers)
	log.Println("Added myself to the active caching peers. Current Peers: ", cachingPeers)

	// create channel to watching for changes on "CachingPeers"
	cachingPeerUpdates := distributed.WatchConfig(dc, "/cachingPeers")
	return cachingPeerUpdates
}

// watch updates the peer list of servers based on changes to the
// doozer configuration or signals from the OS.
func handleUpdates(dc *distributed.Config, cachingPeerUpdates chan []string, signalingUpdates chan os.Signal) {
	for {
		select {
		case <- signalingUpdates:
			// Remove ourselves from the peer list and exit.
			for i, peer := range cachingPeers {
				if peer == "http://" + *listenAddr {
					// remove myself from the cachingPeers
					cachingPeers = append(cachingPeers[:i], cachingPeers[i+1:]...)
					distributed.SetConfigItem(dc, "/cachingPeers", cachingPeers)
					log.Println("Removed myself from caching peers before exiting.")
				}
			}
			os.Exit(1)
		case update, ok := <-cachingPeerUpdates:
			// If the channel was closed, we should stop selecting on it.
			if !ok {
				log.Println("Channel for active caching peers was closed. Stop Watching on it.")
				cachingPeerUpdates = nil
				continue
			}

			// Otherwise, update the peer list.
			cachingPeers = update
			log.Println("Got new caching peers:", strings.Join(cachingPeers, " "))
			pool.Set(cachingPeers...)
		}
	}
}

func imgHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Request received:", r.Method, r.URL.Path)

	imgFile := strings.Trim(path.Base(r.URL.Path), "/")
	imgKey  := keys.Generate(imgFile, r.URL.Query())
	log.Println("Generated Key from Request:", imgKey)

	// Get the image from cache and write it out.
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
