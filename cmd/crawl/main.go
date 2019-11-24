package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/priteshgudge/webcrawler/crawlerlib"
	"log"
	"os"
	"runtime"
)


func main() {
	log.SetFlags(log.Ldate | log.Lshortfile)
	flag.CommandLine.SetOutput(os.Stdout)

	baseURL := flag.String("url", "https://monzo.com", "Starting URL")
	maxDepth := flag.Int("max-depth", 3, "Max depth to Crawl")
	domainRegex := flag.String("domain-regex", "monzo", "Domain regex to limit crawls to. Defaults to base url domain")
	sitemapFile := flag.String("sitemap", "sitemap.xml", "File location to write sitemap to")
	scraperConcurrency := flag.Int("scraper-concurrency", runtime.NumCPU()*2, "Number of concurrent scrapers")
	help := flag.Bool("help", false, "Show Options")
	flag.Parse()

	if *help {
		fmt.Fprintf(os.Stdout, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		return
	}

	if *baseURL == "" {
		log.Fatal("start URL cannot be empty")
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	resp, err := crawlerlib.StartWithDepthAndDomainRegex(ctx, *baseURL, *maxDepth, *domainRegex, scraperConcurrency)
	if err != nil {
		log.Fatalf("couldn't start scrape: %v\n", err)
	}

	if *sitemapFile != "" {
		crawlerlib.Sitemap(resp, *sitemapFile)
		return
	}

	fmt.Print(resp)
}
