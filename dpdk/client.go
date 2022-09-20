// Copyright 2022 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dpdk

import (
	"context"
	"fmt"
	"net/netip"

	dpdkproto "github.com/onmetal/net-dpservice-go/proto"
	"k8s.io/apimachinery/pkg/types"
)

type Client interface {
	GetInterface(ctx context.Context, uid types.UID) (*Interface, error)
	CreateInterface(ctx context.Context, iface *Interface) (*Interface, error)
	DeleteInterface(ctx context.Context, uid types.UID) error

	GetVirtualIP(ctx context.Context, interfaceUID types.UID) (*VirtualIP, error)
	CreateVirtualIP(ctx context.Context, virtualIP *VirtualIP) (*VirtualIP, error)
	DeleteVirtualIP(ctx context.Context, interfaceUID types.UID) error

	ListPrefixes(ctx context.Context, interfaceUID types.UID) (*PrefixList, error)
	CreatePrefix(ctx context.Context, prefix *Prefix) (*Prefix, error)
	DeletePrefix(ctx context.Context, interfaceUID types.UID, prefix netip.Prefix) error

	CreateRoute(ctx context.Context, route *Route) (*Route, error)
	DeleteRoute(ctx context.Context, route *Route) error
}

type Route struct {
	RouteMetadata
	Spec RouteSpec
}

type RouteMetadata struct {
	VNI uint32
}

type RouteSpec struct {
	Prefix  netip.Prefix
	NextHop RouteNextHop
}

type RouteNextHop struct {
	VNI     uint32
	Address netip.Addr
}

type PrefixList struct {
	Items []Prefix
}

type Prefix struct {
	PrefixMetadata
	Spec PrefixSpec
}

type PrefixMetadata struct {
	InterfaceUID types.UID
}

type PrefixSpec struct {
	Prefix netip.Prefix
}

type VirtualIP struct {
	VirtualIPMetadata
	Spec VirtualIPSpec
}

type VirtualIPMetadata struct {
	InterfaceUID types.UID
}

type VirtualIPSpec struct {
	Address netip.Addr
}

type Interface struct {
	InterfaceMetadata
	Spec   InterfaceSpec
	Status InterfaceStatus
}

type InterfaceMetadata struct {
	UID types.UID
}

type InterfaceSpec struct {
	VNI                uint32
	Device             string
	PrimaryIPv4Address netip.Addr
	PrimaryIPv6Address netip.Addr
}

type InterfaceStatus struct {
	UnderlayRoute netip.Addr
}

func dpdkInterfaceToInterface(dpdkIface *dpdkproto.Interface) (*Interface, error) {
	primaryIPv4Address, err := netip.ParseAddr(string(dpdkIface.GetPrimaryIPv4Address()))
	if err != nil {
		return nil, fmt.Errorf("error parsing primary ipv4 address: %w", err)
	}

	primaryIPv6Address, err := netip.ParseAddr(string(dpdkIface.GetPrimaryIPv4Address()))
	if err != nil {
		return nil, fmt.Errorf("error parsing primary ipv6 address: %w", err)
	}

	underlayRoute, err := netip.ParseAddr(string(dpdkIface.GetUnderlayRoute()))
	if err != nil {
		return nil, fmt.Errorf("error parsing underlay route: %w", err)
	}

	return &Interface{
		InterfaceMetadata: InterfaceMetadata{
			UID: types.UID(dpdkIface.InterfaceID),
		},
		Spec: InterfaceSpec{
			VNI:                dpdkIface.GetVni(),
			Device:             dpdkIface.GetPciDpName(),
			PrimaryIPv4Address: primaryIPv4Address,
			PrimaryIPv6Address: primaryIPv6Address,
		},
		Status: InterfaceStatus{
			UnderlayRoute: underlayRoute,
		},
	}, nil
}

func netipAddrToDPDKIPVersion(addr netip.Addr) dpdkproto.IPVersion {
	switch {
	case addr.Is4():
		return dpdkproto.IPVersion_IPv4
	case addr.Is6():
		return dpdkproto.IPVersion_IPv6
	default:
		return 0
	}
}

func netipAddrToDPDKIPConfig(addr netip.Addr) *dpdkproto.IPConfig {
	if !addr.IsValid() {
		return nil
	}

	return &dpdkproto.IPConfig{
		IpVersion:      netipAddrToDPDKIPVersion(addr),
		PrimaryAddress: []byte(addr.String()),
	}
}

type client struct {
	dpdkproto.DPDKonmetalClient
}

func NewClient(protoClient dpdkproto.DPDKonmetalClient) Client {
	return &client{protoClient}
}

func (c *client) GetInterface(ctx context.Context, uid types.UID) (*Interface, error) {
	res, err := c.DPDKonmetalClient.GetInterface(ctx, &dpdkproto.InterfaceIDMsg{InterfaceID: []byte(uid)})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetStatus().GetError(); errorCode != 0 {
		return nil, &StatusError{errorCode: errorCode, message: res.GetStatus().GetMessage()}
	}
	return dpdkInterfaceToInterface(res.GetInterface())
}

