package version

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/khulnasoft/go-version-lib/pkg/versionrange"
)

// ToVersionRange lowers the specifiers to the generic versionrange model,
// enabling cross-ecosystem range algebra (Intersect, Union, Complement).
//
// Prefix specifiers ("==1.4.*") are expanded to >=lower <upper bounds.
// "~=" is expanded per PEP 440 for plain release versions; versions with
// epochs, pre/post/dev releases, or local segments return an error.
// "===" (arbitrary equality) is not representable as a version comparison
// and also returns an error.
func (ss Specifiers) ToVersionRange() (versionrange.Range[Version], error) {
	less := func(a, b Version) int { return a.Compare(b) }
	var sets []versionrange.Set[Version]
	for _, group := range ss.specifiers {
		var cmps []versionrange.Comparator[Version]
		for _, s := range group {
			cc, err := s.toComparators()
			if err != nil {
				return versionrange.Range[Version]{}, err
			}
			cmps = append(cmps, cc...)
		}
		sets = append(sets, versionrange.NewSet(cmps...))
	}
	return versionrange.New(less, sets...), nil
}

func (s specifier) toComparators() ([]versionrange.Comparator[Version], error) {
	prefix := strings.HasSuffix(s.version, ".*")
	ver := strings.TrimSuffix(s.version, ".*")

	// "==1.4.*" style prefix matching is only meaningful for equality ops.
	if prefix && s.operator.op != "" && s.operator.op != "=" && s.operator.op != "==" {
		return nil, fmt.Errorf("prefix version %q only supports equality operators, got %q", s.original, s.operator.op)
	}

	cmp := func(op versionrange.Op, v Version) []versionrange.Comparator[Version] {
		return []versionrange.Comparator[Version]{{Op: op, Value: v}}
	}

	if prefix {
		lo, hi, err := prefixBounds(ver)
		if err != nil {
			return nil, err
		}
		if s.operator.op == "!=" {
			return nil, fmt.Errorf("!= with prefix version %q is not representable as a range", s.original)
		}
		return []versionrange.Comparator[Version]{
			{Op: versionrange.OpGE, Value: lo},
			{Op: versionrange.OpLT, Value: hi},
		}, nil
	}

	v, err := Parse(ver)
	if err != nil {
		return nil, fmt.Errorf("invalid specifier version %q: %w", s.original, err)
	}

	switch s.operator.op {
	case "", "=", "==":
		return cmp(versionrange.OpEQ, v), nil
	case "!=":
		return cmp(versionrange.OpNE, v), nil
	case ">":
		return cmp(versionrange.OpGT, v), nil
	case "<":
		return cmp(versionrange.OpLT, v), nil
	case ">=":
		return cmp(versionrange.OpGE, v), nil
	case "<=":
		return cmp(versionrange.OpLE, v), nil
	case "~=":
		return compatibleBounds(s.original, v)
	case "===":
		return nil, fmt.Errorf("arbitrary equality === is not representable as a version range: %q", s.original)
	default:
		return nil, fmt.Errorf("unsupported operator %q", s.operator.op)
	}
}

// compatibleBounds expands "~=V.N" into >=V.N <V'.* per PEP 440. Only plain
// release versions (optionally with an epoch) are supported.
func compatibleBounds(original string, v Version) ([]versionrange.Comparator[Version], error) {
	if v.IsPreRelease() || v.IsPostRelease() || v.Local() != "" || strings.Contains(original, "!") {
		return nil, fmt.Errorf("~= with %q is only supported for plain release versions", original)
	}
	release := v.BaseVersion()
	segs := strings.Split(release, ".")
	if len(segs) < 2 {
		return nil, fmt.Errorf("~= requires at least two release segments: %q", original)
	}
	// drop the last segment and bump the new last one
	last, err := strconv.Atoi(segs[len(segs)-2])
	if err != nil {
		return nil, fmt.Errorf("~= with %q is only supported for numeric release segments", original)
	}
	upper := strings.Join(append(segs[:len(segs)-2], strconv.Itoa(last+1)), ".")
	hi, err := Parse(upper)
	if err != nil {
		return nil, fmt.Errorf("~= upper bound %q: %w", upper, err)
	}
	return []versionrange.Comparator[Version]{
		{Op: versionrange.OpGE, Value: v},
		{Op: versionrange.OpLT, Value: hi},
	}, nil
}

// prefixBounds expands a "1.4.*"-style prefix into >=lower <upper bounds.
func prefixBounds(prefix string) (Version, Version, error) {
	lo, err := Parse(prefix)
	if err != nil {
		return Version{}, Version{}, fmt.Errorf("invalid prefix version %q: %w", prefix, err)
	}
	segs := strings.Split(prefix, ".")
	last, err := strconv.Atoi(segs[len(segs)-1])
	if err != nil {
		return Version{}, Version{}, fmt.Errorf("non-numeric prefix segment in %q", prefix)
	}
	hiStr := strings.Join(append(segs[:len(segs)-1], strconv.Itoa(last+1)), ".")
	hi, err := Parse(hiStr)
	if err != nil {
		return Version{}, Version{}, fmt.Errorf("invalid prefix upper bound %q: %w", hiStr, err)
	}
	return lo, hi, nil
}
