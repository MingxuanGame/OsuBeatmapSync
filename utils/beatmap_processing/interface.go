package beatmap_processing

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/MingxuanGame/OsuBeatmapSync/utils"
	"io"
	"strings"
)

type Processor interface {
	Rule(s string, reader *zip.Reader) (bool, error)
	String() string
}

type readFileProcessor struct {
	singleFile     []string
	animationFiles []string
	backgroundFile []string
}

func getBackgroundFile(osu string) string {
	lines := strings.Split(osu, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "0,0,") {
			file := strings.Split(line, ",")[2]         // get filename
			file = strings.Replace(file, "\"", "", -1)  // remove "
			file = strings.Replace(file, "\\", "/", -1) // replace \ with /
			return strings.ToLower(file)
		}
	}
	return ""
}

func cutImageExt(s string) string {
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".bmp", ".gif"} {
		s = strings.Replace(s, ext, "", -1)
	}
	return s
}

func removeNumber(s string) string {
	for i := 0; i < 10; i++ {
		s = strings.Replace(s, fmt.Sprintf("%d", i), "", -1)
	}
	return s
}

func getStoryBoardFile(osu string) (singleFile, animationFiles []string) {
	lines := strings.Split(osu, "\n")
	for _, line := range lines {
		// https://osu.ppy.sh/wiki/zh/Storyboard/Scripting/Objects
		// https://osu.ppy.sh/wiki/zh/Storyboard/Scripting/Audio
		if strings.HasPrefix(line, "Sprite,") || strings.HasPrefix(line, "Sample,") {
			file := strings.Split(line, ",")[3]         // get filename
			file = strings.Replace(file, "\"", "", -1)  // remove "
			file = strings.Replace(file, "\\", "/", -1) // replace \ with /
			singleFile = append(singleFile, strings.ToLower(file))
		} else if strings.HasPrefix(line, "Animation,") {
			file := strings.Split(line, ",")[3]         // get filename
			file = strings.Replace(file, "\"", "", -1)  // remove "
			file = strings.Replace(file, "\\", "/", -1) // replace \ with /
			file = cutImageExt(file)                    // remove image ext
			animationFiles = append(animationFiles, strings.ToLower(file))
		}
	}
	return
}

func (p *readFileProcessor) init(reader *zip.Reader) error {
	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, ".osu") || strings.HasSuffix(file.Name, ".osb") {
			src, err := file.Open()
			if err != nil {
				return err
			}
			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, src)
			if err != nil {
				return err
			}
			err = src.Close()
			if err != nil {
				return err
			}
			singleFile, animationFiles := getStoryBoardFile(buf.String())
			p.singleFile = append(p.singleFile, singleFile...)
			p.animationFiles = append(p.animationFiles, animationFiles...)
			p.backgroundFile = append(p.backgroundFile, getBackgroundFile(buf.String()))
		}
	}
	return nil
}

func (p *readFileProcessor) isStoryboardFile(s string) bool {
	return strings.HasSuffix(s, ".osb") ||
		utils.In(p.singleFile, strings.ToLower(s)) ||
		utils.In(p.animationFiles, strings.ToLower(removeNumber(cutImageExt(s))))
}

func (p *readFileProcessor) isBackgroundFile(s string) bool {
	return utils.In(p.backgroundFile, strings.ToLower(s))
}

func copyFile(srcFile *zip.File, dstZip *zip.Writer) error {
	dst, err := dstZip.Create(srcFile.Name)
	if err != nil {
		return err
	}
	src, err := srcFile.Open()
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}
	err = src.Close()
	if err != nil {
		return err
	}
	return nil
}

func process(reader *zip.Reader, writer *zip.Writer, p Processor) error {
	for _, file := range reader.File {
		skip, err := p.Rule(file.Name, reader)
		if err != nil {
			return fmt.Errorf("rule error: %w", err)
		}
		if skip {
			continue
		}
		err = copyFile(file, writer)
		if err != nil {
			return fmt.Errorf("copy file %s error: %w", file.Name, err)
		}
	}
	return nil
}

func Process(p Processor, full []byte) ([]byte, error) {
	zipReader := bytes.NewReader(full)
	reader, err := zip.NewReader(zipReader, int64(len(full)))
	if err != nil {
		return nil, fmt.Errorf("cannot read zip file: %w", err)
	}
	var noVideoBuf bytes.Buffer
	noVideoWriter := zip.NewWriter(&noVideoBuf)
	err = process(reader, noVideoWriter, p)
	if err != nil {
		return nil, fmt.Errorf("process %s error: %w", p, err)
	}
	err = noVideoWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("cannot close %s writer: %w", p, err)
	}
	return noVideoBuf.Bytes(), nil
}
