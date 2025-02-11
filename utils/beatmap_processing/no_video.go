package beatmap_processing

import (
	"archive/zip"
	"strings"
)

type NoVideoProcessor struct{}

func NewNoVideoProcessor() *NoVideoProcessor {
	return &NoVideoProcessor{}
}

func noVideoRule(s string) bool {
	return strings.HasSuffix(s, ".avi") || strings.HasSuffix(s, ".mp4") || strings.HasSuffix(s, ".flv")
}
func (p *NoVideoProcessor) Rule(s string, _ *zip.Reader) (bool, error) {
	return noVideoRule(s), nil
}

func (p *NoVideoProcessor) String() string {
	return "no_video"
}
