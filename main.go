package main

import (
	"net"
	"syscall"

	"github.com/vishvananda/netlink"
	"k8s.io/klog/v2"
)

// InterfacesMonitoring starts the monitoring of the network interfaces.
// If there is a change in the network interfaces, it will send a message to the channel.
// With the options, you can choose to monitor only the link, address, or route changes (default: all options are true).
func main() {
	// Create channels to receive notifications for link, address, and route changes
	chLink := make(chan netlink.LinkUpdate)
	chAddr := make(chan netlink.AddrUpdate)
	chRoute := make(chan netlink.RouteUpdate)

	chDoneLink := make(chan struct{})
	defer close(chDoneLink)
	chDoneAddr := make(chan struct{})
	defer close(chDoneAddr)
	chDoneRoute := make(chan struct{})
	defer close(chDoneRoute)

	// Create maps to keep track of interfaces
	interfaces := make(map[string]bool)

	// Subscribe to the link updates
	if err := netlink.LinkSubscribe(chLink, chDoneLink); err != nil {
		klog.Error(err)
		return
	}

	// Get the list of existing links and add them to the interfaces map
	links, err := netlink.LinkList()
	if err != nil {
		klog.Error(err)
		return
	}
	for _, link := range links {
		interfaces[link.Attrs().Name] = true
	}

	// Subscribe to the address updates
	if err := netlink.AddrSubscribe(chAddr, chDoneAddr); err != nil {
		klog.Error(err)
		return
	}

	// Subscribe to the route updates
	if err := netlink.RouteSubscribe(chRoute, chDoneRoute); err != nil {
		klog.Error(err)
		return
	}

	// Start an infinite loop to handle the notifications
	for {
		select {
		case updateLink := <-chLink:
			handleLinkUpdate(&updateLink, interfaces)
		case updateAddr := <-chAddr:
			handleAddrUpdate(&updateAddr)
		case updateRoute := <-chRoute:
			handleRouteUpdate(&updateRoute)
		}
	}
}

func handleLinkUpdate(updateLink *netlink.LinkUpdate, interfaces map[string]bool) {
	switch {
	case updateLink.Header.Type == syscall.RTM_DELLINK:
		// Link has been removed
		klog.Infof("Interface removed: %s", updateLink.Link.Attrs().Name)
		delete(interfaces, updateLink.Link.Attrs().Name)
	case !interfaces[updateLink.Link.Attrs().Name] && updateLink.Header.Type == syscall.RTM_NEWLINK:
		// New link has been added
		klog.Infof("Interface added: %s", updateLink.Link.Attrs().Name)
		interfaces[updateLink.Link.Attrs().Name] = true
	case updateLink.Header.Type == syscall.RTM_NEWLINK:
		// Link has been modified
		if updateLink.Link.Attrs().Flags&net.FlagUp != 0 {
			klog.Infof("Interface %s is up", updateLink.Link.Attrs().Name)
		} else {
			klog.Infof("Interface %s is down", updateLink.Link.Attrs().Name)
		}
	default:
		klog.Warning("Unknown link update type.")
	}
}

func handleAddrUpdate(updateAddr *netlink.AddrUpdate) {
	iface, err := net.InterfaceByIndex(updateAddr.LinkIndex)
	if err != nil {
		// This case is not a real error, it happens when an up interface is removed, so the address is removed too,
		// so there is no need to call the reconcile since is already called by the interface update.
		klog.Infof("Address (%s) removed from the deleted interface", updateAddr.LinkAddress.IP)
		return
	}
	if updateAddr.NewAddr {
		// New address has been added
		klog.Infof("New address (%s) added to the interface: %s", updateAddr.LinkAddress.IP, iface.Name)
	} else {
		// Address has been removed
		klog.Infof("Address (%s) removed from the interface: %s", updateAddr.LinkAddress.IP, iface.Name)
	}
}

func handleRouteUpdate(updateRoute *netlink.RouteUpdate) {
	if updateRoute.Type == syscall.RTM_NEWROUTE {
		// New route has been added
		klog.Infof("New route added: %s", updateRoute.Route.Dst)
	} else if updateRoute.Type == syscall.RTM_DELROUTE {
		// Route has been removed
		klog.Infof("Route removed: %s", updateRoute.Route.Dst)
	}
}
