package service

import (
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

var (
	upstreamResponseBodyPattern     = regexp.MustCompile(`(?is)\b(?:response[_ -]?)?body\s*[:=]\s*.*$`)
	upstreamTrailingMetadataPattern = regexp.MustCompile(`(?s)\s+\((?:\{.*\}|\[.*\])\)\s*$`)
	upstreamBearerPattern           = regexp.MustCompile(`(?i)\bbearer\s+[a-z0-9._~+/=-]+`)
	upstreamCredentialPattern       = regexp.MustCompile(`(?i)((?:["']?)\b(?:api[_ -]?key|access[_ -]?token|refresh[_ -]?token|authorization|client[_ -]?secret|secret|token)\b(?:["']?)\s*[:=]\s*)(?:"[^"]*"|'[^']*'|[^\s,;\}\]]+)`)
)

func SanitizeUpstreamErrorMessage(message string, secrets ...string) string {
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if len(secret) < 4 {
			continue
		}
		message = strings.ReplaceAll(message, secret, "***")
	}
	message = upstreamTrailingMetadataPattern.ReplaceAllString(message, " [metadata omitted]")
	message = upstreamResponseBodyPattern.ReplaceAllString(message, "body: [omitted]")
	message = upstreamBearerPattern.ReplaceAllString(message, "Bearer ***")
	message = upstreamCredentialPattern.ReplaceAllString(message, `${1}***`)
	return strings.TrimSpace(common.MaskSensitiveInfo(message))
}

func NormalizeChannelErrorMessage(message string) string {
	return common.NormalizeChannelErrorMessage(message)
}
