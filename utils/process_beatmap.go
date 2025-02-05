package utils

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
)

func getBackgroundFile(osu string) string {
	lines := strings.Split(osu, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "0,0,") {
			file := strings.Split(line, ",")[2]         // get filename
			file = strings.Replace(file, "\"", "", -1)  // remove "
			file = strings.Replace(file, "\\", "/", -1) // replace \ with /
			return file
		}
	}
	return ""
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
			singleFile = append(singleFile, file)
		} else if strings.HasPrefix(line, "Animation,") {
			file := strings.Split(line, ",")[3]         // get filename
			file = strings.Replace(file, "\"", "", -1)  // remove "
			file = strings.Replace(file, "\\", "/", -1) // replace \ with /
			file = cutImageExt(file)                    // remove image ext
			animationFiles = append(animationFiles, file)
		}
	}
	return
}

func ProcessBeatmapset(full []byte) (noVideo, mini []byte, err error) {
	zipReader := bytes.NewReader(full)
	reader, err := zip.NewReader(zipReader, int64(len(full)))
	if err != nil {
		return nil, nil, fmt.Errorf("cannot read zip file: %w", err)
	}

	var noVideoBuf, miniBuf bytes.Buffer
	noVideoWriter := zip.NewWriter(&noVideoBuf)
	miniWriter := zip.NewWriter(&miniBuf)

	// process storyboard
	var singleFile []string
	var animationFiles []string
	var backgroundFile []string
	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, ".osu") || strings.HasSuffix(file.Name, ".osb") {
			src, err := file.Open()
			if err != nil {
				return nil, nil, fmt.Errorf("cannot open file %s: %w", file.Name, err)
			}
			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, src)
			if err != nil {
				return nil, nil, fmt.Errorf("cannot read file %s: %w", file.Name, err)
			}
			err = src.Close()
			if err != nil {
				return nil, nil, fmt.Errorf("cannot close file %s: %w", file.Name, err)
			}
			singleFile, animationFiles = getStoryBoardFile(buf.String())
			backgroundFile = append(backgroundFile, getBackgroundFile(buf.String()))
		}
	}

	for _, file := range reader.File {
		switch {
		case strings.HasSuffix(file.Name, ".avi") || strings.HasSuffix(file.Name, ".mp4") || strings.HasSuffix(file.Name, ".flv"):
			continue
		case !In(backgroundFile, file.Name) && (strings.HasSuffix(file.Name, ".osb") || In(singleFile, file.Name) || In(animationFiles, file.Name) || In(animationFiles, removeNumber(cutImageExt(file.Name)))):
			err := copyFile(file, noVideoWriter)
			if err != nil {
				return nil, nil, fmt.Errorf("copy file %s error: %w", file.Name, err)
			}
			continue
		default:
			err := copyFile(file, noVideoWriter)
			if err != nil {
				return nil, nil, fmt.Errorf("copy file %s error: %w", file.Name, err)
			}
			err = copyFile(file, miniWriter)
			if err != nil {
				return nil, nil, fmt.Errorf("copy file %s error: %w", file.Name, err)
			}
		}
	}

	err = noVideoWriter.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot close noVideo writer: %w", err)
	}
	err = miniWriter.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot close mini writer: %w", err)
	}

	noVideo = noVideoBuf.Bytes()
	mini = miniBuf.Bytes()
	return
}
