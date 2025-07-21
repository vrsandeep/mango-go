package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/util"
)

var ErrFolderNotFound = errors.New("folder not found")

// CreateFolder inserts a new folder into the database.
func (s *Store) CreateFolder(path, name string, parentID *int64) (*models.Folder, error) {
	query := "INSERT INTO folders (path, name, parent_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)"
	now := time.Now()
	res, err := s.db.Exec(query, path, name, parentID, now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &models.Folder{ID: id, Path: path, Name: name, ParentID: parentID}, nil
}

func (s *Store) GetFolderByPath(path string) (*models.Folder, error) {
	query := "SELECT id, path, name, parent_id, thumbnail, created_at, updated_at FROM folders WHERE path = ?"
	var folder models.Folder
	var parentID sql.NullInt64
	var thumbnail sql.NullString
	err := s.db.QueryRow(query, path).Scan(&folder.ID, &folder.Path, &folder.Name, &parentID, &thumbnail, &folder.CreatedAt, &folder.UpdatedAt)

	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		folder.ParentID = &parentID.Int64
	}
	folder.Thumbnail = thumbnail.String
	return &folder, nil
}

// GetFolder retrieves a single folder by its ID.
func (s *Store) GetFolder(id int64) (*models.Folder, error) {
	var folder models.Folder
	var parentID sql.NullInt64
	var thumbnail sql.NullString
	if id == 0 {
		return nil, fmt.Errorf("folder ID cannot be 0")
	}
	query := "SELECT id, path, name, parent_id, thumbnail, created_at, updated_at FROM folders WHERE id = ?"
	err := s.db.QueryRow(query, id).Scan(&folder.ID, &folder.Path, &folder.Name, &parentID, &thumbnail, &folder.CreatedAt, &folder.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFolderNotFound
		}
		return nil, err
	}
	if parentID.Valid {
		folder.ParentID = &parentID.Int64
	}
	folder.Thumbnail = thumbnail.String

	// Fetch associated tags
	tagQuery := "SELECT t.id, t.name FROM tags t JOIN folder_tags ft ON t.id = ft.tag_id WHERE ft.folder_id = ?"
	rows, err := s.db.Query(tagQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			continue
		}
		folder.Tags = append(folder.Tags, &tag)
	}

	return &folder, nil
}

// GetAllFoldersByPath retrieves all folders and maps them by their full path for efficient lookup.
func (s *Store) GetAllFoldersByPath() (map[string]*models.Folder, error) {
	rows, err := s.db.Query("SELECT id, path, name, parent_id FROM folders")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	folderMap := make(map[string]*models.Folder)
	for rows.Next() {
		var folder models.Folder
		var parentID sql.NullInt64
		if err := rows.Scan(&folder.ID, &folder.Path, &folder.Name, &parentID); err != nil {
			return nil, err
		}
		if parentID.Valid {
			folder.ParentID = &parentID.Int64
		}
		folderMap[folder.Path] = &folder
	}
	return folderMap, nil
}

// DeleteFolder removes a folder by its ID.
func (s *Store) DeleteFolder(id int64) error {
	_, err := s.db.Exec("DELETE FROM folders WHERE id = ?", id)
	return err
}

// UpdateAllFolderThumbnails recursively finds the first chapter in a folder's subtree
// and sets its thumbnail as the folder's thumbnail.
func (s *Store) UpdateAllFolderThumbnails() error {
	// This is a complex operation. A simplified approach:
	// 1. Get all chapters with their folder IDs and thumbnails.
	// 2. Group them by folder ID.
	// 3. For each folder, find the "first" chapter via natural sort.
	// 4. Update that folder's thumbnail.
	// 5. Recursively do this for parent folders.

	rows, err := s.db.Query("SELECT id FROM folders")
	if err != nil {
		return err
	}
	defer rows.Close()

	var folderIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		folderIDs = append(folderIDs, id)
	}

	for _, folderID := range folderIDs {
		s.updateSingleFolderThumbnail(folderID)
	}
	return nil
}

