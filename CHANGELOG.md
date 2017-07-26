# chef-load Change Log

## 2.1.0 (2017-07-26)

* Send data-collector data through Chef Server if it's available
* http.Response handling improvements
* Improve apiRequest capabilities and use it whenever possible
* Make sure the node object's chef environment is set to chef-load's value
* Correct the order of some Chef Client API requests
* Use the default run list if a role doesn't have an environment specific run list

## 2.0.0 (2017-07-18)

* Add habitat plan (thanks @smith!)
* Use number of nodes and interval instead of runs/minute and interval as parameters
* Create client object during first chef client run (thanks @nsdavidson!)
* Add Apache License and Copyright info

## 1.0.0 (2017-06-21)

* Remove the bootstrap process
* Set each node's essential normal attributes
* Fix end_time format for compliance status report
* Redesign how API load is defined.
* Refactor when expandedRunList is calculated in data-collector mode
* Ensure run_list and expanded_run_list are properly set for converge status report
* Use FQDN for converge status report's chef_server_fqdn instead of the full URL
* If complianceJSON is empty don't send compliance status to Automate server
* Set compliance status report's `type` field to `inspec_report`
* Delete top level `controls` key if it exists in compliance status report
* Start numbering nodes at 1 instead of 0
* Add ability to set prefix at command line
* Overhaul the config file to simplify and provide clarity
* Minor change to output
* Add timestamps to output

## 0.5.0 (2017-06-20)

* read complete http response when making API requests
* Update data_collector message_version to 1.1.0
* Use organization_name field in data_collector messages
* Add ability to use converge status JSON for data collector resources field
* Send compliance status report to data collector
* Set default values for essential automatic attributes

## 0.4.0 (2016-10-24)

* Add option to set the node's chef_environment
* Improve run_list handling and cookbook dependency solving
* Add enable_reporting config option
* Add ability to apply load to an Automate server's data-collector API endpoint
* Add a mode option to choose between "chef-client" and "data-collector" modes

## 0.3.0 (2016-07-11)

* Add API requests to /reports endpoint

## 0.2.0 (2016-07-10)

* Add splay
* Add bootstrap_nodes_concurrency throttle setting
* Use ohai json file for automatic attributes ([juozasg](https://github.com/juozasg))
* Update ohai_time in node's automatic attributes ([stevendanna](https://github.com/stevendanna))
* Fail early if config.ClientKey isn't readable ([stevendanna](https://github.com/stevendanna))

## 0.1.1 (2015-12-10)

* Make Interval work when Runs is set to 0

## 0.1.0 (2015-12-10)

* Initial release of chef-load
