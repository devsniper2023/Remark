package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"

	"github.com/umputun/remark/app/migrator"
	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/rest/common"
	"github.com/umputun/remark/app/rest/format"
	"github.com/umputun/remark/app/store"
)

// Server is a rest access server
type Server struct {
	Version      string
	DataService  store.Service
	Admins       []string
	AuthGoogle   *auth.Provider
	AuthGithub   *auth.Provider
	AuthFacebook *auth.Provider
	SessionStore *sessions.FilesystemStore
	Exporter     migrator.Exporter
	DevMode      bool

	httpServer *http.Server
	mod        admin
	respCache  *loadingCache
}

// Run the lister and request's router, activate rest server
func (s *Server) Run(port int) {
	log.Print("[INFO] activate rest server")

	// add auth.Developer flag if dev mode is active
	maybeDevMode := func(mode auth.Mode) (modes []auth.Mode) {
		modes = append(modes, mode)
		if s.DevMode {
			modes = append(modes, auth.Developer)
		}
		return modes
	}

	if len(s.Admins) > 0 {
		log.Printf("[DEBUG] admins %+v", s.Admins)
	}

	// cache for responses. Flushes completely on any modification
	s.respCache = newLoadingCache(4*time.Hour, 15*time.Minute)

	router := chi.NewRouter()
	router.Use(middleware.RealIP, Recoverer)
	router.Use(middleware.Throttle(1000), middleware.Timeout(60*time.Second))
	router.Use(Limiter(10), AppInfo("remark42", s.Version), Ping, Logger(LogAll))
	router.Use(auth.Auth(s.SessionStore, s.Admins, maybeDevMode(auth.Anonymous))) // all request by default allow anonymous access

	router.Use(context.ClearHandler) // if you aren't using gorilla/mux, you need to wrap your handlers with context.ClearHandler

	// auth routes for all providers
	router.Route("/auth", func(r chi.Router) {
		r.Mount("/google", s.AuthGoogle.Routes())
		r.Mount("/github", s.AuthGithub.Routes())
		r.Mount("/facebook", s.AuthFacebook.Routes())
		r.Get("/logout", s.AuthGoogle.LogoutHandler) // shortcut, can be any of providers, all logouts do the same
	})

	// api routes
	router.Route("/api/v1", func(rapi chi.Router) {
		rapi.Get("/find", s.findCommentsCtrl)
		rapi.Get("/id/{id}", s.commentByIDCtrl)
		rapi.Get("/comments", s.findUserCommentsCtrl)
		rapi.Get("/last/{max}", s.lastCommentsCtrl)
		rapi.Get("/count", s.countCtrl)
		rapi.Get("/list", s.listCtrl)

		// protected routes, require auth
		rapi.With(auth.Auth(s.SessionStore, s.Admins, maybeDevMode(auth.Full))).Group(func(rauth chi.Router) {
			rauth.Post("/comment", s.createCommentCtrl)
			rauth.Put("/comment/{id}", s.updateCommentCtrl)
			rauth.Get("/user", s.userInfoCtrl)
			rauth.Put("/vote/{id}", s.voteCtrl)

			// admin routes, admin users only
			s.mod = admin{dataService: s.DataService, exporter: s.Exporter, respCache: s.respCache}
			rauth.Mount("/admin", s.mod.routes())
		})
	})

	// add robots and file server for static content from /web
	router.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		render.PlainText(w, r, "User-agent: *\nDisallow: /auth/\nDisallow: /api/\n")
	})
	s.addFileServer(router, "/web", http.Dir(filepath.Join(".", "web")))

	s.httpServer = &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: router}
	err := s.httpServer.ListenAndServe()
	log.Printf("[WARN] http server terminated, %s", err)
}

