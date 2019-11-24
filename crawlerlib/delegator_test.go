package crawlerlib

import (
	"fmt"
	"net/url"
	"sync"
	"testing"
)

func Test_getIdleScrapers(t *testing.T) {
	g := &delegator{
		scrapers: []*scraper{
			{
				name: "scraper 1",
				busy: true,
				mu:   &sync.RWMutex{},
			},

			{
				name: "scraper 2",
				mu:   &sync.RWMutex{},
			},

			{
				name: "scraper 3",
				mu:   &sync.RWMutex{},
			},

			{
				name: "scraper 4",
				busy: true,
				mu:   &sync.RWMutex{},
			},
		},
	}

	expectedIdleScrapers := map[string]bool{
		"scraper 2": true,
		"scraper 3": true,
	}

	idleScrapers := getIdleScrapers(g)
	if len(idleScrapers) != len(expectedIdleScrapers) {
		t.Fatalf("expected idle scrapers %d but got %d", len(expectedIdleScrapers), len(idleScrapers))
	}

	for _, s := range idleScrapers {
		_, ok := expectedIdleScrapers[s.name]
		if !ok {
			t.Fatalf("expected %s to be idle but busy\n", s.name)
		}
	}
}

func Test_distributePayload(t *testing.T) {
	tests := []struct {
		scrapers int
		busy    int
		urls    int
	}{
		{
			scrapers: 4,
			busy:    4,
		},

		{
			scrapers: 5,
			urls:    4,
		},

		{
			scrapers: 5,
			busy:    1,
			urls:    4,
		},

		{
			scrapers: 5,
			urls:    10,
		},
	}

	scraperCreateF := func(g *delegator, total, busy int) (scrapers []*scraper) {
		for i := 0; i < total; i++ {
			scrapers = append(scrapers, newScraper(fmt.Sprintf("scraper %d", i), g.submitDumpCh))
		}

		for i := 0; i < busy; i++ {
			scrapers[i].busy = true
		}

		return scrapers
	}

	urlCreateF := func(total int) (urls []*url.URL) {
		for i := 0; i < total; i++ {
			u, _ := url.Parse(fmt.Sprintf("http://test.com/%d", i))
			urls = append(urls, u)
		}

		return urls
	}

	for _, c := range tests {
		baseURL, _ := url.Parse("http://test.com")
		g := newDelegator(baseURL, 1)
		g.scrapers = scraperCreateF(g, c.scrapers, c.busy)
		urls := urlCreateF(c.urls)

		testCh := make(chan int)
		for _, m := range g.scrapers {
			go func(m *scraper) {
				for mp := range m.payloadCh {
					testCh <- len(mp.urls)
				}
			}(m)
		}
		distributePayload(g, 1, urls)

		count := 0
		busyScrapers := c.urls
		if busyScrapers > (c.scrapers - c.busy) {
			busyScrapers = c.scrapers - c.busy
		}
		for i := 0; i < busyScrapers; i++ {
			count += <-testCh
		}

		if c.urls != count {
			t.Fatalf("expected %d urls but got %d", c.urls, count)
		}

	}
}

