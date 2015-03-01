package distributed

import (
	"log"
	"strings"
	"sync/atomic"
	"github.com/ha/doozer"
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
