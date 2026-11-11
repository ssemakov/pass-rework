//go:build darwin && production && native_smoke

#import <AppKit/AppKit.h>
void pass_smoke_key(int tilde) { @autoreleasepool {
  NSWindow *window = NSApp.keyWindow;
  if (!window) return;
  NSEventModifierFlags flags = NSEventModifierFlagCommand;
  if (tilde) flags |= NSEventModifierFlagShift;
  NSString *text = tilde ? @"~" : @"l";
  NSEvent *event = [NSEvent keyEventWithType:NSEventTypeKeyDown location:NSZeroPoint
    modifierFlags:flags timestamp:NSProcessInfo.processInfo.systemUptime
    windowNumber:window.windowNumber context:nil characters:text
    charactersIgnoringModifiers:text isARepeat:NO keyCode:(tilde ? 50 : 37)];
  [NSApp sendEvent:event];
}}
