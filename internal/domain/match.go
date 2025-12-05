package domain

import "time"

type Match struct {
	ID          int       `json:"id" db:"id"`
	User1ID     int       `json:"user1_id" db:"user1_id"`
	User2ID     int       `json:"user2_id" db:"user2_id"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	Explanation *string   `json:"explanation" db:"match_explanation"`
	Icebreakers []string  `json:"icebreakers" db:"icebreakers"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

func (m *Match) HasUser(userID int) bool {
	return m.User1ID == userID || m.User2ID == userID
}

func (m *Match) GetOtherUserID(userID int) (int, bool) {
	if m.User1ID == userID {
		return m.User2ID, true
	}
	if m.User2ID == userID {
		return m.User1ID, true
	}
	return 0, false
}
