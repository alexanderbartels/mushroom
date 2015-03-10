package image

import (
	"log"
	"image"
	"strconv"
	"bytes"
	"github.com/disintegration/imaging"
)

// Interface for Image processors
type Processor interface {
	Process(actions map[string]string) (bytes.Buffer, error)
}

type DefaultProcessor struct {
	Img *image.Image
}

// currently the width and height action are supported.
// Other action will be ignored
func (p *DefaultProcessor) Process(actions map[string]string) (*bytes.Buffer, error) {
	// height must always be an integer
	height, hErr := strconv.Atoi(actions["height"])
	if hErr != nil {
		log.Println("Unsupported Param for height: ", actions["height"])
		// use default as fallback
		height = 0
	}

	// width must always be an integer
	width, wErr := strconv.Atoi(actions["width"])
	if wErr != nil {
		log.Println("Unsupported Param for width: ", actions["width"])
		// use default as fallback
		width = 0
	}

	// manipulate image only if needed
	if width > 0 || height > 0 {
		*p.Img = imaging.Resize(*p.Img, width, height, imaging.Box)
	}

	// write image to buffer
	buf := new(bytes.Buffer)
	encodeErr := imaging.Encode(buf, *p.Img, imaging.PNG) // TODO Encoding PNG JPG webP

	return buf, encodeErr
}
