package chef

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type SearchService struct {
	client *Client
}

// SearchQuery Is the struct for holding a query request
type SearchQuery struct {
	// The index you want to search
	Index string

	// The query you want to execute. This is the 'chef' query ex: 'chef_environment:prod'
	Query string

	// Sort order you want the search results returned
	SortBy string

	// Starting position for search
	Start int

	// Number of rows to return
	Rows int
}

// String implements the Stringer Interface for the SearchQuery
func (q SearchQuery) String() string {
	return fmt.Sprintf("%s?q=%s&rows=%d&sort=%s&start=%d", q.Index, q.Query, q.Rows, q.SortBy, q.Start)
}

// SearchResult
type SearchResult struct {
	Total int
	Start int
	Rows  []interface{}
}

// JSearchResult will return a slice of json.RawMessage which can then
// be json.Unmarshaled to any of the chef-like objects (Role/Node/etc)
type JSearchResult struct {
	Total int
	Start int
	Rows  []SearchRow
}

type SearchRow struct {
	Url  string
	Data json.RawMessage
}

var inc = 1000

func (e SearchService) PageSize(setting int) {
	inc = setting
}

// Do will execute the search query on the client
func (q SearchQuery) Do(client *Client) (res SearchResult, err error) {
	fullUrl := fmt.Sprintf("search/%s", q)
	err = client.magicRequestDecoder("GET", fullUrl, nil, &res)
	return
}

// DoJSON will execute the search query on the client and return
// rawJSON formatted results
func (q SearchQuery) DoJSON(client *Client) (res JSearchResult, err error) {
	fullUrl := fmt.Sprintf("search/%s", q)
	err = client.magicRequestDecoder("GET", fullUrl, nil, &res)
	return
}

// DoPartial will execute the search query on the client with partial mapping
func (q SearchQuery) DoPartial(client *Client, params map[string]interface{}) (res SearchResult, err error) {
	fullUrl := fmt.Sprintf("search/%s", q)

	body, err := JSONReader(params)
	if err != nil {
		debug("Problem encoding params for body %v", err.Error())
		return
	}

	err = client.magicRequestDecoder("POST", fullUrl, body, &res)
	return
}

// DoPartialJSON will execute the search query on the client with partial mapping and return raw JSON results
func (q SearchQuery) DoPartialJSON(client *Client, params map[string]interface{}) (res JSearchResult, err error) {
	fullUrl := fmt.Sprintf("search/%s", q)

	body, err := JSONReader(params)
	if err != nil {
		debug("Problem encoding params for body %v", err.Error())
		return
	}

	err = client.magicRequestDecoder("POST", fullUrl, body, &res)
	return
}

// NewSearch is a constructor for a SearchQuery struct. This is used by other search service methods to perform search requests on the server
func (e SearchService) NewQuery(idx, statement string) (query SearchQuery, err error) {
	// validate statement
	if !strings.Contains(statement, ":") {
		err = errors.New("statement is malformed")
		return
	}

	query = SearchQuery{
		Index: idx,
		Query: statement,
		// These are the defaults in chef: https://github.com/opscode/chef/blob/master/lib/chef/search/query.rb#L102-L105
		SortBy: "X_CHEF_id_CHEF_X asc",
		Start:  0,
		Rows:   inc,
	}

	return
}

// Exec runs the query on the index passed in. This is a helper method. If you want more control over the query  use NewQuery and its Do() method.
// BUG(spheromak): Should we use Exec or SearchQuery.Do() or have both ?
func (e SearchService) Exec(idx, statement string) (res SearchResult, err error) {
	//  Copy-paste here till We decide which way to go with Exec vs Do
	if !strings.Contains(statement, ":") {
		err = errors.New("statement is malformed")
		return
	}

	query := SearchQuery{
		Index: idx,
		Query: statement,
		// These are the defaults in chef: https://github.com/opscode/chef/blob/master/lib/chef/search/query.rb#L102-L105
		SortBy: "X_CHEF_id_CHEF_X asc",
		Start:  0,
		Rows:   inc,
	}

	res, err = query.Do(e.client)
	if err != nil {
		return
	}
	start := res.Start
	total := res.Total

	for start+inc <= total {
		query.Start = query.Start + inc
		start = query.Start
		ares, err := query.Do(e.client)
		if err != nil {
			return res, err
		}
		res.Rows = append(res.Rows, ares.Rows...)
	}
	return
}

