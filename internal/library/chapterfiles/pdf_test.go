package chapterfiles

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPDFHandler_InspectAndPage(t *testing.T) {
	pdfPath := filepath.Join("testdata", "test.pdf")
	pages, first, err := InspectChapterFile(context.Background(), pdfPath)
	require.NoError(t, err)
	require.Greater(t, len(pages), 0)
	require.NotEmpty(t, first)
	require.True(t, bytes.HasPrefix(first, []byte{0x89, 0x50, 0x4e, 0x47}), "first page should be PNG")

	data, name, err := GetChapterPage(context.Background(), pdfPath, 0)
	require.NoError(t, err)
	require.Equal(t, syntheticPageFileName, name)
	require.NotEmpty(t, data)
	require.True(t, bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4e, 0x47}))
}

func TestPDFHandler_PageOutOfRange(t *testing.T) {
	pdfPath := filepath.Join("testdata", "test.pdf")
	pages, _, err := InspectChapterFile(context.Background(), pdfPath)
	require.NoError(t, err)
	_, _, err = GetChapterPage(context.Background(), pdfPath, len(pages))
	require.Error(t, err)
}
