package analysis

type EscapeVal int

const (
	STACK EscapeVal = 0
	HEAP  EscapeVal = 1
)

func (e EscapeVal) String() string {
	switch e {
	case STACK:
		return "STACK"
	case HEAP:
		return "HEAP"
	default:
		return "UNKNOWN"
	}
}

func Join(a, b EscapeVal) EscapeVal {
	if a > b {
		return a
	}
	return b
}

func JoinAll(vals []EscapeVal) EscapeVal {
	result := STACK
	for _, v := range vals {
		result = Join(result, v)
		if result == HEAP {
			return HEAP
		}
	}
	return result
}

type EscapeMap map[string]EscapeVal

func (m EscapeMap) Get(name string) EscapeVal {
	if v, ok := m[name]; ok {
		return v
	}
	return STACK
}

func (m EscapeMap) Set(name string, val EscapeVal) bool {
	if m.Get(name) == val {
		return false
	}
	m[name] = val
	return true
}

func (m EscapeMap) MarkHeap(name string) bool {
	return m.Set(name, HEAP)
}

func (m EscapeMap) Clone() EscapeMap {
	clone := make(EscapeMap, len(m))
	for k, v := range m {
		clone[k] = v
	}
	return clone
}

func (m EscapeMap) Equal(other EscapeMap) bool {
	if len(m) != len(other) {
		return false
	}
	for k, v := range m {
		if other.Get(k) != v {
			return false
		}
	}
	return true
}

func MergeInto(dst, src EscapeMap) bool {
	changed := false
	for k, v := range src {
		if dst.Set(k, Join(dst.Get(k), v)) {
			changed = true
		}
	}
	return changed
}
