package version

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	versionRegexp = matchingRegexp{regexp.MustCompile(`\A(?P<release>[0-9A-Za-z_\.]+)(\-(?P<pre_release>[0-9A-Za-z_\-\.]+))?(\+(?P<post_release>[0-9A-Za-z_\-\.]+))?\z`)}
)

type matchingRegexp struct {
	*regexp.Regexp
}

type Version struct {
	Release, PreRelease, PostRelease VersionSegment

	Segments []VersionSegment
}

func MustNewVersionFromString(v string) Version {
	ver, err := NewVersionFromString(v)
	if err != nil {
		panic(fmt.Sprintf("Invalid version '%s': %s", v, err))
	}

	return ver
}

func NewVersionFromString(v string) (Version, error) {
	var err error

	if len(v) == 0 {
		return Version{}, errors.New("Expected version to be non-empty string")
	}

	captures := versionRegexp.FindStringSubmatchMap(v)
	if len(captures) == 0 {
		errMsg := fmt.Sprintf("Expected version '%s' to match version format", v)
		return Version{}, errors.New(errMsg)
	}

	release := VersionSegment{}
	preRelease := VersionSegment{}
	postRelease := VersionSegment{}

	if releaseStr, ok := captures["release"]; ok {
		release, err = NewVersionSegmentFromString(releaseStr)
		if err != nil {
			return Version{}, err
		}
	}

	if preReleaseStr, ok := captures["pre_release"]; ok {
		preRelease, err = NewVersionSegmentFromString(preReleaseStr)
		if err != nil {
			return Version{}, err
		}
	}

	if postReleaseStr, ok := captures["post_release"]; ok {
		postRelease, err = NewVersionSegmentFromString(postReleaseStr)
		if err != nil {
			return Version{}, err
		}
	}

	return NewVersion(release, preRelease, postRelease)
}

func NewVersion(release, preRelease, postRelease VersionSegment) (Version, error) {
	if release.Empty() {
		return Version{}, errors.New("Expected to non-empty release segment for constructing version")
	}

	version := Version{
		Release:     release,
		PreRelease:  preRelease,
		PostRelease: postRelease,
		Segments:    []VersionSegment{release, preRelease, postRelease},
	}

	return version, nil
}

func (v Version) IncrementRelease() (Version, error) {
	incRelease, err := v.Release.Increment()
	if err != nil {
		return Version{}, err
	}

	return NewVersion(incRelease, VersionSegment{}, VersionSegment{})
}

func (v Version) IncrementPostRelease(defaultPostRelease VersionSegment) (Version, error) {
	var newPostRelease VersionSegment
	var err error

	if defaultPostRelease.Empty() {
		return Version{}, errors.New("Expected default post relase to be non-empty")
	}

	if v.PostRelease.Empty() {
		newPostRelease = defaultPostRelease.Copy()
	} else {
		newPostRelease, err = v.PostRelease.Increment()
		if err != nil {
			return Version{}, err
		}
	}

	return NewVersion(v.Release.Copy(), v.PreRelease.Copy(), newPostRelease)
}

func (v Version) Empty() bool { return len(v.Segments) == 0 }

func (v Version) String() string { return v.AsString() }

func (v Version) AsString() string {
	result := v.Release.AsString()

	if !v.PreRelease.Empty() {
		result += "-" + v.PreRelease.AsString()
	}

	if !v.PostRelease.Empty() {
		result += "+" + v.PostRelease.AsString()
	}

	return result
}

func (v Version) Compare(other Version) int {
	result := v.Release.Compare(other.Release)
	if result != 0 {
		return result
	}

	if !v.PreRelease.Empty() || !other.PreRelease.Empty() {
		if v.PreRelease.Empty() {
			return 1
		}
		if other.PreRelease.Empty() {
			return -1
		}
		result = v.PreRelease.Compare(other.PreRelease)
		if result != 0 {
			return result
		}
	}

	if !v.PostRelease.Empty() || !other.PostRelease.Empty() {
		if v.PostRelease.Empty() {
			return -1
		}
		if other.PostRelease.Empty() {
			return 1
		}
		result = v.PostRelease.Compare(other.PostRelease)
		if result != 0 {
			return result
		}
	}

	return 0
}

func (v Version) IsEq(other Version) bool { return v.Compare(other) == 0 }
func (v Version) IsGt(other Version) bool { return v.Compare(other) == 1 }
func (v Version) IsLt(other Version) bool { return v.Compare(other) == -1 }

func (r *matchingRegexp) FindStringSubmatchMap(s string) map[string]string {
	captures := map[string]string{}

	match := r.FindStringSubmatch(s)
	if match == nil {
		return captures
	}

	for i, name := range r.SubexpNames() {
		// 0 is a whole regex
		if i == 0 || name == "" || match[i] == "" {
			continue
		}

		captures[name] = match[i]
	}

	return captures
}
