# chef-load Change Log

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
