package api

import (
	"errors"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/vrsandeep/mango-go/internal/banners"
	"github.com/vrsandeep/mango-go/internal/store"
)

func (s *Server) appDataPath() string {
	if s.app != nil && s.app.Config() != nil && s.app.Config().AppData.Path != "" {
		return s.app.Config().AppData.Path
	}
	return "./data"
}

// SetAppDataPath sets the app data directory. Tests use this. Banner files are written to banners/ inside it.
func (s *Server) SetAppDataPath(path string) {
	if s.app == nil || s.app.Config() == nil {
		return
	}
	s.app.Config().AppData.Path = path
}

// SetBannerSaver overrides banner downloads in tests.
func (s *Server) SetBannerSaver(save func(seriesName, sourceURL string) (string, error)) {
	s.bannerSave = save
}

// localBanner downloads a remote banner into the app data banners directory.
// The bool is false when a download was required and failed, so the row stays unfetched.
func (s *Server) localBanner(seriesName, sourceURL string) (string, bool) {
	if sourceURL == "" {
		return "", true
	}
	path, err := s.saveBanner(seriesName, sourceURL)
	if err != nil {
		log.Printf("banner download for %q: %v", seriesName, err)
		return "", false
	}
	return path, true
}

func (s *Server) bannersDir() string {
	return filepath.Join(s.appDataPath(), banners.FolderName)
}

func (s *Server) saveBanner(seriesName, sourceURL string) (string, error) {
	if s.bannerSave != nil {
		return s.bannerSave(seriesName, sourceURL)
	}
	return banners.Save(s.bannersDir(), seriesName, sourceURL)
}

func (s *Server) handleGetBanner(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if !banners.ValidFileName(name) {
		RespondWithError(w, http.StatusBadRequest, "Invalid banner name")
		return
	}
	http.ServeFile(w, r, filepath.Join(s.bannersDir(), name))
}

// handleUploadFolderBanner stores a user-supplied banner in the app data banners directory.
// The image must be at least 800×300 and wider than it is tall.
// Wider images are scaled down to 2000 pixels.
func (s *Server) handleUploadFolderBanner(w http.ResponseWriter, r *http.Request) {
	folderID, err := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)
	if err != nil || folderID <= 0 {
		RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
		return
	}
	folder, err := s.store.GetFolder(folderID)
	if err != nil {
		if err == store.ErrFolderNotFound {
			RespondWithError(w, http.StatusNotFound, "Folder not found")
			return
		}
		RespondWithError(w, http.StatusInternalServerError, "Failed to load folder")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 20*1024*1024)
	file, _, err := r.FormFile("banner_file")
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid file upload")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Failed to read uploaded file")
		return
	}

	rel, err := banners.SaveBytes(s.bannersDir(), folder.Name, fileBytes)
	if err != nil {
		var dimErr *banners.DimensionError
		if errors.As(err, &dimErr) {
			RespondWithError(w, http.StatusBadRequest, dimErr.Error())
			return
		}
		if errors.Is(err, banners.ErrUnsupportedImage) {
			RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("Save banner for folder %d: %v", folderID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to save banner")
		return
	}

	previous := folder.Banner
	if err := s.store.UpdateFolderBanner(folderID, rel); err != nil {
		_ = banners.Remove(s.bannersDir(), rel)
		RespondWithError(w, http.StatusInternalServerError, "Failed to save banner")
		return
	}
	if previous != "" && previous != rel {
		_ = banners.Remove(s.bannersDir(), previous)
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{
		"message":     "Banner updated successfully.",
		"bannerImage": banners.PublicURL(rel),
	})
}

// handleRestoreFolderBanner drops a custom banner so the AniList banner is shown again.
func (s *Server) handleRestoreFolderBanner(w http.ResponseWriter, r *http.Request) {
	folderID, err := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)
	if err != nil || folderID <= 0 {
		RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
		return
	}
	folder, err := s.store.GetFolder(folderID)
	if err != nil {
		if err == store.ErrFolderNotFound {
			RespondWithError(w, http.StatusNotFound, "Folder not found")
			return
		}
		RespondWithError(w, http.StatusInternalServerError, "Failed to restore banner")
		return
	}
	if folder.Banner != "" {
		_ = banners.Remove(s.bannersDir(), folder.Banner)
	}
	if err := s.store.UpdateFolderBanner(folderID, ""); err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to restore banner")
		return
	}
	image := ""
	if cached, err := s.store.GetFolderAnilist(folderID); err == nil {
		image = banners.PublicURL(cached.BannerPath)
	}
	RespondWithJSON(w, http.StatusOK, map[string]string{"bannerImage": image})
}
