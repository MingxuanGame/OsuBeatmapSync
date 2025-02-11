package beatmap_processing

import (
	"archive/zip"
	"regexp"
)

type NoHitSoundProcessor struct{}

func NewNoHitSoundProcessor() *NoHitSoundProcessor {
	return &NoHitSoundProcessor{}
}

func noHitSoundRule(s string) bool {
	// https://osu.ppy.sh/wiki/en/Client/File_formats/osu_%28file_format%29#hitsounds
	// <sampleSet>-<objType><hitSound><index>.wav

	// <sampleSet> = normal, soft, drum
	// <objType> = slider, hit
	// <hitSound> = normal, whistle, finish, clap, slide, tick
	// <index> = ... (0 or 1 = empty)
	regex := regexp.MustCompile(`(normal|soft|drum)-(slider|hit)(normal|whistle|finish|clap|slide|tick)(\d+)?\.wav`)
	return regex.MatchString(s)
}

func (p *NoHitSoundProcessor) Rule(s string, _ *zip.Reader) (bool, error) {
	return noHitSoundRule(s), nil
}

func (p *NoHitSoundProcessor) String() string {
	return "no_hit_sound"
}
