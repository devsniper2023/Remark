package migrator

import (
	"encoding/xml"
	"io"
	"log"
	"time"

	"strings"

	"github.com/pkg/errors"
	"github.com/umputun/remark/app/store"
)

// Disqus implements Importer from disqus xml
type Disqus struct {
	DataStore store.Interface
}

type disqusThread struct {
	UID         string    `xml:"id,attr"`
	Forum       string    `xml:"forum"`
	Link        string    `xml:"link"`
	Title       string    `xml:"title"`
	Message     string    `xml:"message"`
	CreateAt    time.Time `xml:"createdAt"`
	AuthorName  string    `xml:"author>name"`
	AuthorEmail string    `xml:"author>email"`
	Anonymous   bool      `xml:"author>isAnonymous"`
	IP          string    `xml:"ipAddress"`
	Closed      bool      `xml:"isClosed"`
	Deleted     bool      `xml:"isDeleted"`
}

type disqusComment struct {
	UID            string    `xml:"id,attr"`
	ID             string    `xml:"id"`
	Message        string    `xml:"message"`
	CreatedAt      time.Time `xml:"createdAt"`
	IsSpam         bool      `xml:"isSpam"`
	AuthorEmail    string    `xml:"author>email"`
	AuthorName     string    `xml:"author>name"`
	AuthorUserName string    `xml:"author>username"`
	IP             string    `xml:"ipAddress"`
	Tid            uid       `xml:"thread"`
	Pid            uid       `xml:"parent"`
}

type uid struct {
	Val string `xml:"id,attr"`
}

// Import from disqus and save to store
func (d *Disqus) Import(r io.Reader, siteID string) (err error) {

	commentsCh := d.convert(r, siteID)
	failed := 0
	for c := range commentsCh {
		if _, err = d.DataStore.Create(c); err != nil {
			failed++
		}
	}

	if failed > 0 {
		return errors.Errorf("failed to save %d comments", failed)
	}

	return nil
}

func (d *Disqus) convert(r io.Reader, siteID string) (ch chan store.Comment) {

	postsMap := map[string]string{} // tid:url
	decoder := xml.NewDecoder(r)
	commentsCh := make(chan store.Comment)

	inpThreads, inpComments := 0, 0
	go func() {
		commentsCount := 0
		for {
			t, err := decoder.Token()
			if t == nil || err != nil {
				break
			}

			switch se := t.(type) {
			case xml.StartElement:
				if se.Name.Local == "thread" {
					inpThreads++
					thread := disqusThread{}
					if err := decoder.DecodeElement(&thread, &se); err == nil {
						postsMap[thread.UID] = thread.Link
					}
					continue
				}
				if se.Name.Local == "post" {
					inpComments++
					comment := disqusComment{}
					if err := decoder.DecodeElement(&comment, &se); err != nil {
						continue
					}
					if comment.IsSpam {
						continue
					}
					c := store.Comment{
						ID:        comment.ID,
						Locator:   store.Locator{URL: postsMap[comment.Tid.Val], SiteID: siteID},
						User:      store.User{ID: comment.AuthorUserName, Name: comment.AuthorName, IP: comment.IP},
						Text:      d.cleanText(comment.Message),
						Timestamp: comment.CreatedAt,
						ParentID:  comment.Pid.Val,
					}
					commentsCh <- c
					commentsCount++
					if commentsCount%1000 == 0 {
						log.Printf("[DEBUG] imported %d comments", commentsCount)
					}
				}

			}
		}
		close(commentsCh)
		log.Printf("[INFO] converted %d posts with %d comments from disqus %d/%d", len(postsMap), commentsCount, inpThreads, inpComments)
	}()

	return commentsCh
}

func (d *Disqus) cleanText(text string) string {
	text = strings.Replace(text, "\n", "", -1)
	text = strings.Replace(text, "\t", "", -1)
	return text
}
