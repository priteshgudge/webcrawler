package crawlerlib

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"regexp"
)

// delegator acts a medium for the scrapers and does the following
// 1. Distributed the urls to scrapers
// 2. limit domain
type delegator struct {
	baseURL        *url.URL            // starting url at maxDepth 0
	scrapers        []*scraper         // scrapers that are controlled by this delegator
	scrappedUnique map[string]int      // scrappedUnique holds the map of unique urls we crawled and times its repeated
	unScrapped     map[int][]*url.URL  // unScrapped are those that are yet to be crawled by the scrapers
	scrapped       map[int][]*url.URL  // scrapped holds url found in each depth
	skippedURLs    map[string][]string // skippedURLs contains urls from different domains(if domainRegex is failed) and all invalid urls
	errorURLs      map[string]error    // reason why this url was not crawled
	submitDumpCh   chan *scraperDumps  // submitDump listens for scrapers to submit their dumps
	domainRegex    *regexp.Regexp      // restricts crawling the urls that pass the
	maxDepth       int                 // maxDepth of crawl, -1 means no limit for maxDepth
	interrupted    bool                // says if delegator was interrupted while scraping
	processors     []processor         // list of url processors
}

// scraperPayload holds the urls for the scraper to crawl and scrape
type scraperPayload struct {
	currentDepth int        // depth at which these urls are scrapped from
	urls         []*url.URL // urls to be crawled
}

// scraperDump is the crawl dump by single scraper of a given sourceURL
type scraperDump struct {
	depth       int        // depth at which the urls are scrapped(+1 of sourceURL depth)
	sourceURL   *url.URL   // sourceURL the scraper crawled
	urls        []*url.URL // urls obtained from sourceURL page
	invalidURLs []string   // urls which couldn't be normalized
	err         error      // reason why url is not crawled
}

// scraperDumps holds the crawled data and chan to confirm that dumps are accepted
type scraperDumps struct {
	scraper string
	got    chan bool
	mds    []*scraperDump
}

// newDelegator returns a new delegator with given base url and maxDepth
func newDelegator(baseURL *url.URL, maxDepth int) *delegator {
	g := &delegator{
		baseURL:        baseURL,
		scrappedUnique: make(map[string]int),
		unScrapped:     make(map[int][]*url.URL),
		scrapped:       make(map[int][]*url.URL),
		skippedURLs:    make(map[string][]string),
		errorURLs:      make(map[string]error),
		submitDumpCh:   make(chan *scraperDumps),
		maxDepth:       maxDepth,
		processors: []processor{
			uniqueURLProcessor(),
			errorCheckProcessor(),
			skippedURLProcessor(),
			maxDepthCheckProcessor(),
			domainFilterProcessor(),
		},
	}

	r, _ := regexp.Compile(baseURL.Hostname())
	g.domainRegex = r
	log.Printf("delegator: setting default domain regex to %v\n", r)
	return g
}

// setDomainRegex sets the domainRegex for the delegator
func setDomainRegex(g *delegator, regexStr string) error {
	r, err := regexp.Compile(regexStr)
	if err != nil {
		return fmt.Errorf("failed to compile domain regex: %v\n", err)
	}

	log.Printf("updated domain regex: %v\n", r)
	g.domainRegex = r
	return nil
}

// getIdleScrapers will return all the idle scrapers
func getIdleScrapers(g *delegator) (idleScrapers []*scraper) {
	for _, m := range g.scrapers {
		if isBusy(m) {
			continue
		}

		idleScrapers = append(idleScrapers, m)
	}

	return idleScrapers
}

// pushPayloadToscraper will push payload to scraper
func pushPayloadToScraper(m *scraper, depth int, urls []*url.URL) {
	m.payloadCh <- &scraperPayload{
		currentDepth: depth,
		urls:         urls,
	}
}

// distributePayload will distribute the given urls to idle scrapers, error when there are no idle scrapers
func distributePayload(g *delegator, depth int, urls []*url.URL) error {
	ims := getIdleScrapers(g)
	if len(ims) == 0 {
		return errors.New("all scrapers are busy")
	}

	if len(urls) <= len(ims) {
		for i, u := range urls {
			go pushPayloadToScraper(ims[i], depth, []*url.URL{u})
		}
		return nil
	}

	wd := len(urls) / len(ims)
	i := 0
	for mi, m := range ims {
		if mi+1 == len(ims) {
			go pushPayloadToScraper(m, depth, urls[i:])
			continue
		}

		go pushPayloadToScraper(m, depth, urls[i:i+wd])
		i += wd

	}
	return nil
}

// processDump will process a single scraperDump
func processDump(g *delegator, md *scraperDump) {
	g.scrapped[md.depth-1] = append(g.scrapped[md.depth-1], md.sourceURL)
	for _, p := range g.processors {
		r := p.process(g, md)
		if !r {
			return
		}
	}

	// add the md.urls to unscrapped and md.source to scraped
	if len(md.urls) > 0 {
		g.unScrapped[md.depth] = append(g.unScrapped[md.depth], md.urls...)
	}
}

// processDumps process the scraper dumps and signals when the crawl is complete
func processDumps(g *delegator, mds []*scraperDump) (finished bool) {
	log.Println("processing dumps...")
	for _, md := range mds {
		processDump(g, md)
	}
	log.Println("processing done...")

	if len(getIdleScrapers(g)) < 1 {
		log.Println("all scrapers are busy. deferring payload distribution...")
		return false
	}

	if len(g.unScrapped) > 0 {
		for d, urls := range g.unScrapped {
			err := distributePayload(g, d, urls)
			if err != nil {
				log.Printf("failed to distribute load: %v\n", err)
				break
			}
			delete(g.unScrapped, d)
			log.Printf("distributed payload at depth: %d\n", d)
		}

		return false
	}

	if len(getIdleScrapers(g)) == len(g.scrapers) && len(g.unScrapped) == 0 {
		log.Println("scrapping done...")
		return true
	}

	log.Println("no urls to scrape at the moment...")
	return false
}

// startDelegator initiates delegator to start scraping
func startDelegator(ctx context.Context, g *delegator) {
	log.Printf("Starting Delegator with Base URL: %s\n", g.baseURL)
	distributePayload(g, 0, []*url.URL{g.baseURL})

	for {
		select {
		case <-ctx.Done():
			log.Println("scrapping interrupted...")
			g.interrupted = true
			return
		case mds := <-g.submitDumpCh:
			go func(got chan<- bool) { got <- true }(mds.got)
			log.Printf("got new dump from %s\n", mds.scraper)
			done := processDumps(g, mds.mds)
			if done {
				log.Println("stopping Delegator...")
				return
			}
		}
	}
}
