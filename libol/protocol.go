package libol

import (
	"encoding/binary"
	"fmt"
)

var (
	ZEROED    = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	BROADED   = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	DEFAULTED = []byte{0x00, 0x16, 0x3e, 0x02, 0x56, 0x23}
)

const (
	ETHPARP  = 0x0806
	ETHPIP4  = 0x0800
	ETHPIP6  = 0x86DD
	ETHPVLAN = 0x8100
)

type Ether struct {
	Dst  []byte
	Src  []byte
	Type uint16
	Len  int
}

func NewEther(t uint16) (e *Ether) {
	e = &Ether{
		Type: t,
		Src:  ZEROED,
		Dst:  ZEROED,
		Len:  14,
	}
	return
}

func NewEtherArp() (e *Ether) {
	return NewEther(ETHPARP)
}

func NewEtherIP4() (e *Ether) {
	return NewEther(ETHPIP4)
}

func NewEtherFromFrame(frame []byte) (e *Ether, err error) {
	e = &Ether{
		Len: 14,
	}
	err = e.Decode(frame)
	return
}

func (e *Ether) Decode(frame []byte) error {
	if len(frame) < 14 {
		return NewErr("Ether.Decode too small header: %d", len(frame))
	}

	e.Dst = frame[:6]
	e.Src = frame[6:12]
	e.Type = binary.BigEndian.Uint16(frame[12:14])
	e.Len = 14

	return nil
}

func (e *Ether) Encode() []byte {
	buffer := make([]byte, 14)

	copy(buffer[:6], e.Dst)
	copy(buffer[6:12], e.Src)
	binary.BigEndian.PutUint16(buffer[12:14], e.Type)

	return buffer[:14]
}

func (e *Ether) IsVlan() bool {
	return e.Type == ETHPVLAN
}

func (e *Ether) IsArp() bool {
	return e.Type == ETHPARP
}

func (e *Ether) IsIP4() bool {
	return e.Type == ETHPIP4
}

type Vlan struct {
	Tci uint16
	Vid uint16
	Pro uint16
	Len int
}

func NewVlan(tci uint16, vid uint16) (n *Vlan) {
	n = &Vlan{
		Tci: tci,
		Vid: vid,
		Len: 4,
	}

	return
}

func NewVlanFromFrame(frame []byte) (n *Vlan, err error) {
	n = &Vlan{
		Len: 4,
	}
	err = n.Decode(frame)
	return
}

func (n *Vlan) Decode(frame []byte) error {
	if len(frame) < 4 {
		return NewErr("Vlan.Decode: too small header")
	}

	v := binary.BigEndian.Uint16(frame[0:2])
	n.Tci = uint16(v >> 12)
	n.Vid = uint16(0x0fff & v)
	n.Pro = binary.BigEndian.Uint16(frame[2:4])

	return nil
}

func (n *Vlan) Encode() []byte {
	buffer := make([]byte, 16)

	v := (n.Tci << 12) | n.Vid
	binary.BigEndian.PutUint16(buffer[0:2], v)
	binary.BigEndian.PutUint16(buffer[2:4], n.Pro)

	return buffer[:4]
}

const (
	ARP_REQUEST = 1
	ARP_REPLY   = 2
)

const (
	ARPHRD_NETROM = 0
	ARPHRD_ETHER  = 1
	ARPHRD_EETHER = 2
)

type Arp struct {
	HrdCode uint16 // format hardware address
	ProCode uint16 // format protocol address
	HrdLen  uint8  // length of hardware address
	ProLen  uint8  // length of protocol address
	OpCode  uint16 // ARP Op(command)

	SHwAddr []byte // sender hardware address.
	SIpAddr []byte // sender IP address.
	THwAddr []byte // target hardware address.
	TIpAddr []byte // target IP address.
	Len     int
}

func NewArp() (a *Arp) {
	a = &Arp{
		HrdCode: ARPHRD_ETHER,
		ProCode: ETHPIP4,
		HrdLen:  6,
		ProLen:  4,
		OpCode:  ARP_REQUEST,
		Len:     0,
	}

	return
}

func NewArpFromFrame(frame []byte) (a *Arp, err error) {
	a = &Arp{
		Len: 0,
	}
	err = a.Decode(frame)
	return
}

func (a *Arp) Decode(frame []byte) error {
	var err error

	if len(frame) < 8 {
		return NewErr("Arp.Decode: too small header: %d", len(frame))
	}

	a.HrdCode = binary.BigEndian.Uint16(frame[0:2])
	a.ProCode = binary.BigEndian.Uint16(frame[2:4])
	a.HrdLen = uint8(frame[4])
	a.ProLen = uint8(frame[5])
	a.OpCode = binary.BigEndian.Uint16(frame[6:8])

	p := uint8(8)
	if len(frame) < int(p+2*(a.HrdLen+a.ProLen)) {
		return NewErr("Arp.Decode: too small frame: %d", len(frame))
	}

	a.SHwAddr = frame[p : p+a.HrdLen]
	p += a.HrdLen
	a.SIpAddr = frame[p : p+a.ProLen]
	p += a.ProLen

	a.THwAddr = frame[p : p+a.HrdLen]
	p += a.HrdLen
	a.TIpAddr = frame[p : p+a.ProLen]
	p += a.ProLen

	a.Len = int(p)

	return err
}

