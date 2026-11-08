#import <AppKit/AppKit.h>
#import <WebKit/WebKit.h>
// Pinned Wails3.1.0's navigation delegate has no navigation-action policy.
// A category adds one using public WKNavigationDelegate APIs, without replacing
// its existing lifecycle/scheme handlers. Dependency updates must re-review it.
@interface WebviewWindowDelegate : NSObject
@end
static BOOL trustedURL(NSURL *url){return url&&[[url scheme] isEqualToString:@"wails"]&&[[url host] isEqualToString:@"localhost"]&&!url.user&&!url.password&&!url.port&&([[url path] isEqualToString:@"/"]||[[url path] isEqualToString:@"/index.html"]);}
int pass_navigation_allowed(const void *b,long long n){@autoreleasepool{NSString *s=[[NSString alloc] initWithBytes:b length:(NSUInteger)n encoding:NSUTF8StringEncoding];BOOL ok=trustedURL([NSURL URLWithString:s]);[s release];return ok;}}
@implementation WebviewWindowDelegate (PassNavigation)
- (void)webView:(WKWebView*)view decidePolicyForNavigationAction:(WKNavigationAction*)action decisionHandler:(void(^)(WKNavigationActionPolicy))decision {
 BOOL allowed=action.targetFrame&&action.targetFrame.isMainFrame&&trustedURL(action.request.URL);
 decision(allowed?WKNavigationActionPolicyAllow:WKNavigationActionPolicyCancel);
}
- (WKWebView*)webView:(WKWebView*)view createWebViewWithConfiguration:(WKWebViewConfiguration*)configuration forNavigationAction:(WKNavigationAction*)action windowFeatures:(WKWindowFeatures*)features {return nil;}
@end
