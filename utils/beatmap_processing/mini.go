package beatmap_processing

import "archive/zip"

type MiniProcessor struct {
	readFileProcessor
}

func NewMiniProcessor() *MiniProcessor {
	return &MiniProcessor{}
}

func (p *MiniProcessor) String() string {
	return "mini"
}

func (p *MiniProcessor) Rule(s string, reader *zip.Reader) (bool, error) {
	if p.backgroundFile == nil || p.singleFile == nil || p.animationFiles == nil {
		err := p.init(reader)
		if err != nil {
			return false, err
		}
	}

	skip := noVideoRule(s) || noHitSoundRule(s)
	if skip {
		return true, nil
	}
	return p.isBackgroundFile(s) || p.isStoryboardFile(s), nil
}
