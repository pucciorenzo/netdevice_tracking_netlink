package main

import (
	"fmt"
	"syscall"

	"github.com/vishvananda/netlink"
)

func main() {
	// Create a channel to receive notifications
	ch := make(chan netlink.LinkUpdate)
	done := make(chan struct{})

	// Subscribe to the link updates
	if err := netlink.LinkSubscribe(ch, done); err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Listening for link updates")
	seen := make(map[string]bool)

	for {
		select {
		case update := <-ch:
			if update.Header.Type == syscall.RTM_DELLINK {
				fmt.Println("Interface removed:", update.Link.Attrs().Name)
				delete(seen, update.Link.Attrs().Name)
			}
			_, exists := seen[update.Link.Attrs().Name]
			if !exists {
				if update.Header.Type == syscall.RTM_NEWLINK { // ip link add
					switch update.Link.Type() {
					case "veth":
						fmt.Println("New interface veth added:", update.Link.Attrs().Name)
						seen[update.Link.Attrs().Name] = true
					case "dummy":
						fmt.Println("New interface dummy added:", update.Link.Attrs().Name)
						seen[update.Link.Attrs().Name] = true
						// add any other interface types here
					}
				}
			}
		}
	}
}
