package crawlerlib

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// scraper crawls the link, scrape urls normalises then and returns the dump to delegator
type scraper struct {
	name            string
	busy            bool                 // busy represents whether scraper is idle/busy
	mu              *sync.RWMutex        // protects the above
	payloadCh       chan *scraperPayload // payload listens for urls to be scrapped
	delegatorDumpCh chan<- *scraperDumps // delegatorDumpCh to send finished data to delegator
}

// newScraper returns a new scraper under given delegator
func newScraper(name string, delegatorDumpCh chan<- *scraperDumps) *scraper {
	return &scraper{
		name:            name,
		mu:              &sync.RWMutex{},
		payloadCh:       make(chan *scraperPayload),
		delegatorDumpCh: delegatorDumpCh,
	}
}

// isBusy says if the scraper is busy or idle
func isBusy(m *scraper) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.busy
}

// crawlURL crawls the url and extracts the urls from the page
func crawlURL(depth int, u *url.URL) (md *scraperDump) {
	resp, err := http.DefaultClient.Get(u.String())
	if err != nil {
		return &scraperDump{
			depth:     depth + 1,
			sourceURL: u,
			err:       err,
		}
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &scraperDump{
			depth:     depth + 1,
			sourceURL: u,
			err:       fmt.Errorf("url responsed with code %d", resp.StatusCode),
		}
	}

	ct := resp.Header.Get("Content-type")
	if ct != "" && !strings.Contains(ct, "text/html") {
		return &scraperDump{
			depth:     depth + 1,
			sourceURL: u,
			err:       fmt.Errorf("unknown content type: %s", ct),
		}
	}

	s, iu := extractURLsFromHTML(u, resp.Body)
	return &scraperDump{
		depth:       depth + 1,
		sourceURL:   u,
		urls:        s,
		invalidURLs: iu,
	}
}

// crawlURLs crawls given urls and return extracted url from the page
func crawlURLs(depth int, urls []*url.URL) (mds []*scraperDump) {
	for _, u := range urls {
		mds = append(mds, crawlURL(depth, u))
	}

	return mds
}

// startScraper starts the scraper
func startScraper(ctx context.Context, m *scraper) {
	log.Printf("Starting %s...\n", m.name)

	for {
		select {
		case <-ctx.Done():
			return
		case mp := <-m.payloadCh:
			m.busy = true
			log.Printf("Crawling urls(%d) from depth %d\n", len(mp.urls), mp.currentDepth)
			mds := crawlURLs(mp.currentDepth, mp.urls)
			got := make(chan bool)
			m.delegatorDumpCh <- &scraperDumps{
				scraper: m.name,
				got:    got,
				mds:    mds,
			}
			<-got
			m.busy = false
		}
	}
}
