// search is a service which exposes an API to text search a repo at a
// specific commit.
//
// Architecture Notes:
// * Archive is fetched from gitserver
// * Simple HTTP API exposed
// * Currently no concept of authorization
// * On disk cache of fetched archives to reduce load on gitserver
// * Run search on archive. Rely on OS file buffers
// * Simple to scale up since stateless
// * Use ingress with affinity to increase local cache hit ratio
package search

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/schema"
)

// ArchiveStore is how the service gets the content to search.
type ArchiveStore interface {
	// FetchZip returns a []byte to a zip archive. If the error implements
	// "BadRequest() bool", it will be used to determine if the error is a
	// bad request (eg invalid repo).
	//
	// NOTE: gitcmd.Open.Archive returns the bytes in memory. However, we
	// only need to be able to stream it in. Update to io.ReadCloser once
	// we have a nice way to stream in the archive.
	FetchZip(ctx context.Context, repo, commit string) ([]byte, error)
}

// Service is the search service. It is an http.Handler.
type Service struct {
	ArchiveStore ArchiveStore
}

var decoder = schema.NewDecoder()

// Params are the input for a search request. Most of the fields are based on
// PatternInfo used in vscode.
type Params struct {
	// Repo is which repository to search. eg "github.com/gorilla/mux"
	Repo string
	// Commit is which commit to search. It is required to be resolved,
	// not a ref like HEAD or master. eg
	// "599cba5e7b6137d46ddf58fb1765f5d928e69604"
	Commit string
	// Pattern is the search query. It is a regular expression if IsRegExp
	// is true, otherwise a fixed string. eg "route variable"
	Pattern string
	// IsRegExp if true will treat the Pattern as a regular expression.
	IsRegExp bool
	// IsWordMatch if true will only match the pattern at word boundaries.
	IsWordMatch bool
	// IsCaseSensitive if false will ignore the case of text and pattern
	// when finding matches.
	IsCaseSensitive bool
}

func (p Params) String() string {
	opts := make([]byte, 1, 4)
	opts[0] = ' '
	if p.IsRegExp {
		opts = append(opts, 'r')
	}
	if p.IsWordMatch {
		opts = append(opts, 'w')
	}
	if p.IsCaseSensitive {
		opts = append(opts, 'c')
	}
	var optsS string
	if len(opts) > 1 {
		optsS = string(opts)
	}

	return fmt.Sprintf("search.Params{%q%s}", p.Pattern, optsS)
}

// FileMatch is the struct used by vscode to receive search results
type FileMatch struct {
	Path        string
	LineMatches []LineMatch
}

// LineMatch is the struct used by vscode to receive search results for a line
type LineMatch struct {
	Preview    string
	LineNumber int
	// TODO vscode also wants to know the range of matches on the line.
	// OffsetAndLengths [][2]int
}

// ServeHTTP handles HTTP based search requests
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	var p Params
	err = decoder.Decode(&p, r.Form)
	if err != nil {
		http.Error(w, "failed to decode form: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err = validateParams(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	matches, err := s.search(r.Context(), &p)
	if err != nil {
		code := http.StatusInternalServerError
		if isBadRequest(err) {
			code = http.StatusBadRequest
		}
		http.Error(w, err.Error(), code)
		return
	}
	if matches == nil {
		// Return an empty list
		matches = make([]FileMatch, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(matches)
	if err != nil {
		// We may have already started writing to w
		http.Error(w, "failed to encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Service) search(ctx context.Context, p *Params) ([]FileMatch, error) {
	// TODO use platinum searcher or sift to search
	// TODO pretty aggressively skip files to search

	matcher, err := compile(p)
	if err != nil {
		return nil, badRequestError{err.Error()}
	}

	r, err := s.openReader(ctx, p.Repo, p.Commit)
	if err != nil {
		return nil, err
	}

	var matches []FileMatch
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		lm, err := matcher(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		if lm != nil {
			matches = append(matches, FileMatch{
				Path:        f.Name, // TODO name likely needs to be changed
				LineMatches: lm,
			})
		}
	}
	return matches, nil
}

// openReader will open a zip reader to the
func (s *Service) openReader(ctx context.Context, repo, commit string) (*zip.Reader, error) {
	// TODO single-flight
	// TODO disk backed with cache eviction
	// TODO rewrite zip on disk to be more efficient to access (prune files, etc)
	b, err := s.ArchiveStore.FetchZip(ctx, repo, commit)
	if err != nil {
		return nil, err
	}
	rAt := bytes.NewReader(b)
	return zip.NewReader(rAt, int64(len(b)))
}

func validateParams(p *Params) error {
	if p.Repo == "" {
		return errors.New("Repo must be non-empty")
	}
	// Surprisingly this is the same sanity check used in the git source.
	if len(p.Commit) != 40 {
		return fmt.Errorf("Commit must be resolved (Commit=%q)", p.Commit)
	}
	if p.Pattern == "" {
		return errors.New("Pattern must be non-empty")
	}
	return nil
}

type badRequestError struct{ msg string }

func (e badRequestError) Error() string    { return e.msg }
func (e badRequestError) BadRequest() bool { return true }

func isBadRequest(err error) bool {
	e, ok := err.(interface {
		BadRequest() bool
	})
	return ok && e.BadRequest()
}
