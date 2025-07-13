package store

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/util"
)

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
	query := "SELECT id, path, name, parent_id, thumbnail, created_at, updated_at FROM folders WHERE id = ?"
	err := s.db.QueryRow(query, id).Scan(&folder.ID, &folder.Path, &folder.Name, &parentID, &thumbnail, &folder.CreatedAt, &folder.UpdatedAt)
	if err != nil {
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
			s.db.Exec("UPDATE folders SET thumbnail = ? WHERE id = ?", firstChapter.Thumbnail.String, folderID)
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
	if opts.ParentID != nil {
		f, err := s.GetFolder(*opts.ParentID)
		if err != nil {
			return nil, nil, nil, 0, err
		}
		currentFolder = f
	}
	// --- Build WHERE clauses and arguments dynamically based on options ---
	var folderWhere, chapterWhere, tagJoin string
	var folderArgs, chapterArgs []interface{}

	// Filter by parent folder
	if *opts.ParentID == 0 { // A special case for root
		folderWhere = "f.parent_id IS NULL"
		chapterWhere = "c.folder_id IS NULL"
	} else {
		folderWhere = "f.parent_id = ?"
		folderArgs = append(folderArgs, *opts.ParentID)
		chapterWhere = "c.folder_id = ?"
		chapterArgs = append(chapterArgs, *opts.ParentID)
	}

	// Filter by tag
	if opts.TagID != nil {
		tagJoin = "JOIN folder_tags ft ON f.id = ft.folder_id"
		if folderWhere != "" {
			folderWhere += " AND"
		}
		folderWhere += " ft.tag_id = ?"
		folderArgs = append(folderArgs, *opts.TagID)
		// Tags apply to folders, which act as series containers, so we don't filter chapters by tag directly.
		// Instead, we show all folders with that tag.
		chapterWhere = "1=0" // This effectively returns no chapters at the tag level.
	}

	if opts.Search != "" {
		if folderWhere != "" {
			folderWhere += " AND"
		}
		folderWhere += " f.name LIKE ?"
		folderArgs = append(folderArgs, "%"+opts.Search+"%")

		if chapterWhere != "" {
			chapterWhere += " AND"
		}
		chapterWhere += " c.path LIKE ?"
		chapterArgs = append(chapterArgs, "%"+filepath.Base(opts.Search)+"%")
	}

	// Default to an impossible condition if no filter is set, to avoid returning all items.
	if folderWhere == "" {
		folderWhere = "1=0"
	}
	if chapterWhere == "" {
		chapterWhere = "1=0"
	}

	// --- Use the UNION ALL query from the previous step, now with dynamic WHERE clauses ---
	baseQuery := fmt.Sprintf(`
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
	`, tagJoin, folderWhere, chapterWhere)

	// Count total items
	var totalItems int
	countQuery := fmt.Sprintf("SELECT (SELECT COUNT(*) FROM folders f WHERE %s) + (SELECT COUNT(*) FROM chapters c WHERE %s);", folderWhere, chapterWhere)
	s.db.QueryRow(countQuery, append(folderArgs, chapterArgs...)...).Scan(&totalItems)

	// ... (The rest of the complex UNION ALL query, sorting, and pagination logic follows) ...

	// For brevity, the full query implementation is represented by this placeholder.
	// The key change is the dynamic construction of the WHERE clauses above.
	// Build final query with sorting and pagination
	// finalQuery := fmt.Sprintf(baseQuery, folderWhere, chapterWhere)
	finalQuery := baseQuery
	sortBy := opts.SortBy
	if sortBy == "" {
		sortBy = "auto" // Default to natural sorting
	}
	sortDir := "ASC"
	if opts.SortDir == "desc" {
		sortDir = "DESC"
	}
	sortClause := "ORDER BY item_type ASC, sort_name %s"
	if sortBy == "created_at" {
		sortClause = "ORDER BY item_type ASC, sort_created_at %s"
	}
	finalQuery += " " + fmt.Sprintf(sortClause, sortDir)
	finalQuery += " LIMIT ? OFFSET ?"

	allArgs := append(folderArgs, opts.UserID)
	allArgs = append(allArgs, chapterArgs...)
	offset := (opts.Page - 1) * opts.PerPage
	allArgs = append(allArgs, opts.PerPage, offset)

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
		var userRead sql.NullBool
		var userProgress sql.NullInt64
		if err := rows.Scan(&itemType, &folder.ID, &folder.Path, &folder.Name, &folder.Thumbnail,
			&chapter.PageCount, &chapter.CreatedAt, &chapter.UpdatedAt,
			&userRead, &userProgress, &folder.CreatedAt, &folder.Name); err != nil {
			return currentFolder, nil, nil, 0, err
		}
		switch itemType {
		case 1:
			subfolders = append(subfolders, &folder)
		case 2:
			chapters = append(chapters, &chapter)
		}
	}
	// Sort subfolders naturally if requested
	switch sortBy {
	case "auto":
		sort.Slice(subfolders, func(i, j int) bool {
			return util.NaturalSortLess(subfolders[i].Name, subfolders[j].Name)
		})
		sort.Slice(chapters, func(i, j int) bool {
			return util.NaturalSortLess(chapters[i].Path, chapters[j].Path)
		})
	case "title":
		sort.Slice(subfolders, func(i, j int) bool {
			return util.NaturalSortLess(subfolders[i].Name, subfolders[j].Name)
		})
		sort.Slice(chapters, func(i, j int) bool {
			return util.NaturalSortLess(chapters[i].Path, chapters[j].Path)
		})
	case "updated_at":
		sort.Slice(subfolders, func(i, j int) bool {
			return subfolders[i].UpdatedAt.Before(subfolders[j].UpdatedAt)
		})
		sort.Slice(chapters, func(i, j int) bool {
			return chapters[i].UpdatedAt.Before(chapters[j].UpdatedAt)
		})
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

// AddTagToFolder creates the association between a folder and a tag.
func (s *Store) AddTagToFolder(folderID int64, tagName string) (*models.Tag, error) {
	tagName = strings.TrimSpace(strings.ToLower(tagName))
	if tagName == "" {
		return nil, fmt.Errorf("tag name cannot be empty")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	tag, err := s.GetOrCreateTag(tagName)
	if err != nil {
		return nil, err
	}
	_, err = s.db.Exec("INSERT OR IGNORE INTO folder_tags (folder_id, tag_id) VALUES (?, ?)", folderID, tag.ID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			// If the tag is already associated with the folder, ignore the error
			return &models.Tag{ID: tag.ID, Name: tag.Name}, nil
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return tag, nil
}

// RemoveTagFromFolder removes the association between a folder and a tag.
func (s *Store) RemoveTagFromFolder(folderID, tagID int64) error {
	_, err := s.db.Exec("DELETE FROM folder_tags WHERE folder_id = ? AND tag_id = ?", folderID, tagID)
	return err
}
