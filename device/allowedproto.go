package device

import (
	"container/list"
	"net/netip"
	"sync"
	"unsafe"
)

const AnyProto = 255

type AllowedProtos struct {
	Protos [256]AllowedIPs
	mutex  sync.RWMutex
}

func (ap *AllowedProtos) Insert(proto byte, prefix netip.Prefix, peer *Peer) {
	ap.mutex.Lock()
	defer ap.mutex.Unlock()
	ap.Protos[proto].Insert(proto, prefix, peer)
}

func (ap *AllowedProtos) Lookup(proto byte, ip []byte) (peer *Peer) {
	ap.mutex.RLock()
	defer ap.mutex.RUnlock()
	peer = ap.Protos[proto].Lookup(ip)
	if peer == nil {
		peer = ap.Protos[AnyProto].Lookup(ip)
	}
	return peer
}

func (ap *AllowedProtos) RemoveByPeer(peer *Peer) {
	ap.mutex.Lock()
	defer ap.mutex.Unlock()

	var next *list.Element
	for elem := peer.trieEntries.Front(); elem != nil; elem = next {
		next = elem.Next()
		node := elem.Value.(*trieEntry)

		node.removeFromPeerEntries()
		node.peer = nil
		if node.child[0] != nil && node.child[1] != nil {
			continue
		}
		bit := 0
		if node.child[0] == nil {
			bit = 1
		}
		child := node.child[bit]
		if child != nil {
			child.parent = node.parent
		}
		*node.parent.parentBit = child
		if node.child[0] != nil || node.child[1] != nil || node.parent.parentBitType > 1 {
			node.zeroizePointers()
			continue
		}
		parent := (*trieEntry)(unsafe.Pointer(uintptr(unsafe.Pointer(node.parent.parentBit)) - unsafe.Offsetof(node.child) - unsafe.Sizeof(node.child[0])*uintptr(node.parent.parentBitType)))
		if parent.peer != nil {
			node.zeroizePointers()
			continue
		}
		child = parent.child[node.parent.parentBitType^1]
		if child != nil {
			child.parent = parent.parent
		}
		*parent.parent.parentBit = child
		node.zeroizePointers()
		parent.zeroizePointers()
	}
}

func (ap *AllowedProtos) EntriesForPeer(peer *Peer, cb func(proto byte, prefix netip.Prefix) bool) {
	ap.mutex.RLock()
	defer ap.mutex.RUnlock()
	for elem := peer.trieEntries.Front(); elem != nil; elem = elem.Next() {
		node := elem.Value.(*trieEntry)
		a, _ := netip.AddrFromSlice(node.bits)
		if !cb(node.proto, netip.PrefixFrom(a, int(node.cidr))) {
			return
		}
	}
}
