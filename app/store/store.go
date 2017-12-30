package store

//go:generate sh -c "mockery -inpkg -name Interface -print > file.tmp && mv file.tmp store_mock.go"

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"html/template"
	"log"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
)

// Comment represents a single comment with optional reference to its parent
type Comment struct {
	ID        string          `json:"id"`
	ParentID  string          `json:"pid"`
	Text      string          `json:"text"`
	User      User            `json:"user"`
	Locator   Locator         `json:"locator"`
	Score     int             `json:"score"`
	Votes     map[string]bool `json:"votes"`
	Timestamp time.Time       `json:"time"`
	Pin       bool            `json:"pin,omitempty"`
}

// Locator keeps site and url of the post
type Locator struct {
	SiteID string `json:"site,omitempty"`
	URL    string `json:"url"`
}

// User holds user-related info
type User struct {
	Name    string `json:"name"`
	ID      string `json:"id"`
	Picture string `json:"picture"`
	Profile string `json:"profile"`
	Admin   bool   `json:"admin"`
	Blocked bool   `json:"block,omitempty"`
	IP      string `json:"-"`
}

// Request is a container for all finds
type Request struct {
	Locator Locator `json:"locator"`
	Sort    string  `json:"sort"`
	Offset  int     `json:"offset"`
	Limit   int     `json:"limit"`
}

// Interface combines all store interfaces
type Interface interface {
	Accessor
	Admin
}

// Accessor defines all usual access ops avail for regular user
type Accessor interface {
	Create(comment Comment) (commentID string, err error)
	Get(locator Locator, commentID string) (comment Comment, err error)
	Put(locator Locator, comment Comment) error
	Find(request Request) ([]Comment, error)
	Last(locator Locator, max int) ([]Comment, error)
	GetByID(locator Locator, commentID string) (Comment, error)
	GetByUser(locator Locator, userID string) ([]Comment, error)
	Count(locator Locator) (int, error)
	List(locator Locator) ([]string, error)
}

// Admin defines all store ops avail for admin only
type Admin interface {
	Delete(locator Locator, commentID string) error
	SetBlock(locator Locator, userID string, status bool) error
	IsBlocked(locator Locator, userID string) bool
}

func makeCommentID() string {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("[ERROR] can't get randoms, %s", err)
	}
	s := sha1.New()
	if _, err := s.Write(b); err != nil {
		log.Fatalf("[ERROR] can't make sha1 for random, %s", err)
	}
	return fmt.Sprintf("%x", s.Sum(nil))
}

func sanitizeComment(comment Comment) Comment {
	p := bluemonday.UGCPolicy()
	comment.Text = p.Sanitize(comment.Text)
	comment.User.ID = template.HTMLEscapeString(comment.User.ID)
	comment.User.Name = template.HTMLEscapeString(comment.User.Name)
	comment.User.Picture = p.Sanitize(comment.User.Picture)
	comment.User.Profile = p.Sanitize(comment.User.Profile)

	comment.Text = strings.Replace(comment.Text, "\n", "", -1)
	comment.Text = strings.Replace(comment.Text, "\t", "", -1)

	return comment
}