func (c *client) CreateInterface(ctx context.Context, iface *Interface) (*Interface, error) {
	res, err := c.DPDKonmetalClient.CreateInterface(ctx, &dpdkproto.CreateInterfaceRequest{
		InterfaceType: dpdkproto.InterfaceType_VirtualInterface,
		InterfaceID:   []byte(iface.UID),
		Vni:           iface.Spec.VNI,
		Ipv4Config:    netipAddrToDPDKIPConfig(iface.Spec.PrimaryIPv4Address),
		Ipv6Config:    netipAddrToDPDKIPConfig(iface.Spec.PrimaryIPv6Address),
		DeviceName:    iface.Spec.Device,
	})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetResponse().GetStatus().GetError(); errorCode != 0 {
		return nil, &StatusError{errorCode: errorCode, message: res.GetResponse().GetStatus().GetMessage()}
	}

	underlayRoute, err := netip.ParseAddr(string(res.GetResponse().GetUnderlayRoute()))
	if err != nil {
		return nil, fmt.Errorf("error parsing underlay route: %w", err)
	}

	return &Interface{
		InterfaceMetadata: iface.InterfaceMetadata,
		Spec:              iface.Spec, // TODO: Enable dynamic device allocation
		Status: InterfaceStatus{
			UnderlayRoute: underlayRoute,
		},
	}, nil
}

func (c *client) DeleteInterface(ctx context.Context, uid types.UID) error {
	res, err := c.DPDKonmetalClient.DeleteInterface(ctx, &dpdkproto.InterfaceIDMsg{InterfaceID: []byte(uid)})
	if err != nil {
		return err
	}
	if errorCode := res.GetError(); errorCode != 0 {
		return &StatusError{errorCode: errorCode, message: res.GetMessage()}
	}
	return nil
}

func dpdkVirtualIPToVirtualIP(interfaceUID types.UID, dpdkVIP *dpdkproto.InterfaceVIPIP) (*VirtualIP, error) {
	addr, err := netip.ParseAddr(string(dpdkVIP.GetAddress()))
	if err != nil {
		return nil, fmt.Errorf("error parsing virtual ip address: %w", err)
	}

	return &VirtualIP{
		VirtualIPMetadata: VirtualIPMetadata{
			InterfaceUID: interfaceUID,
		},
		Spec: VirtualIPSpec{
			Address: addr,
		},
	}, nil
}

func (c *client) GetVirtualIP(ctx context.Context, interfaceUID types.UID) (*VirtualIP, error) {
	res, err := c.DPDKonmetalClient.GetInterfaceVIP(ctx, &dpdkproto.InterfaceIDMsg{
		InterfaceID: []byte(interfaceUID),
	})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetStatus().GetError(); errorCode != 0 {
		return nil, &StatusError{errorCode: errorCode, message: res.GetStatus().GetMessage()}
	}

	return dpdkVirtualIPToVirtualIP(interfaceUID, res)
}

func (c *client) CreateVirtualIP(ctx context.Context, virtualIP *VirtualIP) (*VirtualIP, error) {
	res, err := c.DPDKonmetalClient.AddInterfaceVIP(ctx, &dpdkproto.InterfaceVIPMsg{
		InterfaceVIPIP: &dpdkproto.InterfaceVIPIP{
			IpVersion: netipAddrToDPDKIPVersion(virtualIP.Spec.Address),
			Address:   []byte(virtualIP.Spec.Address.String()),
		},
	})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetStatus().GetError(); errorCode != 0 {
		return nil, &StatusError{errorCode: errorCode, message: res.GetStatus().GetMessage()}
	}

	return virtualIP, nil
}

func (c *client) DeleteVirtualIP(ctx context.Context, interfaceUID types.UID) error {
	res, err := c.DPDKonmetalClient.DeleteInterfaceVIP(ctx, &dpdkproto.InterfaceIDMsg{
		InterfaceID: []byte(interfaceUID),
	})
	if err != nil {
		return err
	}
	if errorCode := res.GetError(); errorCode != 0 {
		return &StatusError{errorCode: errorCode, message: res.GetMessage()}
	}
	return nil
}

func dpdkPrefixToPrefix(interfaceUID types.UID, dpdkPrefix *dpdkproto.Prefix) (*Prefix, error) {
	addr, err := netip.ParseAddr(string(dpdkPrefix.GetAddress()))
	if err != nil {
		return nil, fmt.Errorf("error parsing dpdk prefix address: %w", err)
	}

	prefix, err := addr.Prefix(int(dpdkPrefix.PrefixLength))
	if err != nil {
		return nil, fmt.Errorf("invalid dpdk prefix length %d for address %s", dpdkPrefix.PrefixLength, addr)
	}

	return &Prefix{
		PrefixMetadata: PrefixMetadata{
			InterfaceUID: interfaceUID,
		},
		Spec: PrefixSpec{
			Prefix: prefix,
		},
	}, nil
}

