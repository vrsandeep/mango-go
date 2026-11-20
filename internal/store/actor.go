package store

// Actor is the signed-in user for a queue or subscription change.
// Admin is unfiltered so an operator can see every row. The download worker
// uses an admin actor. A normal user only matches rows they created.
type Actor struct {
	UserID int64
	Admin  bool
}

// owns returns a SQL fragment and args that limit a query to this actor.
// column must be a fixed column name from the caller, never request input.
func (a Actor) owns(column string) (string, []any) {
	if a.Admin {
		return "", nil
	}
	return " AND " + column + " = ?", []any{a.UserID}
}
