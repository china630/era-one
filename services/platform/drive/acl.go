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

// IsLocked reports whether the object currently has a lock holder.
func IsLocked(obj Object) bool {
	return strings.TrimSpace(obj.LockedBy) != ""
}

// IsLocker reports whether principal holds the current lock.
func IsLocker(obj Object, p Principal) bool {
	return IsLocked(obj) && p.UserID != "" && obj.LockedBy == p.UserID
}

// IsOwner reports whether principal is the object owner.
func IsOwner(obj Object, p Principal) bool {
	return p.UserID != "" && obj.OwnerUserID == p.UserID
}

// CanMutateWhileLocked allows rename/move/content updates when unlocked or by the locker.
func CanMutateWhileLocked(obj Object, p Principal) bool {
	return !IsLocked(obj) || IsLocker(obj, p)
}

// CanUnlock allows unlock by the locker or the owner.
func CanUnlock(obj Object, p Principal) bool {
	return IsLocker(obj, p) || IsOwner(obj, p)
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