func (c *client) ListPrefixes(ctx context.Context, interfaceUID types.UID) (*PrefixList, error) {
	res, err := c.DPDKonmetalClient.ListInterfacePrefixes(ctx, &dpdkproto.InterfaceIDMsg{
		InterfaceID: []byte(interfaceUID),
	})
	if err != nil {
		return nil, err
	}

	var prefixes []Prefix
	for _, dpdkPrefix := range res.GetPrefixes() {
		prefix, err := dpdkPrefixToPrefix(interfaceUID, dpdkPrefix)
		if err != nil {
			return nil, err
		}

		prefixes = append(prefixes, *prefix)
	}

	return &PrefixList{
		Items: prefixes,
	}, nil
}

func (c *client) CreatePrefix(ctx context.Context, prefix *Prefix) (*Prefix, error) {
	res, err := c.DPDKonmetalClient.AddInterfacePrefix(ctx, &dpdkproto.InterfacePrefixMsg{
		InterfaceID: &dpdkproto.InterfaceIDMsg{
			InterfaceID: []byte(prefix.InterfaceUID),
		},
		Prefix: &dpdkproto.Prefix{
			IpVersion:    netipAddrToDPDKIPVersion(prefix.Spec.Prefix.Addr()),
			Address:      []byte(prefix.Spec.Prefix.Addr().String()),
			PrefixLength: uint32(prefix.Spec.Prefix.Bits()),
		},
	})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetStatus().GetError(); errorCode != 0 {
		return nil, &StatusError{errorCode: errorCode, message: res.GetStatus().GetMessage()}
	}
	return prefix, nil
}

func (c *client) DeletePrefix(ctx context.Context, interfaceUID types.UID, prefix netip.Prefix) error {
	res, err := c.DPDKonmetalClient.DeleteInterfacePrefix(ctx, &dpdkproto.InterfacePrefixMsg{
		InterfaceID: &dpdkproto.InterfaceIDMsg{
			InterfaceID: []byte(interfaceUID),
		},
		Prefix: &dpdkproto.Prefix{
			IpVersion:    netipAddrToDPDKIPVersion(prefix.Addr()),
			Address:      []byte(prefix.Addr().String()),
			PrefixLength: uint32(prefix.Bits()),
		},
	})
	if err != nil {
		return err
	}
	if errorCode := res.GetError(); errorCode != 0 {
		return &StatusError{errorCode: errorCode, message: res.GetMessage()}
	}
	return nil
}

func (c *client) CreateRoute(ctx context.Context, route *Route) (*Route, error) {
	res, err := c.DPDKonmetalClient.AddRoute(ctx, &dpdkproto.VNIRouteMsg{
		Vni: &dpdkproto.VNIMsg{Vni: route.VNI},
		Route: &dpdkproto.Route{
			IpVersion: netipAddrToDPDKIPVersion(route.Spec.NextHop.Address),
			Weight:    100,
			Prefix: &dpdkproto.Prefix{
				IpVersion:    netipAddrToDPDKIPVersion(route.Spec.Prefix.Addr()),
				Address:      []byte(route.Spec.Prefix.String()),
				PrefixLength: uint32(route.Spec.Prefix.Bits()),
			},
			NexthopVNI:     route.Spec.NextHop.VNI,
			NexthopAddress: []byte(route.Spec.NextHop.Address.String()),
		},
	})
	if err != nil {
		return nil, err
	}
	if errorCode := res.GetError(); errorCode != 0 {
		return nil, &StatusError{errorCode: errorCode, message: res.GetMessage()}
	}
	return route, nil
}

func (c *client) DeleteRoute(ctx context.Context, route *Route) error {
	res, err := c.DPDKonmetalClient.DeleteRoute(ctx, &dpdkproto.VNIRouteMsg{
		Vni: &dpdkproto.VNIMsg{Vni: route.VNI},
		Route: &dpdkproto.Route{
			IpVersion: netipAddrToDPDKIPVersion(route.Spec.NextHop.Address),
			Weight:    100,
			Prefix: &dpdkproto.Prefix{
				IpVersion:    netipAddrToDPDKIPVersion(route.Spec.Prefix.Addr()),
				Address:      []byte(route.Spec.Prefix.String()),
				PrefixLength: uint32(route.Spec.Prefix.Bits()),
			},
			NexthopVNI:     route.Spec.NextHop.VNI,
			NexthopAddress: []byte(route.Spec.NextHop.Address.String()),
		},
	})
	if err != nil {
		return err
	}
	if errorCode := res.GetError(); errorCode != 0 {
		return &StatusError{errorCode: errorCode, message: res.GetMessage()}
	}
	return nil
}