pkg_name=chef-load
pkg_origin=chef
pkg_version="4.0.0"
pkg_license=('Apache-2.0')
pkg_description="A tool that simulates Chef Client API load on a Chef Server and/or a Chef Automate server"
pkg_upstream_url="https://github.com/chef/chef-load/tree/lbaker/fixload"
pkg_bin_dirs=(bin)
pkg_deps=(core/glibc)
pkg_build_deps=(
    core/bash
    core/make
    core/go
)
pkg_binds_optional=(
  [automate]="port"
  [chef-server]="port"
)

do_build() {
    return 0
}

do_install() {
  build_line "copying binary: $PLAN_CONTEXT"
  (
    cd "$PLAN_CONTEXT/../"
    make bin BIN="${pkg_prefix}/bin/" CGO_ENABLED=0 BUILD_COMMIT="" VERSION="${pkg_version}"
  )
}

do_strip() {
    :
}
