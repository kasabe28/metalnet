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

package v1alpha1

import (
	"encoding/json"
	"net/netip"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

// LocalUIDReference is a reference to another entity including its UID
type LocalUIDReference struct {
	// Name is the name of the referenced entity.
	Name string `json:"name"`
	// UID is the UID of the referenced entity.
	// +optional
	UID types.UID `json:"uid,omitempty"`
}

// LBPort consists of port and protocol
type LBPort struct {
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
}

// IP is an IP address.
// +kubebuilder:validation:Type=string
type IP struct {
	netip.Addr `json:"-"`
}

func (in *IP) DeepCopyInto(out *IP) {
	*out = *in
}

func (in *IP) DeepCopy() *IP {
	return &IP{in.Addr}
}

func (i IP) GomegaString() string {
	return i.String()
}

func (i *IP) UnmarshalJSON(b []byte) error {
	if len(b) == 4 && string(b) == "null" {
		i.Addr = netip.Addr{}
		return nil
	}

	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}

	p, err := netip.ParseAddr(str)
	if err != nil {
		return err
	}

	i.Addr = p
	return nil
}

func (i IP) MarshalJSON() ([]byte, error) {
	if i.IsZero() {
		// Encode unset/nil objects as JSON's "null".
		return []byte("null"), nil
	}
	return json.Marshal(i.String())
}

func (i IP) ToUnstructured() interface{} {
	if i.IsZero() {
		return nil
	}
	return i.Addr.String()
}

func (i *IP) IsValid() bool {
	return i != nil && i.Addr.IsValid()
}

func (i *IP) IsZero() bool {
	return i == nil || !i.Addr.IsValid()
}

func (i IP) Family() corev1.IPFamily {
	switch {
	case i.Is4():
		return corev1.IPv4Protocol
	case i.Is6():
		return corev1.IPv6Protocol
	default:
		return ""
	}
}

func (_ IP) OpenAPISchemaType() []string { return []string{"string"} }

func (_ IP) OpenAPISchemaFormat() string { return "ip" }

func NewIP(ip netip.Addr) IP {
	return IP{ip}
}

func ParseIP(s string) (IP, error) {
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return IP{}, err
	}
	return IP{addr}, nil
}

func ParseNewIP(s string) (*IP, error) {
	ip, err := ParseIP(s)
	if err != nil {
		return nil, err
	}
	return &ip, nil
}

func MustParseIP(s string) IP {
	return IP{netip.MustParseAddr(s)}
}

func MustParseNewIP(s string) *IP {
	ip, err := ParseNewIP(s)
	utilruntime.Must(err)
	return ip
}

func NewIPPtr(ip netip.Addr) *IP {
	return &IP{ip}
}

func PtrToIP(addr IP) *IP {
	return &addr
}

func EqualIPs(a, b IP) bool {
	return a == b
}

// IPPrefix represents a network prefix.
// +kubebuilder:validation:Type=string
// +nullable
type IPPrefix struct {
	netip.Prefix `json:"-"`
}

func (i IPPrefix) GomegaString() string {
	return i.String()
}

func (i IPPrefix) IP() IP {
	return IP{i.Prefix.Addr()}
}

func (i *IPPrefix) UnmarshalJSON(b []byte) error {
	if len(b) == 4 && string(b) == "null" {
		i.Prefix = netip.Prefix{}
		return nil
	}

	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}

	p, err := netip.ParsePrefix(str)
	if err != nil {
		return err
	}

	i.Prefix = p
	return nil
}

func (i IPPrefix) MarshalJSON() ([]byte, error) {
	if i.IsZero() {
		// Encode unset/nil objects as JSON's "null".
		return []byte("null"), nil
	}
	return json.Marshal(i.String())
}

func (i IPPrefix) ToUnstructured() interface{} {
	if i.IsZero() {
		return nil
	}
	return i.String()
}

func (in *IPPrefix) DeepCopyInto(out *IPPrefix) {
	*out = *in
}

func (in *IPPrefix) DeepCopy() *IPPrefix {
	return &IPPrefix{in.Prefix}
}

func (in *IPPrefix) IsValid() bool {
	return in != nil && in.Prefix.IsValid()
}

func (in *IPPrefix) IsZero() bool {
	return in == nil || !in.Prefix.IsValid()
}

func (_ IPPrefix) OpenAPISchemaType() []string { return []string{"string"} }

func (_ IPPrefix) OpenAPISchemaFormat() string { return "ip-prefix" }

func NewIPPrefix(prefix netip.Prefix) *IPPrefix {
	return &IPPrefix{Prefix: prefix}
}

func ParseIPPrefix(s string) (IPPrefix, error) {
	prefix, err := netip.ParsePrefix(s)
	if err != nil {
		return IPPrefix{}, err
	}
	return IPPrefix{prefix}, nil
}

func ParseNewIPPrefix(s string) (*IPPrefix, error) {
	prefix, err := ParseIPPrefix(s)
	if err != nil {
		return nil, err
	}
	return &prefix, nil
}

func MustParseIPPrefix(s string) IPPrefix {
	return IPPrefix{netip.MustParsePrefix(s)}
}

func MustParseNewIPPrefix(s string) *IPPrefix {
	prefix, err := ParseNewIPPrefix(s)
	utilruntime.Must(err)
	return prefix
}

func PtrToIPPrefix(prefix IPPrefix) *IPPrefix {
	return &prefix
}

func EqualIPPrefixes(a, b IPPrefix) bool {
	return a == b
}