func (a *Arp) Encode() []byte {
	buffer := make([]byte, 1024)

	binary.BigEndian.PutUint16(buffer[0:2], a.HrdCode)
	binary.BigEndian.PutUint16(buffer[2:4], a.ProCode)
	buffer[4] = byte(a.HrdLen)
	buffer[5] = byte(a.ProLen)
	binary.BigEndian.PutUint16(buffer[6:8], a.OpCode)

	p := uint8(8)
	copy(buffer[p:p+a.HrdLen], a.SHwAddr[0:a.HrdLen])
	p += a.HrdLen
	copy(buffer[p:p+a.ProLen], a.SIpAddr[0:a.ProLen])
	p += a.ProLen

	copy(buffer[p:p+a.HrdLen], a.THwAddr[0:a.HrdLen])
	p += a.HrdLen
	copy(buffer[p:p+a.ProLen], a.TIpAddr[0:a.ProLen])
	p += a.ProLen

	a.Len = int(p)

	return buffer[:p]
}

func (a *Arp) IsIP4() bool {
	return a.ProCode == ETHPIP4
}

const (
	IPV4_VER = 0x04
	IPV6_VER = 0x06
)

const (
	IPPROTO_ICMP = 0x01
	IPPROTO_IGMP = 0x02
	IPPROTO_IPIP = 0x04
	IPPROTO_TCP  = 0x06
	IPPROTO_UDP  = 0x11
	IPPROTO_ESP  = 0x32
	IPPROTO_AH   = 0x33
	IPPROTO_OSPF = 0x59
	IPPROTO_PIM  = 0x67
	IPPROTO_VRRP = 0x70
	IPPROTO_ISIS = 0x7c
)

func IpProto2Str(proto uint8) string {
	switch proto {
	case IPPROTO_ICMP:
		return "icmp"
	case IPPROTO_IGMP:
		return "igmp"
	case IPPROTO_IPIP:
		return "ipip"
	case IPPROTO_ESP:
		return "esp"
	case IPPROTO_AH:
		return "ah"
	case IPPROTO_OSPF:
		return "ospf"
	case IPPROTO_ISIS:
		return "isis"
	case IPPROTO_UDP:
		return "udp"
	case IPPROTO_TCP:
		return "tcp"
	case IPPROTO_PIM:
		return "pim"
	case IPPROTO_VRRP:
		return "vrrp"
	default:
		return fmt.Sprintf("%02x", proto)
	}
}

const IPV4_LEN = 20

type Ipv4 struct {
	Version        uint8 //4bite v4: 0100, v6: 0110
	HeaderLen      uint8 //4bit 15*4
	ToS            uint8 //Type of Service
	TotalLen       uint16
	Identifier     uint16
	Flag           uint16 //3bit Z|DF|MF
	Offset         uint16 //13bit Fragment offset
	ToL            uint8  //Time of Live
	Protocol       uint8
	HeaderChecksum uint16 //Header Checksum
	Source         []byte
	Destination    []byte
	Options        uint32 //Reserved
	Len            int
}

func NewIpv4() (i *Ipv4) {
	i = &Ipv4{
		Version:        0x04,
		HeaderLen:      0x05,
		ToS:            0,
		TotalLen:       0,
		Identifier:     0,
		Flag:           0,
		Offset:         0,
		ToL:            0xff,
		Protocol:       0,
		HeaderChecksum: 0,
		Options:        0,
		Len:            IPV4_LEN,
	}
	return
}

func NewIpv4FromFrame(frame []byte) (i *Ipv4, err error) {
	i = NewIpv4()
	err = i.Decode(frame)
	return
}

func (i *Ipv4) Decode(frame []byte) error {
	if len(frame) < IPV4_LEN {
		return NewErr("Ipv4.Decode: too small header: %d", len(frame))
	}

	h := uint8(frame[0])
	i.Version = h >> 4
	i.HeaderLen = h & 0x0f
	i.ToS = uint8(frame[1])
	i.TotalLen = binary.BigEndian.Uint16(frame[2:4])
	i.Identifier = binary.BigEndian.Uint16(frame[4:6])
	f := binary.BigEndian.Uint16(frame[6:8])
	i.Offset = f & 0x1fFf
	i.Flag = f >> 13
	i.ToL = uint8(frame[8])
	i.Protocol = uint8(frame[9])
	i.HeaderChecksum = binary.BigEndian.Uint16(frame[10:12])
	if !i.IsIP4() {
		return NewErr("Ipv4.Decode: not right ipv4 version: 0x%x", i.Version)
	}
	i.Source = frame[12:16]
	i.Destination = frame[16:20]

	return nil
}

