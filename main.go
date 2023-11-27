package main

import (
	"fmt"
	"net"
	"syscall"

	"github.com/vishvananda/netlink"
)

func main() {
	// Create a channel to receive notifications
	chLink := make(chan netlink.LinkUpdate)
	doneLink := make(chan struct{})
	defer close(doneLink)

	chAddr := make(chan netlink.AddrUpdate)
	doneAddr := make(chan struct{})
	defer close(doneAddr)

	// Subscribe to the address updates
	if err := netlink.AddrSubscribe(chAddr, doneAddr); err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Subscribe to the link updates
	if err := netlink.LinkSubscribe(chLink, doneLink); err != nil {
		fmt.Println("Error:", err)
		return
	}
	newlyCreated := make(map[string]bool)
	// Create a map to keep track of all interfaces
	interfaces := make(map[string]bool)
	links, err := netlink.LinkList()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	for _, link := range links {
		interfaces[link.Attrs().Name] = true
	}
	fmt.Println("Monitoring started. Press Ctrl+C to stop it.")
	for {
		select {
		case updateLink := <-chLink:
			if updateLink.Header.Type == syscall.RTM_DELLINK { // ip link del
				fmt.Println("Interface removed:", updateLink.Link.Attrs().Name)
				delete(interfaces, updateLink.Link.Attrs().Name)
				delete(newlyCreated, updateLink.Link.Attrs().Name)
			}
			_, exists := interfaces[updateLink.Link.Attrs().Name]
			if !exists && updateLink.Header.Type == syscall.RTM_NEWLINK { // ip link add
				switch updateLink.Link.Type() {
				case "veth":
					fmt.Println("New veth interface added:", updateLink.Link.Attrs().Name)
				case "dummy":
					fmt.Println("New dummy interface added:", updateLink.Link.Attrs().Name)
				default:
					fmt.Println("New interface added:", updateLink.Link.Attrs().Name)
				}
				interfaces[updateLink.Link.Attrs().Name] = true
				newlyCreated[updateLink.Link.Attrs().Name] = true
			} else if updateLink.Header.Type == syscall.RTM_NEWLINK { // ip link set
				if updateLink.Link.Attrs().Flags&net.FlagUp != 0 {
					fmt.Println("Interface", updateLink.Link.Attrs().Name, "is up")
					delete(newlyCreated, updateLink.Link.Attrs().Name)
				} else if !newlyCreated[updateLink.Link.Attrs().Name] {
					fmt.Println("Interface", updateLink.Link.Attrs().Name, "is down")
				}
			}
		case updateAddr := <-chAddr:
			iface, err := net.InterfaceByIndex(updateAddr.LinkIndex)
			if err != nil {
				fmt.Println("Address (", updateAddr.LinkAddress.IP, ") removed from the deleted interface")
				continue
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
