package beatmap_processing

import (
	"archive/zip"
)

type NoStoryboardProcessor struct {
	readFileProcessor
}

func NewNoStoryboardProcessor() *NoStoryboardProcessor {
	return &NoStoryboardProcessor{}
}

func (p *NoStoryboardProcessor) String() string {
	return "no_storyboard"
}

func (p *NoStoryboardProcessor) Rule(s string, reader *zip.Reader) (bool, error) {
	if p.backgroundFile == nil || p.singleFile == nil || p.animationFiles == nil {
		err := p.init(reader)
		if err != nil {
			return false, err
		}
	}

	return !p.isBackgroundFile(s) && p.isStoryboardFile(s), nil
}
