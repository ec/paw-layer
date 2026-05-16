//go:build darwin && cgo

#import <Cocoa/Cocoa.h>
#import "macos_appkit_bridge.h"

typedef struct {
    double x;
    double y;
    double scale;
    int direction_right;
    int visible;
    int tile_width;
    int tile_height;
    int frame;
} PawLayerCatState;

@interface PawLayerView : NSView
@property(nonatomic, assign) uintptr_t handle;
@property(nonatomic, assign) PawLayerCatState cat;
@property(nonatomic, copy) NSString *spritePath;
@property(nonatomic, strong) NSMutableDictionary<NSString *, NSImage *> *imageCache;
@end

@implementation PawLayerView

- (BOOL)isFlipped { return YES; }

- (void)setFrameSize:(NSSize)newSize {
    [super setFrameSize:newSize];
    pawlayerMacOSViewportChanged(self.handle, (int)newSize.width, (int)newSize.height);
}

- (void)drawRect:(NSRect)dirtyRect {
    (void)dirtyRect;
    [[NSColor clearColor] setFill];
    NSRectFillUsingOperation(self.bounds, NSCompositingOperationClear);

    PawLayerCatState cat = self.cat;
    if (!cat.visible) {
        return;
    }

    CGFloat s = cat.scale > 0 ? cat.scale : 1.0;
    CGFloat x = cat.x;
    CGFloat y = cat.y;

    if (self.spritePath != nil && cat.tile_width > 0 && cat.tile_height > 0) {
        if (self.imageCache == nil) {
            self.imageCache = [NSMutableDictionary dictionary];
        }
        NSImage *image = self.imageCache[self.spritePath];
        if (image == nil) {
            image = [[NSImage alloc] initWithContentsOfFile:self.spritePath];
            if (image != nil) {
                self.imageCache[self.spritePath] = image;
            }
        }
        if (image != nil) {
            [NSGraphicsContext saveGraphicsState];
            [[NSGraphicsContext currentContext] setImageInterpolation:NSImageInterpolationNone];
            NSRect dst = NSMakeRect(x, y, cat.tile_width * s, cat.tile_height * s);
            if (!cat.direction_right) {
                NSAffineTransform *transform = [NSAffineTransform transform];
                [transform translateXBy:(x + cat.tile_width * s) yBy:0];
                [transform scaleXBy:-1 yBy:1];
                [transform concat];
                dst = NSMakeRect(0, y, cat.tile_width * s, cat.tile_height * s);
            }
            NSRect src = NSMakeRect(cat.frame * cat.tile_width, 0, cat.tile_width, cat.tile_height);
            [image drawInRect:dst fromRect:src operation:NSCompositingOperationSourceOver fraction:1.0 respectFlipped:YES hints:nil];
            [NSGraphicsContext restoreGraphicsState];
            return;
        }
    }

    void (^rect)(CGFloat, CGFloat, CGFloat, CGFloat, NSColor *) = ^(CGFloat px, CGFloat py, CGFloat w, CGFloat h, NSColor *color) {
        [color setFill];
        NSRectFill(NSMakeRect(x + px * s, y + py * s, w * s, h * s));
    };

    NSColor *fur = [NSColor colorWithCalibratedRed:0.93 green:0.60 blue:0.24 alpha:1.0];
    NSColor *dark = [NSColor colorWithCalibratedRed:0.35 green:0.20 blue:0.12 alpha:1.0];

    rect(4, 8, 15, 8, fur);
    rect(7, 4, 8, 6, fur);
    rect(7, 2, 2, 3, fur);
    rect(13, 2, 2, 3, fur);

    if (cat.direction_right) {
        rect(18, 7, 4, 2, fur);
        rect(21, 5, 2, 2, fur);
    } else {
        rect(1, 7, 4, 2, fur);
        rect(0, 5, 2, 2, fur);
    }

    rect(6, 16, 3, 2, dark);
    rect(15, 16, 3, 2, dark);
    rect(9, 7, 1, 1, dark);
    rect(13, 7, 1, 1, dark);
    rect(11, 9, 1, 1, dark);
}

@end

@interface PawLayerAppDelegate : NSObject <NSApplicationDelegate>
@property(nonatomic, assign) uintptr_t handle;
@property(nonatomic, assign) int initialWidth;
@property(nonatomic, assign) int initialHeight;
@property(nonatomic, strong) NSWindow *window;
@property(nonatomic, strong) PawLayerView *view;
@end

@implementation PawLayerAppDelegate

