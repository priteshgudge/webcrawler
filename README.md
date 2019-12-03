# WebCrawler Implementation in Golang

## Running the crawler

`go run cmd/crawl/main.go --url=https://mongo.com --scraper-concurrency=10` 

## Primary Crawler Code
Primary Crawler code resides in `crawlerlib`

Run: `go test` in `crawlerlib` to verify tests

# Implementation
The numbers denote the steps in a single scraping workflow.

![Workflow Diagram](https://github.com/priteshgudge/webcrawler/blob/master/assets/WebCrawler.png)

### Notes:

This project uses go modules

External libary is used `golang.org/x/net` is used for parsing html
