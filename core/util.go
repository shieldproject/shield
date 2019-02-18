package core

import (
	"net"

	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/db"
)

func IsValidTenantRole(role string) bool {
	return role == "admin" || role == "engineer" || role == "operator"
}

func IsValidSystemRole(role string) bool {
	return role == "admin" || role == "manager" || role == "engineer"
}

func ip() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "(unknown)"
	}

	var v4ip, v6ip string
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP

			found := false
			switch addr.(type) {
			case *net.IPNet:
				ip = addr.(*net.IPNet).IP
				found = !ip.IsLoopback()
			case *net.IPAddr:
				ip = addr.(*net.IPAddr).IP
				found = !ip.IsLoopback()
			}
			isv4 := ip.To4() != nil
			if !found || (!isv4 && v6ip != "") || (isv4 && v4ip != "") {
				continue
			}

			if isv4 {
				v4ip = ip.String()
			} else {
				v6ip = ip.String()
			}
		}
	}

	if v4ip != "" {
		return v4ip
	}
	if v6ip != "" {
		return v6ip
	}
	return "(unknown)"
}

func (c *Core) DeltaIncrease(filter *db.ArchiveFilter) (int64, error) {
	delta_increase, err := c.db.ArchiveStorageFootprint(&db.ArchiveFilter{
		ForStore:   filter.ForStore,
		ForTenant:  filter.ForTenant,
		Before:     filter.Before,
		After:      filter.After,
		WithStatus: []string{"valid"},
	})
	if err != nil {
		log.Errorf("Failed to get archive stats for daily storage statistics: %s", err)
		return -1, err
	}

	delta_purged, err := c.db.ArchiveStorageFootprint(&db.ArchiveFilter{
		ForStore:      filter.ForStore,
		ForTenant:     filter.ForTenant,
		ExpiresBefore: filter.ExpiresBefore,
		ExpiresAfter:  filter.ExpiresAfter,
		WithStatus:    []string{"purged"},
	})
	if err != nil {
		log.Errorf("Failed to get archive stats for daily storage statistics: %s", err)
		return -1, err
	}
	return (delta_increase - delta_purged), nil
}