func (i *Ipv4) Encode() []byte {
	buffer := make([]byte, 32)

	h := uint8((i.Version << 4) | i.HeaderLen)
	buffer[0] = h
	buffer[1] = i.ToS
	binary.BigEndian.PutUint16(buffer[2:4], i.TotalLen)
	binary.BigEndian.PutUint16(buffer[4:6], i.Identifier)
	f := uint16((i.Flag << 13) | i.Offset)
	binary.BigEndian.PutUint16(buffer[6:8], f)
	buffer[8] = i.ToL
	buffer[9] = i.Protocol
	binary.BigEndian.PutUint16(buffer[10:12], i.HeaderChecksum)
	copy(buffer[12:16], i.Source[:4])
	copy(buffer[16:20], i.Destination[:4])

	return buffer[:i.Len]
}

func (i *Ipv4) IsIP4() bool {
	return i.Version == IPV4_VER
}

const TCP_LEN = 20

type Tcp struct {
	Source         uint16
	Destination    uint16
	Sequence       uint32
	Acknowledgment uint32
	DataOffset     uint8
	ControlBits    uint8
	Window         uint16
	Checksum       uint16
	UrgentPointer  uint16
	Options        []byte
	Padding        []byte
	Len            int
}

func NewTcp() (t *Tcp) {
	t = &Tcp{
		Source:         0,
		Destination:    0,
		Sequence:       0,
		Acknowledgment: 0,
		DataOffset:     0,
		ControlBits:    0,
		Window:         0,
		Checksum:       0,
		UrgentPointer:  0,
		Len:            TCP_LEN,
	}
	return
}

func NewTcpFromFrame(frame []byte) (t *Tcp, err error) {
	t = NewTcp()
	err = t.Decode(frame)
	return
}

func (t *Tcp) Decode(frame []byte) error {
	if len(frame) < TCP_LEN {
		return NewErr("Tcp.Decode: too small header: %d", len(frame))
	}

	t.Source = binary.BigEndian.Uint16(frame[0:2])
	t.Destination = binary.BigEndian.Uint16(frame[2:4])
	t.Sequence = binary.BigEndian.Uint32(frame[4:8])
	t.Acknowledgment = binary.BigEndian.Uint32(frame[8:12])
	t.DataOffset = uint8(frame[12])
	t.ControlBits = uint8(frame[13])
	t.Window = binary.BigEndian.Uint16(frame[14:16])
	t.Checksum = binary.BigEndian.Uint16(frame[16:18])
	t.UrgentPointer = binary.BigEndian.Uint16(frame[18:20])

	return nil
}

func (t *Tcp) Encode() []byte {
	buffer := make([]byte, 32)

	binary.BigEndian.PutUint16(buffer[0:2], t.Source)
	binary.BigEndian.PutUint16(buffer[2:4], t.Destination)
	binary.BigEndian.PutUint32(buffer[4:8], t.Sequence)
	binary.BigEndian.PutUint32(buffer[8:12], t.Acknowledgment)
	buffer[12] = t.DataOffset
	buffer[13] = t.ControlBits
	binary.BigEndian.PutUint16(buffer[14:16], t.Window)
	binary.BigEndian.PutUint16(buffer[16:18], t.Checksum)
	binary.BigEndian.PutUint16(buffer[18:20], t.UrgentPointer)

	return buffer[:t.Len]
}

const UDP_LEN = 8

type Udp struct {
	Source      uint16
	Destination uint16
	Length      uint16
	Checksum    uint16
	Len         int
}

func NewUdp() (u *Udp) {
	u = &Udp{
		Source:      0,
		Destination: 0,
		Length:      0,
		Checksum:    0,
		Len:         UDP_LEN,
	}
	return
}

func NewUdpFromFrame(frame []byte) (u *Udp, err error) {
	u = NewUdp()
	err = u.Decode(frame)
	return
}

func (u *Udp) Decode(frame []byte) error {
	if len(frame) < UDP_LEN {
		return NewErr("Udp.Decode: too small header: %d", len(frame))
	}

	u.Source = binary.BigEndian.Uint16(frame[0:2])
	u.Destination = binary.BigEndian.Uint16(frame[2:4])
	u.Length = binary.BigEndian.Uint16(frame[4:6])
	u.Checksum = binary.BigEndian.Uint16(frame[6:8])

	return nil
}

func (u *Udp) Encode() []byte {
	buffer := make([]byte, 32)

	binary.BigEndian.PutUint16(buffer[0:2], u.Source)
	binary.BigEndian.PutUint16(buffer[2:4], u.Destination)
	binary.BigEndian.PutUint16(buffer[4:6], u.Length)
	binary.BigEndian.PutUint16(buffer[6:8], u.Checksum)

	return buffer[:u.Len]
}