// updateSingleFolderThumbnail is a helper for the above function.
func (s *Store) updateSingleFolderThumbnail(folderID int64) {
	// This query recursively finds all chapters within a folder and its subfolders.
	query := `
		WITH RECURSIVE folder_tree(id) AS (
			SELECT ?
			UNION ALL
			SELECT f.id FROM folders f JOIN folder_tree ft ON f.parent_id = ft.id
		)
		SELECT c.thumbnail, c.path FROM chapters c WHERE c.folder_id IN folder_tree ORDER BY c.created_at ASC;
	`
	rows, err := s.db.Query(query, folderID)
	if err != nil {
		log.Printf("Error finding chapters for folder thumbnail %d: %v", folderID, err)
		return
	}
	defer rows.Close()

	var chapters []struct {
		Thumbnail sql.NullString
		Path      string
	}
	for rows.Next() {
		var c struct {
			Thumbnail sql.NullString
			Path      string
		}
		if err := rows.Scan(&c.Thumbnail, &c.Path); err != nil {
			continue
		}
		chapters = append(chapters, c)
	}

	if len(chapters) > 0 {
		// Sort naturally to find the true first chapter
		sort.Slice(chapters, func(i, j int) bool {
			return util.NaturalSortLess(chapters[i].Path, chapters[j].Path)
		})
		firstChapter := chapters[0]
		if firstChapter.Thumbnail.Valid {
			_, err := s.db.Exec("UPDATE folders SET thumbnail = ? WHERE id = ?", firstChapter.Thumbnail.String, folderID)
			if err != nil {
				log.Printf("Error updating folder thumbnail %d: %v", folderID, err)
			}
		}
	}
}

// ListItemsOptions provides flexible filtering for listing folders and chapters.
type ListItemsOptions struct {
	UserID   int64  `json:"user_id"`
	ParentID *int64 `json:"parent_id,omitempty"` // Filter by parent folder
	TagID    *int64 `json:"tag_id,omitempty"`    // Filter by tag
	Search   string `json:"search,omitempty"`
	SortBy   string `json:"sort_by,omitempty"`
	SortDir  string `json:"sort_dir,omitempty"`
	Page     int    `json:"page"`
	PerPage  int    `json:"per_page"`
}

