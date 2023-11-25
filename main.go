package main

import (
	"fmt"
	"net"
	"syscall"

	"github.com/vishvananda/netlink"
)

func main() {
	// Create a channel to receive notifications
	ch := make(chan netlink.LinkUpdate) // change name to chLink
	done := make(chan struct{})

	chAddr := make(chan netlink.AddrUpdate)
	doneAddr := make(chan struct{})

	// Subscribe to the address updates
	if err := netlink.AddrSubscribe(chAddr, doneAddr); err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Subscribe to the link updates
	if err := netlink.LinkSubscribe(ch, done); err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Listening for link updates")
	// Create a map to keep track of all existing interfaces
	interfaces := make(map[string]bool)
	links, err := netlink.LinkList()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	for _, link := range links {
		interfaces[link.Attrs().Name] = true
	}

	for {
		select {
		case update := <-ch: // change name to updateLink
			if update.Header.Type == syscall.RTM_DELLINK { // ip link del
				fmt.Println("Interface removed:", update.Link.Attrs().Name)
				delete(interfaces, update.Link.Attrs().Name)
				continue
			}
			_, exists := interfaces[update.Link.Attrs().Name]
			if !exists && update.Header.Type == syscall.RTM_NEWLINK { // ip link add
				switch update.Link.Type() {
				case "veth":
					fmt.Println("New interface veth type added:", update.Link.Attrs().Name)
					interfaces[update.Link.Attrs().Name] = true
				case "dummy":
					fmt.Println("New interface dummy type added:", update.Link.Attrs().Name)
					interfaces[update.Link.Attrs().Name] = true
				default:
					fmt.Println("New interface added:", update.Link.Attrs().Name)
					interfaces[update.Link.Attrs().Name] = true
					// add any other interface types here
				}
			} else if exists && update.Header.Type == syscall.RTM_NEWLINK { // ip link set
				if update.Link.Attrs().Flags&net.FlagUp != 0 {
					fmt.Println("Interface up:", update.Link.Attrs().Name)
				} else {
					fmt.Println("Interface down:", update.Link.Attrs().Name)
				}
			}
		case updateAddr := <-chAddr:
			iface, err := net.InterfaceByIndex(updateAddr.LinkIndex)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			switch updateAddr.NewAddr {
			case true: // ip addr add
				fmt.Println("New address (", updateAddr.LinkAddress.IP, ") added to the interface:", iface.Name)
			case false: // ip addr del
				fmt.Println("Address (", updateAddr.LinkAddress.IP, ") removed from the interface:", iface.Name)
			}
		}
	}
}