// POST /comment - adds comment, resets all immutable fields
func (s *Server) createCommentCtrl(w http.ResponseWriter, r *http.Request) {

	comment := store.Comment{}
	if err := render.DecodeJSON(r.Body, &comment); err != nil {
		common.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't bind comment")
		return
	}

	user, err := common.GetUserInfo(r)
	if err != nil { // this not suppose to happen (handled by Auth), just dbl-check
		common.SendErrorJSON(w, r, http.StatusUnauthorized, err, "can't get user info")
		return
	}

	// reset comment to initial state
	func() {
		comment.ID = ""                 // don't allow user to define ID, force auto-gen
		comment.Timestamp = time.Time{} // reset time, force auto-gen
		comment.Votes = make(map[string]bool)
		comment.Score = 0
		comment.Edit = nil
		comment.Pin = false
	}()

	comment.User = user
	comment.User.IP = strings.Split(r.RemoteAddr, ":")[0]

	log.Printf("[DEBUG] create comment %+v", comment)

	// check if user blocked
	if s.mod.checkBlocked(comment.Locator.SiteID, comment.User) {
		common.SendErrorJSON(w, r, http.StatusForbidden, errors.New("rejected"), "user blocked")
		return
	}

	id, err := s.DataService.Create(comment)
	if err != nil {
		common.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't save comment")
		return
	}

	s.respCache.flush() // reset all caches

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, JSON{"id": id, "loc": comment.Locator})
}

// PUT /comment/{id}?site=siteID&url=post-url - update comment
func (s *Server) updateCommentCtrl(w http.ResponseWriter, r *http.Request) {

	edit := struct {
		Text    string
		Summary string
	}{}

	if err := render.DecodeJSON(r.Body, &edit); err != nil {
		common.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't bind comment")
		return
	}

	user, err := common.GetUserInfo(r)
	if err != nil { // this not suppose to happen (handled by Auth), just dbl-check
		common.SendErrorJSON(w, r, http.StatusUnauthorized, err, "can't get user info")
		return
	}
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	id := chi.URLParam(r, "id")
	log.Printf("[DEBUG] update comment %s, %+v", id, edit)

	var currComment store.Comment
	if currComment, err = s.DataService.Get(locator, id); err != nil {
		common.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't find comment")
		return
	}

	if currComment.User.ID != user.ID {
		common.SendErrorJSON(w, r, http.StatusForbidden, errors.New("rejected"), "can not edit comments for other users")
		return
	}

	res, err := s.DataService.EditComment(locator, id, edit.Text, store.Edit{Summary: edit.Summary})
	if err != nil {
		common.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't update comment")
		return
	}

	s.respCache.flush() // reset all caches
	render.JSON(w, r, res)
}

// DELETE /comment/{id}?site=siteID&url=post-url
func (s *Server) deleteCommentCtrl(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	log.Printf("[DEBUG] delete comment %s", id)

	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	err := s.DataService.Delete(locator, id)
	if err != nil {
		common.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't delete comment")
		return
	}

	s.respCache.flush()

	render.Status(r, http.StatusOK)
	render.JSON(w, r, JSON{"id": id, "loc": locator})
}

// GET /find?site=siteID&url=post-url&format=[tree|plain]&sort=[+/-time|+/-score]
// find comments for given post. Returns in tree or plain formats, sorted
func (s *Server) findCommentsCtrl(w http.ResponseWriter, r *http.Request) {
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	log.Printf("[DEBUG] get comments for %+v", locator)

	data, err := s.respCache.get(r.URL.String(), time.Hour, func() ([]byte, error) {
		comments, e := s.DataService.Find(locator, r.URL.Query().Get("sort"))
		if e != nil {
			return nil, e
		}
		comments = s.mod.maskBlockedUsers(comments)
		var b []byte
		switch r.URL.Query().Get("format") {
		case "tree":
			b, e = encodeJSONWithHTML(format.MakeTree(comments, r.URL.Query().Get("sort")))
		default:
			b, e = encodeJSONWithHTML(comments)
		}
		return b, e
	})

	if err != nil {
		common.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't find comments")
		return
	}
	renderJSONFromBytes(w, r, data)
}

// GET /last/{max}?site=siteID - last comments for the siteID, across all posts, sorted by time
func (s *Server) lastCommentsCtrl(w http.ResponseWriter, r *http.Request) {

	log.Printf("[DEBUG] get last comments for %s", r.URL.Query().Get("site"))

	max, err := strconv.Atoi(chi.URLParam(r, "max"))
	if err != nil {
		max = 0
	}

	data, err := s.respCache.get(r.URL.String(), time.Hour, func() ([]byte, error) {
		comments, e := s.DataService.Last(r.URL.Query().Get("site"), max)
		if e != nil {
			return nil, e
		}
		comments = s.mod.maskBlockedUsers(comments)
		return encodeJSONWithHTML(comments)
	})

	if err != nil {
		common.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't get last comments")
		return
	}
	renderJSONFromBytes(w, r, data)
}

