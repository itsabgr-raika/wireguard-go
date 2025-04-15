package device

import (
	"net/netip"
)

const AnyProto = 255

type AllowedProtos struct {
	Protos [256]AllowedIPs
}

func (ap *AllowedProtos) Insert(proto byte, prefix netip.Prefix, peer *Peer) {
	ap.Protos[proto].Insert(prefix, peer)
}

func (ap *AllowedProtos) Lookup(proto byte, ip []byte) (peer *Peer) {
	peer = ap.Protos[proto].Lookup(ip)
	if peer == nil {
		peer = ap.Protos[AnyProto].Lookup(ip)
	}
	return peer
}

func (ap *AllowedProtos) RemoveByPeer(peer *Peer) {
	for i := range ap.Protos {
		ap.Protos[i].RemoveByPeer(peer)
	}
}

func (ap *AllowedProtos) EntriesForPeer(peer *Peer, cb func(proto byte, prefix netip.Prefix) bool) {
	continue_ := true
	for i := range ap.Protos {
		ap.Protos[i].EntriesForPeer(peer, func(prefix netip.Prefix) bool {
			continue_ = cb(byte(i), prefix)
			return continue_
		})
		if !continue_ {
			return
		}
	}
}
