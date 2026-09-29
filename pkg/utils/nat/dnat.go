// Package nat validates the user-provided fields of NAT rules before they are
// handed to Kube-OVN, whose NAT gateway applies them with iptables.
package nat

import (
	"fmt"
	"strconv"
	"strings"
)

// ValidateDNAT checks the ports and protocol of a DNAT rule: each port is a
// number between 1 and 65535 or a range "a-b" with a <= b, and the protocol is
// "tcp", "udp" or empty (Kube-OVN default).
func ValidateDNAT(externalPort, internalPort, protocol string) error {
	if err := validatePort(externalPort); err != nil {
		return fmt.Errorf("invalid external port: %w", err)
	}
	if err := validatePort(internalPort); err != nil {
		return fmt.Errorf("invalid internal port: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "", "tcp", "udp":
		return nil
	default:
		return fmt.Errorf("invalid protocol %q: must be tcp or udp", protocol)
	}
}

func validatePort(value string) error {
	value = strings.TrimSpace(value)
	first, last, isRange := strings.Cut(value, "-")
	start, err := parsePort(first)
	if err != nil {
		return fmt.Errorf("%q: %w", value, err)
	}
	if !isRange {
		return nil
	}
	end, err := parsePort(last)
	if err != nil {
		return fmt.Errorf("%q: %w", value, err)
	}
	if start > end {
		return fmt.Errorf("%q: range start is greater than its end", value)
	}
	return nil
}

func parsePort(value string) (int, error) {
	if value == "" || len(value) > 5 {
		return 0, fmt.Errorf("must be a number between 1 and 65535")
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("must be a number between 1 and 65535")
		}
	}
	port, _ := strconv.Atoi(value)
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("must be a number between 1 and 65535")
	}
	return port, nil
}
