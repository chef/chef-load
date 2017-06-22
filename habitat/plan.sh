pkg_name=chef-load
pkg_origin=jeremiahsnapp
pkg_version="1.0.0"
pkg_license=('UNLICENSED')
# This plan does not build the package from source, but instead downloads the
# statically linked binary associated with the $pkg_version from GitHub
# releases.
pkg_source="https://github.com/jeremiahsnapp/$pkg_name/releases/download/v$pkg_version/${pkg_name}_${pkg_version}_Linux_64bit"
pkg_shasum="32bd6f451b5b83fc5a7c2790e2d7a040f596f5c79b6c88a581542920cc5eb3d9"
pkg_bin_dirs=(bin)
pkg_binds_optional=(
  [automate]="port"
  [chef-server]="port"
)
pkg_description="A tool for simulating load on a Chef Server"
pkg_upstream_url="https://github.com/jeremiahsnapp/chef-load"

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
