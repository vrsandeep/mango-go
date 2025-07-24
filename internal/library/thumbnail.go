package library

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // Register PNG decoder

	"github.com/nfnt/resize"
)

const thumbnailWidth uint = 200
const thumbnailHeight uint = 300

// GenerateThumbnail takes raw image data, resizes it, encodes it as a
// Base64 JPEG, and returns it as a data URI string.
func GenerateThumbnail(imageData []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Get image dimensions
	imgHeight := img.Bounds().Dy()
	imgWidth := img.Bounds().Dx()

	var resizedImg image.Image
	if imgHeight > imgWidth {
		resizedImg = resize.Resize(thumbnailWidth, 0, img, resize.Lanczos3)
	} else {
		resizedImg = resize.Resize(0, thumbnailHeight, img, resize.Lanczos3)

	}

	var buf bytes.Buffer
	// Encode the resized image as a JPEG. Quality 75 is a good balance.
	if err := jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: 75}); err != nil {
		return "", fmt.Errorf("failed to encode jpeg: %w", err)
	}

	// Encode the byte buffer to a Base64 string.
	base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Format as a Data URI.
	return fmt.Sprintf("data:image/jpeg;base64,%s", base64Str), nil
}
