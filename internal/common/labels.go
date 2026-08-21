package common

import (
	"encoding/json"
	"sort"
	"strings"
)

type Labels map[string]string

func (l Labels) Clone() Labels {
	out := make(Labels, len(l))
	for k, v := range l {
		out[k] = v
	}
	return out
}

func (l Labels) Merge(other Labels) Labels {
	out := l.Clone()
	for k, v := range other {
		out[k] = v
	}
	return out
}

func (l Labels) SortedKeys() []string {
	keys := make([]string, 0, len(l))
	for k := range l {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (l Labels) String() string {
	if len(l) == 0 {
		return "{}"
	}
	keys := l.SortedKeys()
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+l[k])
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func (l Labels) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string(l))
}

func (l *Labels) UnmarshalJSON(data []byte) error {
	m := map[string]string{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*l = m
	return nil
}

// Match returns true when every matcher entry equals the corresponding label.
func (l Labels) Match(matcher Labels) bool {
	for k, want := range matcher {
		if got, ok := l[k]; !ok || got != want {
			return false
		}
	}
	return true
}

func (l Labels) MatchesAny(matchers []Labels) bool {
	for _, m := range matchers {
		if l.Match(m) {
			return true
		}
	}
	return false
}
