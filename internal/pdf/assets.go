package pdf

import (
	"fmt"
	"os"

	"github.com/ledongthuc/pdf"
)

// Assets handles PDF asset extraction operations
type Assets struct {
	maxFileSize int64
	validator   *Validator
}

// NewAssets creates a new PDF assets extractor with the specified constraints
func NewAssets(maxFileSize int64) *Assets {
	return &Assets{
		maxFileSize: maxFileSize,
		validator:   NewValidator(maxFileSize),
	}
}

// ExtractAssets extracts visual assets (images) from a PDF file
func (a *Assets) ExtractAssets(req PDFAssetsFileRequest) (*PDFAssetsFileResult, error) {
	if req.Path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	// Check if file exists and get basic info
	fileInfo, err := os.Stat(req.Path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", req.Path)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot access file: %w", err)
	}

	// Validate file
	if err := a.validator.ValidateFileInfo(req.Path, fileInfo); err != nil {
		return nil, err
	}

	// Open and parse PDF
	f, r, err := pdf.Open(req.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	var images []ImageInfo

	// Scan through pages looking for images
	images = a.extractImagesFromPages(r)

	result := &PDFAssetsFileResult{
		Path:       req.Path,
		Images:     images,
		TotalCount: len(images),
	}

	return result, nil
}

// extractImagesFromPages scans all pages for image objects
func (a *Assets) extractImagesFromPages(r *pdf.Reader) []ImageInfo {
	var images []ImageInfo

	for pageNum := 1; pageNum <= r.NumPage(); pageNum++ {
		pageImages := a.extractImagesFromPage(r, pageNum)
		images = append(images, pageImages...)
	}

	return images
}

// extractImagesFromPage extracts images from a specific page
func (a *Assets) extractImagesFromPage(r *pdf.Reader, pageNum int) []ImageInfo {
	var images []ImageInfo

	defer func() {
		// Recover from any panics during image extraction
		if recover() != nil {
			// Image extraction failed for this page, continue with others
		}
	}()

	page := r.Page(pageNum)
	if page.V.IsNull() {
		return images
	}

	// Get page resources
	resources := page.V.Key("Resources")
	if resources.IsNull() {
		return images
	}

	// Get XObject dictionary (where images are typically stored)
	xObjects := resources.Key("XObject")
	if xObjects.IsNull() || xObjects.Kind() != pdf.Dict {
		return images
	}

	// Iterate through XObjects looking for images
	for _, key := range xObjects.Keys() {
		obj := xObjects.Key(key)
		if obj.IsNull() {
			continue
		}

		// Check if this XObject is an image
		subtype := obj.Key("Subtype")
		if subtype.IsNull() || subtype.Name() != "Image" {
			continue
		}

		// Extract image information
		imageInfo := a.extractImageInfo(obj, pageNum)
		if imageInfo != nil {
			images = append(images, *imageInfo)
		}
	}

	return images
}

// extractImageInfo extracts information from an image XObject
func (a *Assets) extractImageInfo(obj pdf.Value, pageNum int) *ImageInfo {
	defer func() {
		// Recover from any panics during image info extraction
		if recover() != nil {
			// Failed to extract this image info
		}
	}()

	imageInfo := &ImageInfo{
		PageNumber: pageNum,
		Width:      0,
		Height:     0,
		Format:     "unknown",
		Size:       0,
	}

	// Extract width
	if width := obj.Key("Width"); !width.IsNull() {
		imageInfo.Width = int(width.Int64())
	}

	// Extract height
	if height := obj.Key("Height"); !height.IsNull() {
		imageInfo.Height = int(height.Int64())
	}

	// Extract format from Filter
	if filter := obj.Key("Filter"); !filter.IsNull() {
		filterName := filter.Name()
		imageInfo.Format = a.normalizeImageFormat(filterName)
	}

	// Try to extract color space information
	if colorSpace := obj.Key("ColorSpace"); !colorSpace.IsNull() {
		if imageInfo.Format == "unknown" {
			// Sometimes color space gives us hints about the format
			csName := colorSpace.Name()
			if csName != "" {
				imageInfo.Format = csName
			}
		}
	}

	// Extract bits per component
	bitsPerComponent := 8 // default
	if bpc := obj.Key("BitsPerComponent"); !bpc.IsNull() {
		bitsPerComponent = int(bpc.Int64())
	}

	// Estimate size (this is approximate)
	if imageInfo.Width > 0 && imageInfo.Height > 0 {
		// Rough estimation: width * height * (bits per component / 8) * components
		// Assume 3 components (RGB) for estimation
		estimatedSize := int64(imageInfo.Width * imageInfo.Height * (bitsPerComponent / 8) * 3)
		imageInfo.Size = estimatedSize
	}

	// Only return valid image info
	if imageInfo.Width > 0 && imageInfo.Height > 0 {
		return imageInfo
	}

	return nil
}

// normalizeImageFormat converts PDF filter names to more readable format names
func (a *Assets) normalizeImageFormat(filterName string) string {
	switch filterName {
	case "DCTDecode":
		return "JPEG"
	case "JPXDecode":
		return "JPEG2000"
	case "CCITTFaxDecode":
		return "TIFF/Fax"
	case "JBIG2Decode":
		return "JBIG2"
	case "FlateDecode":
		return "PNG/Deflate"
	case "LZWDecode":
		return "LZW"
	case "RunLengthDecode":
		return "RLE"
	default:
		if filterName != "" {
			return filterName
		}
		return "unknown"
	}
}

// GetSupportedFormats returns a list of image formats that can be detected
func (a *Assets) GetSupportedFormats() []string {
	return []string{
		"JPEG",
		"JPEG2000",
		"TIFF/Fax",
		"JBIG2",
		"PNG/Deflate",
		"LZW",
		"RLE",
	}
}
