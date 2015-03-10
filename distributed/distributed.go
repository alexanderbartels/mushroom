package distributed

import (
	"fmt"
	"log"
	"flag"
	"strings"
	"sync/atomic"
	"github.com/ha/doozer"
	"github.com/golang/groupcache"
	"github.com/alexanderbartels/mushroom/keys"
	"github.com/alexanderbartels/mushroom/image"
)

var (
	imageSrc      = flag.String("imageSrc", "img/", "Directory with images to serve")
)

type Config struct {
	d *doozer.Conn
	rev int64
}

// Close Distributed Config and connection to doozer
func (dc *Config) Close() {
	dc.d.Close()
}

func (d *Config) Exit() {

}

// fetch configuration for the given file from the given distributed config
func FetchConfigItem(dc *Config, file string) []string{
	data, rev, err := dc.d.Get(file, nil)
	var dataString []string

	if err != nil {
		log.Println("Fetch initial config file (", file, ") failed:", err)
		log.Println("using empty config to start")
		dataString = []string{}
	} else {
		dataString = strings.Split(string(data), " ")
	}

	dc.rev = rev
	return dataString
}

// Watches for changes on the given file in the given distributed configuration
func WatchConfig(dc *Config, file string) chan []string {
	// channel for config updates on the given file
	c := make(chan []string, 1)

	// start func to wait for changes
	go func() {
		for {
			// Wait for the change.
			e, err := dc.d.Wait(file, dc.rev+1)
			if err != nil {
				log.Println("waiting failed (no longer watching):", err)
				close(c)
				return
			}
			// Update the revision and send the change on the channel.
			atomic.CompareAndSwapInt64(&dc.rev, dc.rev, e.Rev)
			c <- strings.Split(string(e.Body), " ")
		}
	}()

	return c
}

func SetConfigItem(dc *Config, file string, items []string) {
	rev, err := dc.d.Set(file, dc.rev,
		[]byte(strings.Join(items, " ")))
	if err != nil {
		log.Println("unable to add myself to the peer list (no longer watching).")
		return
	}
	dc.rev = rev
}

// creates an new distributed config
// addr: Address for the distributed config server (doozerd)
func NewConfig(addr string) (Config, error) {
	// Setup the doozer connection.
	d, err := doozer.Dial(addr)
	if err != nil {
		log.Fatalf("connecting to doozer: %v\n", err)
		return  Config{}, err
	}

	return Config{d, 0}, nil
}

func CacheItemProvider(ctx groupcache.Context, key string, dest groupcache.Sink) error {
	// parse the key
	captures, err := keys.Parse(key)
	if err != nil {
		err = fmt.Errorf("Querying image (parse key): %v", err)
		log.Println(err)
		return err
	}

	// read image with the file provider
	imgProvider := image.FileProvider{}
	imgProvider.Src = *imageSrc + captures[keys.FILE_NAME]
	img, loadingErr := imgProvider.Provide()
	if loadingErr != nil {
		loadingErr = fmt.Errorf("Querying image (load image): %v", loadingErr)
		log.Println(loadingErr)
		return loadingErr
	}

	// process the image
	imgProcessor := image.DefaultProcessor{}
	imgProcessor.Img = &img
	buf, processErr := imgProcessor.Process(captures)
	if processErr != nil {
		processErr = fmt.Errorf("Querying image (process image): %v", processErr)
		log.Println(processErr)
		return processErr
	}

	log.Println("Querying image: successful with key =", key)
	// image loading and processing successful -> write it to the cache
	dest.SetBytes(buf.Bytes())
	return nil
}
