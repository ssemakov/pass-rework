//go:build (darwin && !production) || server || mcp || devtools || debug || purego

package main

// Deliberately refuse builds that expose control/debug/server surfaces.
var _ = pass_release_does_not_allow_server_mcp_devtools_debug_or_purego
