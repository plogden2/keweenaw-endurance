package rfid

import (
	"fmt"
	"regexp"
	"strconv"
)

// TagFamily is the ISO14443-A chip family used for logical UUID storage.
type TagFamily int

const (
	TagFamilyNone TagFamily = iota
	TagFamilyUltralight
	TagFamilyClassic1K
	TagFamilyUnsupported
)

func (f TagFamily) String() string {
	switch f {
	case TagFamilyUltralight:
		return "ultralight"
	case TagFamilyClassic1K:
		return "classic1k"
	case TagFamilyUnsupported:
		return "unsupported"
	default:
		return "none"
	}
}

// sakPattern matches Proxmark `hf 14a reader` SAK lines, e.g. "SAK: 00", "SAK: 0x08".
var sakPattern = regexp.MustCompile(`(?i)\bSAK\s*:\s*(?:0x)?([0-9a-f]{1,2})\b`)

// ClassifyISO14443A maps hf 14a reader / select transcript SAK to a tag family.
func ClassifyISO14443A(stdout string) (TagFamily, error) {
	m := sakPattern.FindStringSubmatch(stdout)
	if m == nil {
		return TagFamilyNone, nil
	}
	sak, err := strconv.ParseUint(m[1], 16, 8)
	if err != nil {
		return TagFamilyNone, fmt.Errorf("parse SAK %q: %w", m[1], err)
	}
	switch sak {
	case 0x00:
		return TagFamilyUltralight, nil
	case 0x08:
		return TagFamilyClassic1K, nil
	default:
		return TagFamilyUnsupported, nil
	}
}

func unsupportedTagTypeMessage(sak byte) string {
	return fmt.Sprintf(
		"unsupported tag type (SAK %02X) — use NTAG/Ultralight or Classic 1K",
		sak,
	)
}

func unsupportedTagTypeMessageFromStdout(stdout string) string {
	m := sakPattern.FindStringSubmatch(stdout)
	if m == nil {
		return "unsupported tag type — use NTAG/Ultralight or Classic 1K"
	}
	sak, err := strconv.ParseUint(m[1], 16, 8)
	if err != nil {
		return "unsupported tag type — use NTAG/Ultralight or Classic 1K"
	}
	return unsupportedTagTypeMessage(byte(sak))
}

func noTagSelectedMessage() string {
	return "No tag selected — hold one chip steady on the antenna and retry"
}

// familyFromStdout is a convenience wrapper used by dispatch paths.
func familyFromStdout(stdout string) TagFamily {
	fam, err := ClassifyISO14443A(stdout)
	if err != nil {
		return TagFamilyNone
	}
	return fam
}
