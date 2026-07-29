package drive

import (
	"strings"
)

// CanRead returns true if principal may read the object.
func CanRead(obj Object, p Principal) bool {
	if p.UserID != "" && obj.OwnerUserID == p.UserID {
		return true
	}
	for _, e := range obj.ACL {
		if !principalMatches(e.Principal, p) {
			continue
		}
		switch e.Role {
		case ACLRoleOwner, ACLRoleRead, ACLRoleWrite:
			return true
		}
	}
	return false
}

// CanWrite returns true if principal may modify the object or ACL.
func CanWrite(obj Object, p Principal) bool {
	if p.UserID != "" && obj.OwnerUserID == p.UserID {
		return true
	}
	for _, e := range obj.ACL {
		if !principalMatches(e.Principal, p) {
			continue
		}
		switch e.Role {
		case ACLRoleOwner, ACLRoleWrite:
			return true
		}
	}
	return false
}

func principalMatches(principal string, p Principal) bool {
	principal = strings.TrimSpace(principal)
	if principal == "" {
		return false
	}
	parts := strings.SplitN(principal, ":", 2)
	if len(parts) != 2 {
		return false
	}
	switch parts[0] {
	case "user":
		return p.UserID != "" && parts[1] == p.UserID
	case "group":
		for _, g := range p.Groups {
			if g == parts[1] {
				return true
			}
		}
	case "tenant":
		return parts[1] == "members" && p.TenantID != "" && p.TenantID == objTenant(p)
	}
	return false
}

func objTenant(p Principal) string {
	return p.TenantID
}

// DefaultOwnerACL seeds owner-only ACL (tenant-wide grants via PATCH).
func DefaultOwnerACL(_ string, ownerID string) []ACLEntry {
	return []ACLEntry{{Principal: "user:" + ownerID, Role: ACLRoleOwner}}
}