// PartialExec Executes a partial search based on passed in params and the query.
func (e SearchService) PartialExec(idx, statement string, params map[string]interface{}) (res SearchResult, err error) {
	query := SearchQuery{
		Index: idx,
		Query: statement,
		// These are the defaults in chef: https://github.com/opscode/chef/blob/master/lib/chef/search/query.rb#L102-L105
		// SortBy: "X_CHEF_id_CHEF_X asc",
		SortBy: "X_CHEF_id_CHEF_X asc",
		Start:  0,
		Rows:   inc,
	}

	fullUrl := fmt.Sprintf("search/%s", query)
	body, err := JSONSeeker(params)
	if err != nil {
		debug("Problem encoding params for body")
		return
	}

	err = e.client.magicRequestDecoder("POST", fullUrl, body, &res)
	if err != nil {
		return
	}

	start := res.Start
	// the total rows available for this query across all pages
	total := res.Total
	paged_res := SearchResult{}

	for start+inc <= total {
		query.Start = query.Start + inc
		start = query.Start
		body.Seek(0, io.SeekStart)
		if err != nil {
			fmt.Printf("Seek error %+v\n", err)
			return
		}
		fullUrl := fmt.Sprintf("search/%s", query)
		err = e.client.magicRequestDecoder("POST", fullUrl, body, &paged_res)
		if err != nil {
			fmt.Printf("Partial search error %+v\n", err)
			return
		}
		// add this page of results to the primary SearchResult instance
		res.Rows = append(res.Rows, paged_res.Rows...)
	}
	return
}

// ExecJSON runs the query on the index passed in. This is a helper method. If you want more control over the query use NewQuery and its Do() method.
func (e SearchService) ExecJSON(idx, statement string) (res JSearchResult, err error) {
	//  Copy-paste here till We decide which way to go with Exec vs Do
	if !strings.Contains(statement, ":") {
		err = errors.New("statement is malformed")
		return
	}

	query := SearchQuery{
		Index: idx,
		Query: statement,
		// These are the defaults in chef: https://github.com/opscode/chef/blob/master/lib/chef/search/query.rb#L102-L105
		SortBy: "X_CHEF_id_CHEF_X asc",
		Start:  0,
		Rows:   inc,
	}

	res, err = query.DoJSON(e.client)
	if err != nil {
		return
	}
	start := res.Start
	total := res.Total

	for start+inc <= total {
		query.Start = query.Start + inc
		start = query.Start
		ares, err := query.DoJSON(e.client)
		if err != nil {
			return res, err
		}
		res.Rows = append(res.Rows, ares.Rows...)
	}
	return
}

// PartialExecJSON Executes a partial search based on passed in params and the query.
func (e SearchService) PartialExecJSON(idx, statement string, params map[string]interface{}) (res JSearchResult, err error) {
	query := SearchQuery{
		Index: idx,
		Query: statement,
		// These are the defaults in chef: https://github.com/opscode/chef/blob/master/lib/chef/search/query.rb#L102-L105
		// SortBy: "X_CHEF_id_CHEF_X asc",
		SortBy: "X_CHEF_id_CHEF_X asc",
		Start:  0,
		Rows:   inc,
	}

	fullUrl := fmt.Sprintf("search/%s", query)
	body, err := JSONSeeker(params)
	if err != nil {
		debug("Problem encoding params for body")
		return
	}

	err = e.client.magicRequestDecoder("POST", fullUrl, body, &res)
	if err != nil {
		return
	}

	start := res.Start
	// the total rows available for this query across all pages
	total := res.Total
	paged_res := JSearchResult{}

	for start+inc <= total {
		query.Start = query.Start + inc
		start = query.Start
		body.Seek(0, io.SeekStart)
		if err != nil {
			fmt.Printf("Seek error %+v\n", err)
			return
		}
		fullUrl := fmt.Sprintf("search/%s", query)
		err = e.client.magicRequestDecoder("POST", fullUrl, body, &paged_res)
		if err != nil {
			fmt.Printf("Partial search error %+v\n", err)
			return
		}
		// add this page of results to the primary SearchResult instance
		res.Rows = append(res.Rows, paged_res.Rows...)
	}
	return
}

// Chef API docs: https://docs.chef.io/api_chef_server/#get-46
func (e SearchService) Indexes() (data map[string]string, err error) {
	err = e.client.magicRequestDecoder("GET", "search", nil, &data)
	return
}
