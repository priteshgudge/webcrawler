# WebCrawler Implementation in Golang

[![Build Status](https://travis-ci.com/priteshgudge/webcrawler.svg?branch=master)](https://travis-ci.com/priteshgudge/webcrawler)

## Running the crawler

`go run cmd/crawl/main.go --url=https://mongo.com --scraper-concurrency=10` 

## Primary Crawler Code
Primary Crawler code resides in `crawlerlib`

Run: `go test` in `crawlerlib` to verify tests

### Notes:

This project uses go modules

External libary is used `golang.org/x/net` is used for parsing html
