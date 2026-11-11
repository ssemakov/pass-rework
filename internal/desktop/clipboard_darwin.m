#import <AppKit/AppKit.h>
static long long lastOwned=0;
long long pass_clip_write(const void *bytes,long long len){@autoreleasepool{
 NSString *text=[[NSString alloc] initWithBytes:bytes length:(NSUInteger)len encoding:NSUTF8StringEncoding];if(!text)return 0;
 NSPasteboard *p=[NSPasteboard generalPasteboard];[p clearContents];[p setString:@"" forType:@"org.nspasteboard.ConcealedType"];
 BOOL ok=[p setString:text forType:NSPasteboardTypeString];[text release];lastOwned=ok?[p changeCount]:0;return lastOwned;
}}
void pass_clip_clear(long long token){@autoreleasepool{NSPasteboard *p=[NSPasteboard generalPasteboard];if([p changeCount]==token)[p clearContents];}}

void pass_clip_clear_last(void){pass_clip_clear(lastOwned);lastOwned=0;}

void pass_disable_restoration(void){@autoreleasepool{for(NSWindow *window in [NSApp windows]){if([window isKindOfClass:NSClassFromString(@"WebviewWindow")])[window setRestorable:NO];}[[NSUserDefaults standardUserDefaults] setBool:NO forKey:@"NSQuitAlwaysKeepsWindows"];}}