// ListItems is the new generic function for fetching folders and chapters.
func (s *Store) ListItems(opts ListItemsOptions) (*models.Folder, []*models.Folder, []*models.Chapter, int, error) {
	var currentFolder *models.Folder
	if opts.ParentID != nil && *opts.ParentID != 0 {
		f, err := s.GetFolder(*opts.ParentID)
		if err != nil {
			return nil, nil, nil, 0, err
		}
		currentFolder = f
	}
	// --- Build dynamic query parts ---
	var folderWhere, chapterWhere, tagJoin string
	var folderArgs, chapterArgs []interface{}

	// Filter by parent folder
	if opts.TagID == nil && *opts.ParentID == 0 { // A special case for root
		folderWhere = "f.parent_id IS NULL"
		chapterWhere = "1=0" // No chapters at the root level
		// chapterWhere = "c.folder_id IS NULL"
	} else if opts.TagID != nil {
		// Tag filtering - show folders with the specified tag
		tagJoin = "JOIN folder_tags ft ON f.id = ft.folder_id"
		folderWhere = "ft.tag_id = ?"
		folderArgs = append(folderArgs, *opts.TagID)
		chapterWhere = "1=0" // No chapters when filtering by tag
	} else {
		// Regular parent folder filtering
		folderWhere = "f.parent_id = ?"
		folderArgs = append(folderArgs, *opts.ParentID)
		chapterWhere = "c.folder_id = ?"
		chapterArgs = append(chapterArgs, *opts.ParentID)
	}

	if opts.Search != "" {
		folderWhere += " AND f.name LIKE ?"
		folderArgs = append(folderArgs, "%"+opts.Search+"%")
		chapterWhere += " AND c.path LIKE ?"
		chapterArgs = append(chapterArgs, "%"+filepath.Base(opts.Search)+"%")
	}

	// Count total items
	var totalItems int
	countQuery := fmt.Sprintf("SELECT (SELECT COUNT(*) FROM folders f %s WHERE %s) + (SELECT COUNT(*) FROM chapters c WHERE %s);", tagJoin, folderWhere, chapterWhere)
	s.db.QueryRow(countQuery, append(folderArgs, chapterArgs...)...).Scan(&totalItems)

	baseQuery := `
		-- Select Folders
		SELECT
			1 as item_type, -- 1 for folder
			f.id,
			f.path,
			f.name,
			f.thumbnail,
			NULL as chapter_page_count,
			NULL as chapter_created_at,
			NULL as chapter_updated_at,
			NULL as user_read,
			NULL as user_progress,
			f.created_at as sort_created_at,
			f.name as sort_name
		FROM folders f %s WHERE %s
		UNION ALL
		-- Select Chapters
		SELECT
			2 as item_type, -- 2 for chapter
			c.id,
			c.path,
			c.path as name, -- Use path for sorting/display name
			c.thumbnail,
			c.page_count,
			c.created_at,
			c.updated_at,
			COALESCE(ucp.read, 0),
			COALESCE(ucp.progress_percent, 0),
			c.created_at as sort_created_at,
			c.path as sort_name
		FROM chapters c
		LEFT JOIN user_chapter_progress ucp ON c.id = ucp.chapter_id AND ucp.user_id = ?
		WHERE %s
	`

	// Build final query with sorting and pagination
	finalQuery := fmt.Sprintf(baseQuery, tagJoin, folderWhere, chapterWhere)
	sortBy := opts.SortBy
	if sortBy == "" {
		sortBy = "auto" // Default to natural sorting
	}
	sortDir := "ASC"
	if opts.SortDir == "desc" {
		sortDir = "DESC"
	}
	sortClause := "ORDER BY item_type ASC, sort_name %s"
	switch sortBy {
	case "created_at":
		sortClause = "ORDER BY item_type ASC, sort_created_at %s"
	case "updated_at":
		sortClause = "ORDER BY item_type ASC, chapter_updated_at %s"
	case "progress":
		sortClause = "ORDER BY item_type ASC, user_progress %s, sort_name ASC"
	default:
		sortClause = "ORDER BY item_type ASC, sort_name %s"
	}
	finalQuery += " " + fmt.Sprintf(sortClause, sortDir)

	allArgs := append(folderArgs, opts.UserID)
	allArgs = append(allArgs, chapterArgs...)
	if opts.SortBy != "" && opts.SortBy != "auto" {
		finalQuery += " LIMIT ? OFFSET ?"
		offset := (opts.Page - 1) * opts.PerPage
		allArgs = append(allArgs, opts.PerPage, offset)
	}

	rows, err := s.db.Query(finalQuery, allArgs...)
	if err != nil {
		return currentFolder, nil, nil, 0, err
	}
	defer rows.Close()

	var subfolders []*models.Folder
	var chapters []*models.Chapter

	for rows.Next() {
		// Scan results and append to either subfolders or chapters slice based on item_type
		var itemType int
		var folder models.Folder
		var chapter models.Chapter
		var folderThumb, chapPath, sortName sql.NullString
		var pageCount, userProgress sql.NullInt64
		var userRead sql.NullBool
		var createdAtStr, updatedAtStr sql.NullString
		var createdAt, updatedAt sql.NullTime
		var sortDate sql.NullTime

		if err := rows.Scan(
			&itemType, &chapter.ID, &chapPath, &folder.Name, &folderThumb,
			&pageCount,
			&createdAtStr, &updatedAtStr, &userRead, &userProgress, &sortDate, &sortName); err != nil {
			return currentFolder, nil, nil, 0, err
		}
		if createdAtStr.Valid {
			createdAt.Time, _ = time.Parse("2006-01-02 15:04:05", createdAtStr.String)
			createdAt.Valid = true
		}
		if updatedAtStr.Valid {
			updatedAt.Time, _ = time.Parse("2006-01-02 15:04:05", updatedAtStr.String)
			updatedAt.Valid = true
		}
		// create a map of folder id and struct of total chapters and read chapters
		if itemType == 1 { // Folder
			folder.ID = chapter.ID
			folder.Path = chapPath.String
			folder.Thumbnail = folderThumb.String
			subfolders = append(subfolders, &folder)
		} else { // Chapter
			chapter.FolderID = *opts.ParentID
			chapter.Path = chapPath.String
			chapter.Thumbnail = folderThumb.String
			chapter.PageCount = int(pageCount.Int64)
			chapter.Read = userRead.Bool
			chapter.ProgressPercent = int(userProgress.Int64)
			if createdAt.Valid {
				chapter.CreatedAt = createdAt.Time
			}
			if updatedAt.Valid {
				chapter.UpdatedAt = updatedAt.Time
			}
			chapters = append(chapters, &chapter)
		}
	}

	// add total chapters and read chapters to subfolders
	for _, folder := range subfolders {
		totalChapters, readChapters, err := s.GetFolderStats(folder.ID, opts.UserID)
		if err != nil {
			return currentFolder, nil, nil, 0, err
		}
		folder.TotalChapters = totalChapters
		folder.ReadChapters = readChapters
		folder.Settings = &models.FolderSettings{
			SortBy:  opts.SortBy,
			SortDir: opts.SortDir,
		}
	}
	// Sort subfolders naturally if requested
	switch sortBy {
	case "auto":
		folderTitles := make([]string, len(subfolders))
		for i, folder := range subfolders {
			folderTitles[i] = folder.Name
		}
		fs := util.NewChapterSorter(folderTitles)
		slices.SortFunc(subfolders, func(a, b *models.Folder) int {
			comparison := fs.Compare(a.Name, b.Name)
			if strings.ToLower(sortDir) == "desc" {
				return -comparison
			}
			return comparison
		})
		subfolders = limitAndOffsetFolders(subfolders, opts.Page, opts.PerPage)

		// Sort chapters naturally
		chapterTitles := make([]string, len(chapters))
		for i, chapter := range chapters {
			chapterTitles[i] = GetChapterTitle(chapter)
		}
		cs := util.NewChapterSorter(chapterTitles)
		slices.SortFunc(chapters, func(a, b *models.Chapter) int {
			comparison := cs.Compare(GetChapterTitle(a), GetChapterTitle(b))
			if strings.ToLower(sortDir) == "desc" {
				return -comparison
			}
			return comparison
		})
		chapters = limitAndOffsetChapters(chapters, opts.Page, opts.PerPage)
	case "progress":
		sort.Slice(chapters, func(i, j int) bool {
			return chapters[i].ProgressPercent < chapters[j].ProgressPercent
		})
		// Sort subfolders by progress
		sort.Slice(subfolders, func(i, j int) bool {
			iChapters := subfolders[i].Chapters
			iReadChapters := 0
			iTotalChapters := len(iChapters)
			for _, c := range iChapters {
				if c.Read {
					iReadChapters++
				}
			}
			jChapters := subfolders[j].Chapters
			jReadChapters := 0
			jTotalChapters := len(jChapters)
			for _, c := range jChapters {
				if c.Read {
					jReadChapters++
				}
			}
			if iTotalChapters == 0 && jTotalChapters == 0 {
				return false // Both have no chapters, keep original order
			}
			if iTotalChapters == 0 {
				return false // i has no chapters, j comes first
			}
			if jTotalChapters == 0 {
				return true // j has no chapters, i comes first
			}
			iProgress := float64(iReadChapters) / float64(iTotalChapters)
			jProgress := float64(jReadChapters) / float64(jTotalChapters)
			return iProgress < jProgress
		})
	case "name":
		sort.Slice(subfolders, func(i, j int) bool {
			return util.NaturalSortLess(subfolders[i].Name, subfolders[j].Name)
		})
		sort.Slice(chapters, func(i, j int) bool {
			return util.NaturalSortLess(chapters[i].Path, chapters[j].Path)
		})
	}
	// totalItems := totalFolders + 0 // + totalChapters
	return currentFolder, subfolders, chapters, totalItems, nil
}

