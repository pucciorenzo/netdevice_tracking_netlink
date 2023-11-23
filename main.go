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

	for {
		select {
		case update := <-ch:
			switch update.Header.Type {
			case syscall.RTM_NEWLINK:
				fmt.Println("New interface added:", update.Link.Attrs().Name)
			case syscall.RTM_DELLINK:
				fmt.Println("Interface deleted:", update.Link.Attrs().Name)
			}
		}
	}
}
