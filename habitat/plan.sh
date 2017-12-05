pkg_name=chef-load
pkg_origin=chef
pkg_version="3.0.0"
pkg_license=('UNLICENSED')
# This plan does not build the package from source, but instead downloads the
# statically linked binary associated with the $pkg_version from GitHub
# releases.
pkg_source="https://github.com/chef/$pkg_name/releases/download/v$pkg_version/${pkg_name}_${pkg_version}_Linux_64bit"
pkg_shasum="0ad6612192b9bb00089591e6a36d937b14e465500ca721be778e8ab81ac617f1"
pkg_bin_dirs=(bin)
pkg_binds_optional=(
  [automate]="port"
  [chef-server]="port"
)
pkg_description="A tool that simulates Chef Client API load on a Chef Server and/or a Chef Automate server"
pkg_upstream_url="https://github.com/chef/chef-load"

do_unpack() {
  return 0
}

do_build() {
  return 0
}

do_install() {
  install -v -m 0755 \
    "$HAB_CACHE_SRC_PATH/${pkg_name}_${pkg_version}_Linux_64bit" \
    "$pkg_prefix/bin/$pkg_name"
}

do_strip() {
  return 0
}
