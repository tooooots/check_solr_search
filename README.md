## check_solr_Search ##

 Monitoring check for SolrCloud cluster:

 Performs a search and alert on the following metrics:
  - number of docs
  - document last_update date
  - search result
  - search time

 Returns perfdata: num docs (counter) + search time (ms)

This check should work with the typical monitoring tools (icinga, sensu, etc)

