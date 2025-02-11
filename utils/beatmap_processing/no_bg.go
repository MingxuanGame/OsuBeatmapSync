package beatmap_processing

import (
	"archive/zip"
)

type NoBackgroundProcessor struct {
	readFileProcessor
}

func NewNoBackgroundProcessor() *NoBackgroundProcessor {
	return &NoBackgroundProcessor{}
}

func (p *NoBackgroundProcessor) String() string {
	return "no_background"
}

func (p *NoBackgroundProcessor) Rule(s string, reader *zip.Reader) (bool, error) {
	if p.backgroundFile == nil || p.singleFile == nil || p.animationFiles == nil {
		err := p.init(reader)
		if err != nil {
			return false, err
		}
	}
	return p.isBackgroundFile(s) && !p.isStoryboardFile(s), nil
}
