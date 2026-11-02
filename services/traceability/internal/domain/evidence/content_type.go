package evidence

import (
	"strings"

	"rutadeorigen/traceability/internal/platform/helper"
)

type ContentType string

const (
	ContentTypeUnknown ContentType = "unknown"
	ContentTypeJPEG    ContentType = "image/jpeg"
	ContentTypePNG     ContentType = "image/png"
	ContentTypeWebP    ContentType = "image/webp"
)

func ParseContentType(value string) ContentType {
	switch c := ContentType(strings.ToLower(helper.ApplyTrim(value))); c {
	case ContentTypeJPEG, ContentTypePNG, ContentTypeWebP:
		return c
	default:
		return ContentTypeUnknown
	}
}

func (c ContentType) IsValid() bool { return ParseContentType(string(c)) != ContentTypeUnknown }

func (c ContentType) String() string { return string(c) }
