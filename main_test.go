package main

import (
	"os"
	"testing"

	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

func TestMain(m *testing.M) {
	httpmock.RegisterResponder("GET", "https://pastebin.com/api_scraping.php",
		httpmock.NewStringResponder(200, `[
    {
        "scrape_url": "https://pastebin.com/api_scrape_item.php?i=bRuV7pSD",
        "full_url": "https://pastebin.com/bRuV7pSD",
        "date": "1503599704",
        "key": "bRuV7pSD",
        "size": "38042",
        "expire": "0",
        "title": "",
        "syntax": "text",
        "user": ""
    },
	]`))

	httpmock.RegisterResponder("GET", "https://pastebin.com/api_scrape_item.php?i=bRuV7pSD",
		httpmock.NewStringResponder(200, `foo bar`))

	os.Exit(m.Run())
}
