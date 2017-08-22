# PasteScrape
Pastebin scraper written in go, which logs into an elasticsearch backend.

The paste key is used as identifier and pastes are fetched only once.

## Pastebin Scraping API
In order to use the pastebin scraping API you will require a pastebin pro account, see [https://pastebin.com/pro](https://pastebin.com/pro).

## Usage

```
Usage of ./paste-scrape:
  -index string
        pastebin (default "pastebin")
  -url string
        elastic search api url (default "http://localhost:9200")
```

## Misc
The program may take up to 40 seconds to shutdown after SIGINT (Ctrl + C).

## TODO

- [ ] more test coverage
- [ ] use httptest for tests
- [ ] adaptive polling
- [ ] more output modules (csv, json)
