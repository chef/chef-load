//
// Copyright:: Copyright 2018 Chef Software, Inc.
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

// The version is composed by three elements:
//
// 1) Version (semver)
// 2) TODO: SHA (The git sha from HEAD)
// 3) TODO: Built time (Time we built our binary)
//
// We pass all these parameters throught the linker:
// => go build -ldflags "-X config.VERSION=x.x.x -X config.SHA=SHA" ...
//
// where 'x.x.x' is the version from the VERSION file
//
// The following variables are just stakeholders:
// -- DO NOT MANUALLY MODIFY THEM --
var (
	VERSION string = "4.0.0"
	//BUILD_TIME string = "DATE"
	//SHA        string = "4f1un3"
)