// limitAndOffsetChapters is a helper function to remove excess elements from the slice and return it
func limitAndOffsetChapters(slice []*models.Chapter, page, perPage int) []*models.Chapter {
	if slice == nil {
		return nil
	}
	offset := (page - 1) * perPage
	var newslice []*models.Chapter
	if offset < len(slice) {
		end := offset + perPage
		if end > len(slice) {
			end = len(slice)
		}
		newslice = slice[offset:end]
	} else {
		// If offset is beyond the length of slice, return an empty slice
		newslice = []*models.Chapter{}
	}
	return newslice
}

// limitAndOffsetFolders is a helper function to remove excess elements from the slice and return it
func limitAndOffsetFolders(slice []*models.Folder, page, perPage int) []*models.Folder {
	if slice == nil {
		return nil
	}
	offset := (page - 1) * perPage
	var newslice []*models.Folder
	if offset < len(slice) {
		end := offset + perPage
		if end > len(slice) {
			end = len(slice)
		}
		newslice = slice[offset:end]
	} else {
		// If offset is beyond the length of slice, return an empty slice
		newslice = []*models.Folder{}
	}
	return newslice
}

// GetFolderPath retrieves the entire ancestry of a folder for breadcrumbs.
func (s *Store) GetFolderPath(folderID int64) ([]*models.Folder, error) {
	var path []*models.Folder
	currentID := &folderID
	for currentID != nil && *currentID > 0 {
		folder, err := s.GetFolder(*currentID)
		if err != nil {
			return nil, err
		}
		path = append([]*models.Folder{folder}, path...) // Prepend to reverse the order
		currentID = folder.ParentID
	}
	return path, nil
}

