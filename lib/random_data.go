//
// Copyright:: Copyright 2017-2018 Chef Software, Inc.
// License:: Apache License, Version 2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package chef_load

// Random data to use by the generator.go file to create random nodes/ccr's

var (
	ccrStatus = []string{
		"success",
		"failure",
	}

	complianceRoles = [][]string{
		{"base_deb", "apache_deb", "debian-hardening-prod", "dot.role"},
		{"base_linux", "apache_linux", "linux-hardening-prod", "dot.role"},
		{"base_windows", "windows-hardening", "best.role.ever"},
	}

	complianceRecipes = [][]string{
		{"apache_extras", "apache_extras::harden", "java::default", "nagios::fix"},
		{"java", "java::test", "java::security", "nagios::fix"},
		{"linux::harden", "tomcat", "tomcat::setup", "tomcat::configure", "nagios::fix"},
		{"apache::default", "tomcat", "tomcat::setup", "tomcat::configure", "nagios"},
	}

	complianceEnv = []string{
		"DevSec Prod Alpha",
		"DevSec Test Beta",
		"DevSec Dev Delta",
		"DevSec Dev_Gamma",
		"DevSec Prod Zeta",
		"Dot.Comma,Big;\"Trouble",
	}

	environments = []string{
		"arctic",
		"coast",
		"desert",
		"forest",
		"grassland",
		"mountain",
		"swamp",
		"underdark",
		"astral plane",
		"ethereal plane",
		"plane of shadow",
		"feywild",
		"shadowfell",
		"mirror plane",
		"outer space",
		"acceptance-org-proj-master",
	}

	organizations = []string{
		"The Avengers",
		"The Defenders",
		"Justice League of America",
		"The Great Lakes Avengers",
		"The Fantastic Four",
		"Astonishing X-Men",
		"Justice League of Antarctica",
		"The Misfits",
		"The Secret Six",
		"Teen Titans",
		"Watchmen",
		"Guardians of the Galaxy",
		"S.H.I.E.L.D.",
		"Howling Commandos",
		"Ultimates",
		"X-Factor",
		"Uncanny X-Men",
		"Next Wave",
	}

	roles = []string{
		"admin",
		"windows_builder",
		"stash",
		"hamlet",
		"simpsons_guest_character",
		"as_herself",
		"extra_who_died_on_ER",
		"lawyer_in_a_courtroom_procedural",
		"meredith_grey_love_interest",
		"alien_diplomat_on_startrek",
		"zombie_extra",
		"person_eaten_by_zombie_extra",
	}

	platforms = []string{
		"centos",
		"ubuntu",
		"oracle",
		"solaris",
		"windows",
		"mac_os_x",
		"platform 14",
	}

	tags = []string{
		"server",
		"application",
		"middleware",
		"database",
		"network_device",
		"seattle",
		"portland",
		"vancouver",
		"denver",
		"phoenix",
		"dev",
		"alpha",
		"beta",
		"gamma",
		"preprod",
		"prod",
	}

	randCookbooks = []string{
		"tomcat",
		"mysql",
		"etcd",
		"nginx",
		"docker",
		"gems",
		"erlang",
		"chef-vault",
		"fastly",
		"yum",
		"vim",
		"sqitch",
	}

	attributes = map[string]interface{}{
		"attr1": "something",
		"attr2": []string{"some", "other", "complex", "attr"},
		"chef": map[string]interface{}{
			"packages": []string{"a", "b", "c", "x", "y", "z"},
			"channel":  "awesome",
		},
		"install":     "/path/to/installer",
		"application": "delightful",
	}

	resources = []struct {
		Type            string `json:"type"`
		Name            string `json:"name"`
		ID              string `json:"id"`
		Duration        string `json:"duration"`
		Delta           string `json:"delta"`
		IgnoreFailure   bool   `json:"ignore_failure,omitempty"`
		Result          string `json:"result"`
		Status          string `json:"status"`
		CookbookName    string `json:"cookbook_name,omitempty"`
		CookbookVersion string `json:"cookbook_version,omitempty"`
		CookbookType    string `json:"cookbook_type,omitempty"`
		RecipeName      string `json:"recipe_name,omitempty"`
		Conditional     string `json:"conditional,omitempty"`
	}{
		{
			Type:            "chef_gem",
			Name:            "chef-sugar",
			ID:              "chef-sugar",
			Duration:        "138",
			Delta:           "",
			IgnoreFailure:   false,
			Result:          "install",
			Status:          "up-to-date",
			CookbookName:    "gems",
			CookbookVersion: "3.4.0",
			RecipeName:      "default",
		},
		{
			Type:            "execute",
			Name:            "bash script.sh",
			ID:              "bash script.sh",
			Duration:        "100",
			Delta:           "",
			IgnoreFailure:   false,
			Result:          "run",
			Status:          "run",
			CookbookName:    "nginx",
			CookbookVersion: "0.1.0",
			RecipeName:      "install",
		},
		{
			Type:            "chef_gem",
			Name:            "test-kitchen",
			ID:              "test-kitchen",
			Duration:        "523",
			Delta:           "",
			IgnoreFailure:   false,
			Result:          "install",
			Status:          "installed",
			CookbookName:    "gems",
			CookbookVersion: "2.1.0",
			RecipeName:      "default",
		},
		{
			Type:            "execute",
			Name:            "touch /tmp/burger",
			ID:              "touch /tmp/burger",
			Duration:        "10",
			Delta:           "",
			IgnoreFailure:   false,
			Result:          "install",
			Status:          "up-to-date",
			CookbookName:    "erlang",
			CookbookVersion: "0.1.0",
			RecipeName:      "install",
		},
	}

	// For Actions
	entityNameList = []string{
		"nginx",
		"apache",
		"postgres",
		"burger",
		"salsa",
		"sausage",
		"bacon",
	}

	requestorNameList = []string{
		"kyleen",
		"localhost", // This is a chef-zero run
		"knife",
		"rad",
		"lance",
		"afiune",
	}

	sourceFqdns = []string{
		"hostname",
		"localhost",
		"chef.example.com",
		"my.awesome.hostname.com",
	}
)
