package engine

import (
	"os"
	"testing"
	"time"

	"github.com/coreos/bbolt"
	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark/app/store"
)

var testDb = "/tmp/test-remark.db"

func TestBoltDB_CreateAndFind(t *testing.T) {
	defer os.Remove(testDb)
	var b = prep(t)

	res, err := b.Find(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, `some text, <a href="http://radio-t.com">link</a>`, res[0].Text)
	assert.Equal(t, "user1", res[0].User.ID)
	t.Log(res[0].ID)

	_, err = b.Create(store.Comment{ID: res[0].ID, Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}})
	assert.NotNil(t, err)
	assert.Equal(t, "key id-1 already in store", err.Error())
}

func TestBoltDB_Delete(t *testing.T) {
	b := prep(t)
	defer os.Remove(testDb)

	loc := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	res, err := b.Find(loc, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res), "initially 2 comments")

	err = b.Delete(loc, res[0].ID)
	assert.Nil(t, err)

	res, err = b.Find(loc, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, "", res[0].Text)
	assert.True(t, res[0].Deleted, "marked deleted")
	assert.Equal(t, "some text2", res[1].Text)
	assert.False(t, res[1].Deleted)

	comments, err := b.Last("radio-t", 10)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(comments), "1 in last, 1 removed")
}

func TestBoltDB_DeleteAll(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	loc := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	res, err := b.Find(loc, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res), "initially 2 comments")

	err = b.DeleteAll("radio-t")
	assert.Nil(t, err)

	comments, err := b.Last("radio-t", 10)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(comments), "nothing left")

	c, err := b.Count(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"})
	assert.Nil(t, err)
	assert.Equal(t, 0, c, "0 count")
}

func TestBoltDB_Get(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	res, err := b.Find(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))

	comment, err := b.Get(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[1].ID)
	assert.Nil(t, err)
	assert.Equal(t, "some text2", comment.Text)

	comment, err = b.Get(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "1234567")
	assert.NotNil(t, err)
}

func TestBoltDB_Put(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)
	loc := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	res, err := b.Find(loc, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))

	comment := res[0]
	comment.Text = "abc 123"
	comment.Score = 100
	err = b.Put(loc, comment)
	assert.Nil(t, err)

	comment, err = b.Get(loc, res[0].ID)
	assert.Nil(t, err)
	assert.Equal(t, "abc 123", comment.Text)
	assert.Equal(t, res[0].ID, comment.ID)
	assert.Equal(t, 100, comment.Score)
}

func TestBoltDB_Last(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	res, err := b.Last("radio-t", 0)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, "some text2", res[0].Text)

	res, err = b.Last("radio-t", 1)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "some text2", res[0].Text)
}

func TestBoltDB_Count(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	c, err := b.Count(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"})
	assert.Nil(t, err)
	assert.Equal(t, 2, c)
}

func TestBoltDB_BlockUser(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	assert.False(t, b.IsBlocked("radio-t", "user1"), "nothing blocked")

	assert.NoError(t, b.SetBlock("radio-t", "user1", true))
	assert.True(t, b.IsBlocked("radio-t", "user1"), "user1 blocked")

	assert.False(t, b.IsBlocked("radio-t", "user2"), "user2 still unblocked")

	assert.NoError(t, b.SetBlock("radio-t", "user1", false))
	assert.False(t, b.IsBlocked("radio-t", "user1"), "user1 unblocked")
}

func TestBoltDB_BlockList(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	assert.NoError(t, b.SetBlock("radio-t", "user1", true))
	assert.NoError(t, b.SetBlock("radio-t", "user2", true))
	assert.NoError(t, b.SetBlock("radio-t", "user3", false))

	ids, err := b.Blocked("radio-t")
	assert.NoError(t, err)

	assert.Equal(t, 2, len(ids))
	assert.Equal(t, "user1", ids[0].ID)
	assert.Equal(t, "user2", ids[1].ID)
	t.Logf("%+v", ids)
}

func TestBoltDB_List(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t) // two comments for https://radio-t.com

	// add one more for https://radio-t.com/2
	comment := store.Comment{
		ID:        "12345",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err := b.Create(comment)
	assert.Nil(t, err)

	res, err := b.List("radio-t", 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1}, {URL: "https://radio-t.com", Count: 2}}, res)

	res, err = b.List("radio-t", 1, 0)
	assert.Nil(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1}}, res)

	res, err = b.List("radio-t", 1, 1)
	assert.Nil(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com", Count: 2}}, res)
}

func TestBoltDB_GetForUser(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	res, count, err := b.User("radio-t", "user1", 5)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, 2, count)
	assert.Equal(t, "some text2", res[0].Text, "sorted by -time")

	res, count, err = b.User("radio-t", "user1", 1)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(res), "allow 1 comment")
	assert.Equal(t, 2, count)
	assert.Equal(t, "some text2", res[0].Text, "sorted by -time")

}

func TestBoltDB_Ref(t *testing.T) {
	b := BoltDB{}
	comment := store.Comment{
		ID:        "12345",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	res := b.makeRef(comment)
	assert.Equal(t, "https://radio-t.com/2!!12345", string(res))

	url, id, err := b.parseRef([]byte("https://radio-t.com/2!!12345"))
	assert.Nil(t, err)
	assert.Equal(t, "https://radio-t.com/2", url)
	assert.Equal(t, "12345", id)

	_, _, err = b.parseRef([]byte("https://radio-t.com/2"))
	assert.NotNil(t, err)
}

// makes new boltdb, put two records
func prep(t *testing.T) *BoltDB {
	os.Remove(testDb)

	boltStore, err := NewBoltDB(bolt.Options{}, BoltSite{FileName: testDb, SiteID: "radio-t"})
	assert.Nil(t, err)
	b := boltStore

	comment := store.Comment{
		ID:        "id-1",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err = b.Create(comment)
	assert.Nil(t, err)

	comment = store.Comment{
		ID:        "id-2",
		Text:      "some text2",
		Timestamp: time.Date(2017, 12, 20, 15, 18, 23, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err = b.Create(comment)
	assert.Nil(t, err)

	return b
}
