pkg_name=chef-load
pkg_origin=chef
pkg_version="4.0.0"
pkg_license=('Apache-2.0')
pkg_description="A tool that simulates Chef Client API load on a Chef Server and/or a Chef Automate server"
pkg_upstream_url="https://github.com/chef/chef-load"
pkg_bin_dirs=(bin)
pkg_build_deps=(core/dep)
pkg_deps=(core/glibc)
pkg_binds_optional=(
  [automate]="port"
  [chef-server]="port"
)
pkg_scaffolding=core/scaffolding-go
scaffolding_go_base_path=github.com/chef
scaffolding_go_build_deps=()

do_download(){
  export scaffolding_go_pkg_path="$scaffolding_go_workspace_src/$scaffolding_go_base_path/$pkg_name"
  build_line "Getting Go dependencies (dep ensure)"
  pushd $scaffolding_go_pkg_path >/dev/null
    dep ensure
  popd >/dev/null
}
