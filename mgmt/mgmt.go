package mgmt

import (
	"log"
	"net/http"
	"github.com/golang/groupcache"
)

type CachingStats struct {
	Gets int64

}

func NewCachingStatsHandler(imgCacheGroup *groupcache.Group) (func(http.ResponseWriter, *http.Request)) {
	return func (w http.ResponseWriter, r *http.Request) {
		log.Println("received request:", r.Method, r.URL.Path)

		cacheStats := imgCacheGroup.CacheStats(groupcache.MainCache)
		log.Println("Cache Stats: ", cacheStats, "-", imgCacheGroup.Stats)
	}
}
