// Package session — re-export privilegedsession for dlp backward compat.
package session

import "era/services/platform/privilegedsession"

type Record = privilegedsession.Record
type Alert = privilegedsession.Alert
type Store = privilegedsession.Store

func NewStore() *Store {
	return privilegedsession.NewStore()
}
