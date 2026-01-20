// Package scraper provides HTTP fetching and HTML parsing for VGA Golf state events.
//
// The scraper package fetches the public state events page from vgagolf.org and extracts
// event information including state codes, course names, dates, and cities. It handles
// multiple date formats including multi-line dates, embedded dates in titles, and bracketed
// date formats.
package scraper