// GetFolderSettings retrieves the sort settings for a folder.
func (s *Store) GetFolderSettings(folderID int64, userID int64) (*models.FolderSettings, error) {
	var settings models.FolderSettings
	err := s.db.QueryRow(`
		SELECT sort_by, sort_dir
		FROM user_folder_settings
		WHERE folder_id = ? AND user_id = ?
	`, folderID, userID).Scan(&settings.SortBy, &settings.SortDir)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return default settings if not found
			settings.SortBy = "auto"
			settings.SortDir = "asc"
			return &settings, nil
		}
		return nil, err
	}
	return &settings, nil
}

// UpdateFolderSettings saves the sort settings for a Folder.
func (s *Store) UpdateFolderSettings(folderID int64, userID int64, sortBy, sortDir string) error {
	query := `INSERT INTO user_folder_settings (folder_id, user_id, sort_by, sort_dir) VALUES (?, ?, ?, ?)
              ON CONFLICT(user_id, folder_id) DO UPDATE SET sort_by=excluded.sort_by, sort_dir=excluded.sort_dir;`
	_, err := s.db.Exec(query, folderID, userID, sortBy, sortDir)
	return err
}

// MarkAllChaptersAs updates the 'read' status for all chapters of a folder.
func (s *Store) MarkFolderChaptersAs(folderID int64, read bool, userID int64) error {
	// get chapter ids from the folder
	query := `
		SELECT c.id
		FROM chapters c
		WHERE c.folder_id = ?
	`
	rows, err := s.db.Query(query, folderID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var chapterIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		chapterIDs = append(chapterIDs, id)
	}

	// Update user_chapter_progress for each chapter individually
	query = `
		INSERT INTO user_chapter_progress (user_id, chapter_id, progress_percent, read, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, chapter_id) DO UPDATE SET
			progress_percent = excluded.progress_percent,
			read = excluded.read,
			updated_at = CURRENT_TIMESTAMP;
	`
	stmt, err := s.db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	progressPercent := map[bool]int{true: 100, false: 0}[read] // Set progress to 100 if read, 0 if unread

	for _, chapterID := range chapterIDs {
		_, err := stmt.Exec(userID, chapterID, progressPercent, read)
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateFolderThumbnail updates the thumbnail for a single folder.
func (s *Store) UpdateFolderThumbnail(folderID int64, thumbnail string) error {
	query := "UPDATE folders SET thumbnail = ? WHERE id = ?"
	result, err := s.db.Exec(query, thumbnail, folderID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrFolderNotFound
	}
	return nil
}
