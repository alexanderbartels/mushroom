package image

import (
	"image"
	"github.com/disintegration/imaging"
)

// interface for different image providers
// amazon s3, google blobstore, HTTP Providers are all possible options/variants
// currently the FileProvider is enough :-)
type Provider interface {
	Provide() (image.Image, error)
}

// provider for images that will be loaded from the disk
type FileProvider struct {
	Src string
}

func (fp *FileProvider) Provide() (image.Image, error) {
	return imaging.Open(fp.Src)
}

