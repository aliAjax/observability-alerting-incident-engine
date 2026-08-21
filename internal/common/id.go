package common

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// NewID returns a random identifier with an optional short prefix.
func NewID(prefix string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s_%x", prefix, time.Now().UnixNano())
	}
	if prefix == "" {
		return hex.EncodeToString(b[:])
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}

func NewTraceID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func NormalizeID(raw string, prefix string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return NewID(prefix)
	}
	if strings.HasPrefix(raw, prefix+"_") {
		return raw
	}
	return prefix + "_" + raw
}