// GET /id/{id}?site=siteID&url=post-url - gets a comment by id
func (s *Server) commentByIDCtrl(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	siteID := r.URL.Query().Get("site")
	url := r.URL.Query().Get("url")

	log.Printf("[DEBUG] get comments by id %s, %s %s", id, siteID, url)

	comment, err := s.DataService.Get(store.Locator{SiteID: siteID, URL: url}, id)
	if err != nil {
		common.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get comment by id")
		return
	}
	render.Status(r, http.StatusOK)
	renderJSONWithHTML(w, r, comment)
}

// GET /comments?site=siteID&user=id - returns comments for given userID
func (s *Server) findUserCommentsCtrl(w http.ResponseWriter, r *http.Request) {

	userID := r.URL.Query().Get("user")
	siteID := r.URL.Query().Get("site")

	log.Printf("[DEBUG] get comments for userID %s, %s", userID, siteID)

	data, err := s.respCache.get(r.URL.String(), time.Hour, func() ([]byte, error) {
		comments, e := s.DataService.User(siteID, userID)
		if e != nil {
			return nil, e
		}
		return encodeJSONWithHTML(comments)
	})

	if err != nil {
		common.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get comment by user id")
		return
	}
	renderJSONFromBytes(w, r, data)
}

// GET /user - returns user info
func (s *Server) userInfoCtrl(w http.ResponseWriter, r *http.Request) {
	user, err := common.GetUserInfo(r)
	if err != nil {
		common.SendErrorJSON(w, r, http.StatusUnauthorized, err, "can't get user info")
		return
	}
	render.JSON(w, r, user)
}

// GET /count?site=siteID&url=post-url - get number of comments for given post
func (s *Server) countCtrl(w http.ResponseWriter, r *http.Request) {
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	count, err := s.DataService.Count(locator)
	if err != nil {
		common.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get count")
		return
	}
	render.JSON(w, r, JSON{"count": count, "loc": locator})
}

// GET /list?site=siteID - list posts with comments
func (s *Server) listCtrl(w http.ResponseWriter, r *http.Request) {

	siteID := r.URL.Query().Get("site")
	data, err := s.respCache.get(r.URL.String(), 8*time.Hour, func() ([]byte, error) {
		posts, e := s.DataService.List(siteID)
		if e != nil {
			return nil, e
		}
		return encodeJSONWithHTML(posts)
	})

	if err != nil {
		common.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get list of comments for "+siteID)
		return
	}
	renderJSONFromBytes(w, r, data)
}

// PUT /vote/{id}?site=siteID&url=post-url&vote=1 - vote for/against comment
func (s *Server) voteCtrl(w http.ResponseWriter, r *http.Request) {

	user, err := common.GetUserInfo(r)
	if err != nil {
		common.SendErrorJSON(w, r, http.StatusUnauthorized, err, "can't get user info")
		return
	}
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	id := chi.URLParam(r, "id")
	log.Printf("[DEBUG] vote for comment %s", id)

	vote := r.URL.Query().Get("vote") == "1"

	comment, err := s.DataService.Vote(locator, id, user.ID, vote)
	if err != nil {
		common.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't vote for comment")
		return
	}
	s.respCache.flush()
	render.JSON(w, r, JSON{"id": comment.ID, "score": comment.Score})
}

// renderJSONWithHTML allows html tags and forces charset=utf-8
func renderJSONWithHTML(w http.ResponseWriter, r *http.Request, v interface{}) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if status, ok := r.Context().Value(render.StatusCtxKey).(int); ok {
		w.WriteHeader(status)
	}
	_, _ = w.Write(buf.Bytes())
}

func encodeJSONWithHTML(v interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, errors.Wrap(err, "can't encode to json")
	}
	return buf.Bytes(), nil
}

// renderJSONWithHTML allows html tags and forces charset=utf-8
func renderJSONFromBytes(w http.ResponseWriter, r *http.Request, data []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if status, ok := r.Context().Value(render.StatusCtxKey).(int); ok {
		w.WriteHeader(status)
	}
	_, _ = w.Write(data)
}

// serves static files from /web
func (s *Server) addFileServer(r chi.Router, path string, root http.FileSystem) {
	fs := http.StripPrefix(path, http.FileServer(root))

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// don't show dirs, just serve files
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		fs.ServeHTTP(w, r)
	}))
}
