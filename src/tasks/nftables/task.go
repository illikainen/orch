package nftables

import "github.com/illikainen/orch/src/codec"

type Task struct {
	Cond    *bool     `json:"condition" hcl:"condition,optional" default:"true"`
	File    string    `json:"file"      hcl:"file,optional"`
	Policy  Policy    `json:"policy"    hcl:"policy,optional"`
	Ingress []Ingress `json:"ingress"   hcl:"ingress,optional"`
	Egress  []Egress  `json:"egress"    hcl:"egress,optional"`
	NAT     []NAT     `json:"nat"       hcl:"nat,optional"`
}

type Policy struct {
	Ingress string `json:"ingress" hcl:"ingress,optional" validate:"in:[\"accept\", \"drop\"]" default:"drop"`
	Egress  string `json:"egress"  hcl:"egress,optional"  validate:"in:[\"accept\", \"drop\"]" default:"drop"`
	Forward string `json:"forward" hcl:"forward,optional" validate:"in:[\"accept\", \"drop\"]" default:"drop"`
}

type Packet struct { // revive:disable:line-length-limit
	Action   string          `json:"action"    hcl:"action"             validate:"in:[\"accept\", \"drop\", \"reject\"]"`
	Proto    string          `json:"proto"     hcl:"proto"              validate:"in:[\"tcp\", \"udp\", \"icmp\"]"`
	SrcIface string          `json:"src_iface" hcl:"src_iface,optional" validate:"rx:^[a-zA-Z0-9-]+$"`
	DstIface string          `json:"dst_iface" hcl:"dst_iface,optional" validate:"rx:^[a-zA-Z0-9-]+$"`
	SrcPort  codec.Stringify `json:"src_port"  hcl:"src_port,optional"  validate:"port"`
	SrcAddr  string          `json:"src_addr"  hcl:"src_addr,optional"  validate:"ip:autocidr"`
	DstPort  codec.Stringify `json:"dst_port"  hcl:"dst_port,optional"  validate:"port"`
	DstAddr  string          `json:"dst_addr"  hcl:"dst_addr,optional"  validate:"ip:autocidr"`
	ICMPType string          `json:"icmp_type" hcl:"icmp_type,optional" validate:"rx:^[a-z-]+$"`
}

type Ingress struct {
	Packet
}

type Egress struct {
	Packet
	User  string `json:"user"      hcl:"user,optional"      validate:"user"`
	Group string `json:"group"     hcl:"group,optional"     validate:"group"`
}

type NAT struct {
	Packet
	Action string   `json:"action"    hcl:"action"             validate:"in:[\"masquerade\"]"`
	Flags  []string `json:"flags"     hcl:"flags,optional"     validate:"in:[\"fully-random\"]"`
}