- (void)applicationDidFinishLaunching:(NSNotification *)notification {
    (void)notification;
    NSLog(@"paw-layer: applicationDidFinishLaunching");

    NSScreen *screen = [NSScreen mainScreen];
    NSRect frame = screen != nil ? [screen frame] : NSMakeRect(0, 0, self.initialWidth > 0 ? self.initialWidth : 800, self.initialHeight > 0 ? self.initialHeight : 600);

    self.window = [[NSWindow alloc]
        initWithContentRect:frame
                  styleMask:NSWindowStyleMaskBorderless
                    backing:NSBackingStoreBuffered
                      defer:NO];
    [self.window setOpaque:NO];
    [self.window setBackgroundColor:[NSColor clearColor]];
    [self.window setIgnoresMouseEvents:YES];
    [self.window setHasShadow:NO];
    [self.window setLevel:NSScreenSaverWindowLevel];
    [self.window setCollectionBehavior:NSWindowCollectionBehaviorCanJoinAllSpaces |
                                       NSWindowCollectionBehaviorFullScreenAuxiliary |
                                       NSWindowCollectionBehaviorStationary |
                                       NSWindowCollectionBehaviorIgnoresCycle];
    [self.window setReleasedWhenClosed:NO];

    self.view = [[PawLayerView alloc] initWithFrame:NSMakeRect(0, 0, frame.size.width, frame.size.height)];
    self.view.handle = self.handle;
    self.view.wantsLayer = YES;
    self.view.layer.backgroundColor = [[NSColor clearColor] CGColor];
    [self.window setContentView:self.view];
    [self.window orderFrontRegardless];
    NSLog(@"paw-layer: overlay window ordered front %.0fx%.0f", frame.size.width, frame.size.height);

    pawlayerMacOSReady(self.handle, (int)frame.size.width, (int)frame.size.height);
}

- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)sender {
    (void)sender;
    return YES;
}

@end

static NSMutableDictionary<NSNumber *, PawLayerAppDelegate *> *pawlayer_delegates(void) {
    static NSMutableDictionary<NSNumber *, PawLayerAppDelegate *> *delegates = nil;
    static dispatch_once_t onceToken;
    dispatch_once(&onceToken, ^{
        delegates = [NSMutableDictionary dictionary];
    });
    return delegates;
}

int pawlayer_macos_run(uintptr_t handle, int initial_width, int initial_height) {
    @autoreleasepool {
        NSApplication *app = [NSApplication sharedApplication];
        [app setActivationPolicy:NSApplicationActivationPolicyAccessory];

        PawLayerAppDelegate *delegate = [[PawLayerAppDelegate alloc] init];
        delegate.handle = handle;
        delegate.initialWidth = initial_width;
        delegate.initialHeight = initial_height;
        pawlayer_delegates()[@(handle)] = delegate;
        [app setDelegate:delegate];
        [app finishLaunching];
        [app run];
        [pawlayer_delegates() removeObjectForKey:@(handle)];
    }
    return 0;
}

void pawlayer_macos_set_cat(uintptr_t handle, double x, double y, double scale, int direction_right, int visible) {
    dispatch_async(dispatch_get_main_queue(), ^{
        PawLayerAppDelegate *delegate = pawlayer_delegates()[@(handle)];
        if (delegate == nil || delegate.view == nil) {
            return;
        }
        PawLayerCatState cat = {
            .x = x,
            .y = y,
            .scale = scale,
            .direction_right = direction_right,
            .visible = visible,
            .tile_width = 0,
            .tile_height = 0,
            .frame = 0,
        };
        delegate.view.cat = cat;
        delegate.view.spritePath = nil;
        [delegate.view setNeedsDisplay:YES];
    });
}

void pawlayer_macos_set_sprite(uintptr_t handle, const char *path, double x, double y, double scale, int tile_width, int tile_height, int frame, int direction_right, int visible) {
    NSString *spritePath = path != NULL ? [NSString stringWithUTF8String:path] : nil;
    dispatch_async(dispatch_get_main_queue(), ^{
        PawLayerAppDelegate *delegate = pawlayer_delegates()[@(handle)];
        if (delegate == nil || delegate.view == nil) {
            return;
        }
        PawLayerCatState cat = {
            .x = x,
            .y = y,
            .scale = scale,
            .direction_right = direction_right,
            .visible = visible,
            .tile_width = tile_width,
            .tile_height = tile_height,
            .frame = frame,
        };
        delegate.view.cat = cat;
        delegate.view.spritePath = spritePath;
        [delegate.view setNeedsDisplay:YES];
    });
}

void pawlayer_macos_quit(uintptr_t handle) {
    dispatch_async(dispatch_get_main_queue(), ^{
        PawLayerAppDelegate *delegate = pawlayer_delegates()[@(handle)];
        if (delegate != nil && delegate.window != nil) {
            [delegate.window close];
        }
        [NSApp terminate:nil];
    });
}
