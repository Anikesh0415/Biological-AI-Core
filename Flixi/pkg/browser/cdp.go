package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
)

// CDP uses the Chrome DevTools Protocol to natively control the browser without physical clicks.
// It assumes Chrome is started with --remote-debugging-port=9222.

func getContext() (context.Context, context.CancelFunc) {
	// connect to existing browser on port 9222
	allocatorCtx, cancel1 := chromedp.NewRemoteAllocator(context.Background(), "ws://127.0.0.1:9222/")
	ctx, cancel2 := chromedp.NewContext(allocatorCtx)
	
	// Optional timeout
	ctx, cancel3 := context.WithTimeout(ctx, 10*time.Second)
	
	return ctx, func() {
		cancel3()
		cancel2()
		cancel1()
	}
}

func Navigate(url string) error {
	ctx, cancel := getContext()
	defer cancel()

	fmt.Printf("CDP: Navigating to %s\n", url)
	return chromedp.Run(ctx,
		chromedp.Navigate(url),
	)
}

func Click(selector string) error {
	ctx, cancel := getContext()
	defer cancel()

	fmt.Printf("CDP: Clicking selector %s\n", selector)
	return chromedp.Run(ctx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Click(selector, chromedp.ByQuery),
	)
}

func Type(selector, text string) error {
	ctx, cancel := getContext()
	defer cancel()

	fmt.Printf("CDP: Typing into %s\n", selector)
	return chromedp.Run(ctx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.SendKeys(selector, text, chromedp.ByQuery),
	)
}
