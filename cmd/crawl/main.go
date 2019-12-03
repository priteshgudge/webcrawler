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
	sitemapFile := flag.String("sitemap", "sitemap.xml", "File location to write sitemap to")
	scraperConcurrency := flag.Int("concurrency", runtime.NumCPU()*2, "Number of concurrent scrapers")
	domain := flag.String("domain", "monzo.com", "Domain for URLs")
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

	log.Printf("Scraping url: %s  maxDepth: %d concurrency: %d", *baseURL, *maxDepth, *scraperConcurrency)
	resp, err := crawlerlib.StartWithDepthAndDomainRegex(ctx, *baseURL, *maxDepth,*domain,  *scraperConcurrency)
	if err != nil {
		log.Fatalf("couldn't start scrape: %v\n", err)
	}

	if *sitemapFile != "" {
		crawlerlib.Sitemap(resp, *sitemapFile)
		return
	}

	fmt.Print(resp)
}
