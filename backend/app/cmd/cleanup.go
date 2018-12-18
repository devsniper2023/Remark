package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/umputun/remark/backend/app/store"
)

// CleanupCommand set of flags and command for cleanup
type CleanupCommand struct {
	Site     string   `short:"s" long:"site" env:"SITE" default:"remark" description:"site name"`
	Dry      bool     `long:"dry" env:"DRY" description:"dry mode, will not remove comments"`
	From     string   `long:"from" description:"from yyyymmdd"`
	To       string   `long:"to" description:"from yyyymmdd"`
	BadWords []string `short:"w" long:"bword" description:"bad word(s)"`
	BadUsers []string `short:"u" long:"buser" description:"bad user(s)"`
	CommonOpts
}

// Execute runs cleanup with CleanupCommand parameters, entry point for "cleanup" command
func (cc *CleanupCommand) Execute(args []string) error {
	log.Printf("[INFO] cleanup for site %s", cc.Site)

	posts, err := cc.postsInRange(cc.From, cc.To)
	if err != nil {
		return errors.Wrap(err, "can't get posts")
	}

	log.Printf("[DEBUG] got %d posts", len(posts))

	totalComments, spamComments := 0, 0
	for _, post := range posts {
		comments, err := cc.listComments(post.URL)
		if err != nil {
			continue
		}
		for _, comment := range comments {
			totalComments++
			spam, score := cc.isSpam(comment)
			if spam {
				spamComments++
				log.Printf("[SPAM] %+v [%.0f]", comment, score)
				if !cc.Dry {
					if err = cc.deleteComment(comment); err != nil {
						log.Printf("[WARN] can't remove comment, %v", err)
					}
				}
			}
		}
	}
	log.Printf("[INFO] comments=%d, spam=%d", totalComments, spamComments)
	return err
}

// get list of posts in from/to represented as yyyymmdd
func (cc *CleanupCommand) postsInRange(fromS, toS string) ([]store.PostInfo, error) {
	posts, err := cc.listPosts()
	if err != nil {
		return nil, errors.Wrapf(err, "can't list posts for %s", cc.Site)
	}

	from := time.Date(1970, 1, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(2999, 1, 1, 0, 0, 0, 0, time.Local)

	if fromS != "" {
		from, err = time.ParseInLocation("20060102", fromS, time.Local)
		if err != nil {
			return nil, errors.Wrap(err, "can't parse --from")
		}
	}

	if toS != "" {
		to, err = time.ParseInLocation("20060102", toS, time.Local)
		if err != nil {
			return nil, errors.Wrap(err, "can't parse --to")
		}
	}

	var filteredList []store.PostInfo
	for _, postInfo := range posts {
		if postInfo.FirstTS.After(from) && postInfo.LastTS.Before(to) {
			filteredList = append(filteredList, postInfo)
		}
	}
	return filteredList, nil
}

// get all posts via GET /list?site=siteID&limit=50&skip=10
func (cc *CleanupCommand) listPosts() ([]store.PostInfo, error) {
	listURL := fmt.Sprintf("%s/api/v1/list?site=%s&limit=10000", cc.RemarkURL, cc.Site)
	r, err := http.Get(listURL)
	if err != nil {
		return nil, errors.Wrapf(err, "get request failed for list of posts, site %s", cc.Site)
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		return nil, errors.Errorf("request %s failed with status %d", listURL, r.StatusCode)
	}

	list := []store.PostInfo{}
	if err = json.NewDecoder(r.Body).Decode(&list); err != nil {
		return nil, errors.Wrapf(err, "can't decode list of posts for site %s", cc.Site)
	}
	return list, nil
}

// get all comments for post url via /find?site=siteID&url=post-url&format=[tree|plain]
func (cc *CleanupCommand) listComments(postURL string) ([]store.Comment, error) {

	commentsURL := fmt.Sprintf("%s/api/v1/find?site=%s&url=%s&format=plain", cc.RemarkURL, cc.Site, postURL)

	var r *http.Response
	var err error

	for {
		r, err = http.Get(commentsURL)
		if err != nil {
			return nil, errors.Wrapf(err, "get request failed for comments, %s", postURL)
		}
		if r.StatusCode == 429 {
			r.Body.Close()
			time.Sleep(500 * time.Millisecond)
			continue
		}
		break
	}

	if r.StatusCode != 200 {
		return nil, errors.Errorf("request %s failed with status %d", commentsURL, r.StatusCode)
	}

	commentsWithInfo := struct {
		Comments []store.Comment `json:"comments"`
		Info     store.PostInfo  `json:"info,omitempty"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&commentsWithInfo); err != nil {
		return nil, errors.Wrapf(err, "can't decode list of comments for %s", postURL)
	}
	return commentsWithInfo.Comments, nil
}

// deleteComment with DELETE /admin/comment/{id}?site=siteID&url=post-url
func (cc *CleanupCommand) deleteComment(c store.Comment) error {

	deleteURL := fmt.Sprintf("%s/api/v1/admin/comment/%s?site=%s&url=%s&format=plain&secret=%s",
		cc.RemarkURL, c.ID, cc.Site, c.Locator.URL, cc.SharedSecret)
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to make delete request for comment %s, %s", c.ID, c.Locator.URL)
	}
	client := http.Client{}
	r, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "delete request failed for comment %s, %s", c.ID, c.Locator.URL)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return errors.Errorf("delete request failed with status %s", r.Status)
	}
	return nil
}

// isSpam calculates spam's probability as a score
func (cc *CleanupCommand) isSpam(comment store.Comment) (bool, float64) {

	badWord := func(txt string) float64 {
		res := 0.0
		for _, w := range cc.BadWords {
			if strings.Contains(txt, w) {
				res += 0.20
			}
			if res > 1 {
				return 1
			}
		}
		return res
	}

	hasBadUser := func(txt string) bool {
		for _, w := range cc.BadUsers {
			if strings.Contains(txt, w) {
				return true
			}
		}
		return false
	}

	score := 0.0

	// don't mark deleted as spam
	if comment.Deleted {
		return false, 0
	}

	score += 50 * badWord(comment.Text) // up to 50, 5 bad words will reach max

	if hasBadUser(comment.User.ID) { // predefined list of bad user substrings
		score += 10
	}

	if comment.Score == 0 { // most of spam comments with 0 score
		score += 10
	}

	// any link inside
	if strings.Contains(comment.Text, "http:") || strings.Contains(comment.Text, "https:") {
		score += 20
	}

	// 5 or more links
	if strings.Count(comment.Text, "href") >= 5 {
		score += 10
	}

	// any score probably not for spam
	if comment.Score != 0 {
		score -= 30
	}

	return score > 50, score
}
