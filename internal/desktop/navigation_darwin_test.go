package desktop

import "testing"

func TestNativeNavigationPolicy(t *testing.T) {
	for _, url := range []string{"wails://localhost/", "wails://localhost/index.html"} {
		if !NavigationAllowed(url) {
			t.Fatal("trusted root rejected", url)
		}
	}
	for _, url := range []string{"https://example.test/", "data:text/html,test", "file:///tmp/index.html", "wails://localhost/other", "wails://evil/", "wails://name@localhost/", "wails://localhost:123/"} {
		if NavigationAllowed(url) {
			t.Fatal("untrusted navigation allowed", url)
		}
	}
}
