/*
 * Tool to monitor a SolrCloud cluster collection.
 *
 * Perform a search and alert on the following metrics:
 * - number of docs
 * - document last_update date
 * - search result
 * - search time
 *
 * returns num docs + search time as perfdata
 */

package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/olorin/nagiosplugin"
	"github.com/rtt/Go-Solr"
)

var (
	host     = flag.String("host", "localhost", "HTTP host of the SOLR service")
	port     = flag.Int("port", 8080, "HTTP port of the SOLR service")
	core     = flag.String("core", "", "Solr core name")
	query    = flag.String("query", "*:*", "Search query in the form key:value")
	sortkey  = flag.String("sortkey", "", "Search result sort key (descending order) - should be a date field")
	minhits  = flag.Int("minhits", 1000000, "Number of expected hits in the response")
	maxqtime = flag.Int("maxqtime", 200, "Max query processing time (ms)")
)

func main() {

	// Initialize the check - this will return an UNKNOWN result
	// until more results are added.
	check := nagiosplugin.NewCheck()
	// If we exit early or panic() we'll still output a result.
	defer check.Finish()

	// parse the cmd line args
	flag.Parse()

	// init the connection
	s, err := solr.Init(*host, *port, *core)
	if err != nil {
		check.Unknownf("Invalid connection parameters")
		log.Fatal(err)
	}

	// Build and perform the query
	q := solr.Query{
		Params: solr.URLParamMap{
			"q": []string{*query},
		},
		Rows: 1,
		Sort: fmt.Sprintf("%s+desc", *sortkey),
	}

	res, err := s.Select(&q)
	if err != nil {
		check.Unknownf("Unable to perform search query, check parameters and connection")
		log.Fatal(err)
	}

	// grab results for ease of use later on
	results := res.Results

	// process the results
	if res.Status != 0 {
		check.Criticalf("Search failed: Invalid Solr response status.")
	}
	if results.Len() == 0 {
		check.Criticalf("Search returned zero documents.")
	}
	if results.NumFound < *minhits {
		check.AddResult(nagiosplugin.WARNING, "Number of documents hits is lower than expected")
	}
	if res.QTime > *maxqtime {
		check.AddPerfDatum("qtime", "ms", float64(res.QTime))
		check.AddPerfDatum("documents", "c", float64(results.NumFound))
		check.Criticalf("Response too slow: %d ms", res.QTime)
	}

	// check the date of the document returned
	// type-assert the result date field to a string
	sortkeydate, ok := results.Get(0).Field(*sortkey).(string)
	if ok {
		lastupdate, err := time.Parse("2006-01-02T15:04:05Z", sortkeydate)
		if err != nil {
			check.Unknownf("Can't parse document's lastupdate field")
			log.Fatal(err)
		}
		// check if last document's date is too old
		if lastupdate.Before(time.Now().Add(-30 * time.Minute)) {
			check.AddPerfDatum("qtime", "ms", float64(res.QTime))
			check.AddPerfDatum("documents", "c", float64(results.NumFound))
			check.Criticalf("Collection update issue: Last document is too old (%s)", lastupdate)
		}

	} else {
		check.Unknownf("Cannot parse date field specified in sortkey")
		log.Fatal("sortkey type error")
	}

	// print the result and exit
	check.AddPerfDatum("qtime", "ms", float64(res.QTime))
	check.AddPerfDatum("documents", "c", float64(results.NumFound))
	check.AddResultf(nagiosplugin.OK, "Search processed in %dms, %d documents found", res.QTime, results.NumFound)
}
