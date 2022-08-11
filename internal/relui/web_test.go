// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package relui

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/build/internal/releasetargets"
	"golang.org/x/build/internal/relui/db"
	"golang.org/x/build/internal/workflow"
)

// testStatic is our static web server content.
//
//go:embed testing
var testStatic embed.FS

func TestFileServerHandler(t *testing.T) {
	cases := []struct {
		desc        string
		path        string
		wantCode    int
		wantBody    string
		wantHeaders map[string]string
	}{
		{
			desc:     "sets headers and returns file",
			path:     "/testing/test.css",
			wantCode: http.StatusOK,
			wantBody: "/**\n * Copyright 2022 Go Authors All rights reserved.\n " +
				"* Use of this source code is governed by a BSD-style\n " +
				"* license that can be found in the LICENSE file.\n */\n\n.Header { font-size: 10rem; }\n",
			wantHeaders: map[string]string{
				"Content-Type":  "text/css; charset=utf-8",
				"Cache-Control": "no-cache, private, max-age=0",
			},
		},
		{
			desc:     "handles missing file",
			path:     "/foo.js",
			wantCode: http.StatusNotFound,
			wantBody: "404 page not found\n",
			wantHeaders: map[string]string{
				"Content-Type": "text/plain; charset=utf-8",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, c.path, nil)
			w := httptest.NewRecorder()

			m := &metricsRouter{router: httprouter.New()}
			m.ServeFiles("/*filepath", http.FS(testStatic))
			m.ServeHTTP(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != c.wantCode {
				t.Errorf("rep.StatusCode = %d, wanted %d", resp.StatusCode, c.wantCode)
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("resp.Body = _, %v, wanted no error", err)
			}
			if string(b) != c.wantBody {
				t.Errorf("resp.Body = %q, %v, wanted %q, %v", b, err, c.wantBody, nil)
			}
			for k, v := range c.wantHeaders {
				if resp.Header.Get(k) != v {
					t.Errorf("resp.Header.Get(%q) = %q, wanted %q", k, resp.Header.Get(k), v)
				}
			}
		})
	}
}

func TestServerHomeHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := testDB(ctx, t)

	q := db.New(p)
	wf := db.CreateWorkflowParams{ID: uuid.New()}
	if _, err := q.CreateWorkflow(ctx, wf); err != nil {
		t.Fatalf("CreateWorkflow(_, %v) = _, %v, wanted no error", wf, err)
	}
	tp := db.CreateTaskParams{
		WorkflowID: wf.ID,
		Name:       "TestTask",
		Result:     nullString(`{"Filename": "foo.exe"}`),
	}
	if _, err := q.CreateTask(ctx, tp); err != nil {
		t.Fatalf("CreateTask(_, %v) = _, %v, wanted no error", tp, err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	s := NewServer(p, NewWorker(NewDefinitionHolder(), p, &PGListener{p}), nil, SiteHeader{}, nil)
	s.homeHandler(w, req)
	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("resp.StatusCode = %d, wanted %d", resp.StatusCode, http.StatusOK)
	}
}

func TestServerNewWorkflowHandler(t *testing.T) {
	cases := []struct {
		desc     string
		params   url.Values
		wantCode int
	}{
		{
			desc:     "No selection",
			wantCode: http.StatusOK,
		},
		{
			desc:     "valid workflow",
			params:   url.Values{"workflow.name": []string{"echo"}},
			wantCode: http.StatusOK,
		},
		{
			desc:     "invalid workflow",
			params:   url.Values{"workflow.name": []string{"this workflow does not exist"}},
			wantCode: http.StatusOK,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			u := url.URL{Path: "/new_workflow", RawQuery: c.params.Encode()}
			req := httptest.NewRequest(http.MethodGet, u.String(), nil)
			w := httptest.NewRecorder()

			s := NewServer(testDB(ctx, t), NewWorker(NewDefinitionHolder(), nil, nil), nil, SiteHeader{}, nil)
			s.newWorkflowHandler(w, req)
			resp := w.Result()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("rep.StatusCode = %d, wanted %d", resp.StatusCode, http.StatusOK)
			}
		})
	}
}

func TestServerCreateWorkflowHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cases := []struct {
		desc          string
		params        url.Values
		wantCode      int
		wantWorkflows []db.Workflow
	}{
		{
			desc:     "no params",
			wantCode: http.StatusBadRequest,
		},
		{
			desc:     "invalid workflow name",
			params:   url.Values{"workflow.name": []string{"invalid"}},
			wantCode: http.StatusBadRequest,
		},
		{
			desc:     "missing workflow params",
			params:   url.Values{"workflow.name": []string{"echo"}},
			wantCode: http.StatusBadRequest,
		},
		{
			desc: "successful creation",
			params: url.Values{
				"workflow.name":            []string{"echo"},
				"workflow.params.greeting": []string{"hello"},
				"workflow.params.farewell": []string{"bye"},
			},
			wantCode: http.StatusSeeOther,
			wantWorkflows: []db.Workflow{
				{
					ID:        uuid.New(), // SameUUIDVariant
					Params:    nullString(`{"farewell": "bye", "greeting": "hello"}`),
					Name:      nullString(`echo`),
					Output:    "{}",
					CreatedAt: time.Now(), // cmpopts.EquateApproxTime
					UpdatedAt: time.Now(), // cmpopts.EquateApproxTime
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			p := testDB(ctx, t)
			req := httptest.NewRequest(http.MethodPost, "/workflows/create", strings.NewReader(c.params.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()
			q := db.New(p)

			s := NewServer(p, NewWorker(NewDefinitionHolder(), p, &PGListener{p}), nil, SiteHeader{}, nil)
			s.createWorkflowHandler(rec, req)
			resp := rec.Result()

			if resp.StatusCode != c.wantCode {
				t.Errorf("rep.StatusCode = %d, wanted %d", resp.StatusCode, c.wantCode)
			}
			if c.wantCode == http.StatusBadRequest {
				return
			}
			wfs, err := q.Workflows(ctx)
			if err != nil {
				t.Fatalf("q.Workflows() = %v, %v, wanted no error", wfs, err)
			}
			if diff := cmp.Diff(c.wantWorkflows, wfs, SameUUIDVariant(), cmpopts.EquateApproxTime(time.Minute)); diff != "" {
				t.Fatalf("q.Workflows() mismatch (-want +got):\n%s", diff)
			}
			if c.wantCode == http.StatusSeeOther {
				got := resp.Header.Get("Location")
				want := path.Join("/workflows", wfs[0].ID.String())
				if got != want {
					t.Fatalf("resp.Headers.Get(%q) = %q, wanted %q", "Location", got, want)
				}
			}
		})
	}
}

// resetDB truncates the db connected to in the pgxpool.Pool
// connection.
//
// All tables in the public schema of the connected database will be
// truncated except for the migrations table.
func resetDB(ctx context.Context, t *testing.T, p *pgxpool.Pool) {
	t.Helper()
	tableQuery := `SELECT table_name FROM information_schema.tables WHERE table_schema='public'`
	rows, err := p.Query(ctx, tableQuery)
	if err != nil {
		t.Fatalf("p.Query(_, %q, %q) = %v, %v, wanted no error", tableQuery, "public", rows, err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("rows.Scan() = %v, wanted no error", err)
		}
		if name == "migrations" {
			continue
		}
		truncQ := fmt.Sprintf("TRUNCATE %s CASCADE", pgx.Identifier{name}.Sanitize())
		c, err := p.Exec(ctx, truncQ)
		if err != nil {
			t.Fatalf("p.Exec(_, %q) = %v, %v", truncQ, c, err)
		}
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("rows.Err() = %v, wanted no error", err)
	}
}

var testPoolOnce sync.Once
var testPool *pgxpool.Pool

// testDB connects, creates, and migrates a database in preparation
// for testing, and returns a connection pool to the prepared
// database.
//
// The connection pool is closed as part of a t.Cleanup handler.
// Database connections are expected to be configured through libpq
// compatible environment variables. If no PGDATABASE is specified,
// relui-test will be used.
//
// https://www.postgresql.org/docs/current/libpq-envars.html
func testDB(ctx context.Context, t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping database tests in short mode.")
	}
	testPoolOnce.Do(func() {
		pgdb := url.QueryEscape(os.Getenv("PGDATABASE"))
		if pgdb == "" {
			pgdb = "relui-test"
		}
		if err := InitDB(ctx, fmt.Sprintf("database=%v", pgdb)); err != nil {
			t.Skipf("Skipping database integration test: %v", err)
		}
		p, err := pgxpool.Connect(ctx, fmt.Sprintf("database=%v", pgdb))
		if err != nil {
			t.Skipf("Skipping database integration test: %v", err)
		}
		testPool = p
	})
	if testPool == nil {
		t.Skip("Skipping database integration test: testdb = nil. See first error for details.")
		return nil
	}
	t.Cleanup(func() {
		resetDB(context.Background(), t, testPool)
	})
	return testPool
}

// SameUUIDVariant considers UUIDs equal if they are both the same
// uuid.Variant. Zero-value uuids are considered equal.
func SameUUIDVariant() cmp.Option {
	return cmp.Transformer("SameVariant", func(v uuid.UUID) uuid.Variant {
		return v.Variant()
	})
}

func TestSameUUIDVariant(t *testing.T) {
	cases := []struct {
		desc string
		x    uuid.UUID
		y    uuid.UUID
		want bool
	}{
		{
			desc: "both set",
			x:    uuid.New(),
			y:    uuid.New(),
			want: true,
		},
		{
			desc: "both unset",
			want: true,
		},
		{
			desc: "just x",
			x:    uuid.New(),
			want: false,
		},
		{
			desc: "just y",
			y:    uuid.New(),
			want: false,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if got := cmp.Equal(c.x, c.y, SameUUIDVariant()); got != c.want {
				t.Fatalf("cmp.Equal(%v, %v, SameUUIDVariant()) = %t, wanted %t", c.x, c.y, got, c.want)
			}
		})
	}
}

// nullString returns a sql.NullString for a string.
func nullString(val string) sql.NullString {
	return sql.NullString{String: val, Valid: true}
}

func TestServerBaseLink(t *testing.T) {
	cases := []struct {
		desc    string
		baseURL string
		target  string
		extras  []string
		want    string
	}{
		{
			desc:   "no baseURL, relative",
			target: "/workflows",
			want:   "/workflows",
		},
		{
			desc:   "no baseURL, absolute",
			target: "https://example.test/something",
			want:   "https://example.test/something",
		},
		{
			desc:    "absolute baseURL, relative",
			baseURL: "https://example.test/releases",
			target:  "/workflows",
			want:    "https://example.test/releases/workflows",
		},
		{
			desc:    "relative baseURL, relative",
			baseURL: "/releases",
			target:  "/workflows",
			want:    "/releases/workflows",
		},
		{
			desc:    "relative baseURL, relative with extras",
			baseURL: "/releases",
			target:  "/workflows",
			extras:  []string{"a-workflow"},
			want:    "/releases/workflows/a-workflow",
		},
		{
			desc:    "absolute baseURL, absolute",
			baseURL: "https://example.test/releases",
			target:  "https://example.test/something",
			want:    "https://example.test/something",
		},
		{
			desc:    "absolute baseURL, absolute with extras",
			baseURL: "https://example.test/releases",
			target:  "https://example.test/something",
			extras:  []string{"else"},
			want:    "https://example.test/something/else",
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			base, err := url.Parse(c.baseURL)
			if err != nil {
				t.Fatalf("url.Parse(%q) = %v, %v, wanted no error", c.baseURL, base, err)
			}
			s := NewServer(nil, nil, base, SiteHeader{}, nil)

			got := s.BaseLink(c.target, c.extras...)
			if got != c.want {
				t.Errorf("s.BaseLink(%q) = %q, wanted %q", c.target, got, c.want)
			}
		})
	}
}

func TestServerApproveTaskHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hourAgo := time.Now().Add(-1 * time.Hour)
	wfID := uuid.New()

	cases := []struct {
		desc        string
		params      map[string]string
		wantCode    int
		wantHeaders map[string]string
		want        db.Task
	}{
		{
			desc:     "no params",
			wantCode: http.StatusNotFound,
			want: db.Task{
				WorkflowID: wfID,
				Name:       "approve please",
				CreatedAt:  hourAgo,
				UpdatedAt:  hourAgo,
			},
		},
		{
			desc:     "invalid workflow id",
			params:   map[string]string{"id": "invalid", "name": "greeting"},
			wantCode: http.StatusBadRequest,
			want: db.Task{
				WorkflowID: wfID,
				Name:       "approve please",
				CreatedAt:  hourAgo,
				UpdatedAt:  hourAgo,
			},
		},
		{
			desc:     "wrong workflow id",
			params:   map[string]string{"id": uuid.New().String(), "name": "greeting"},
			wantCode: http.StatusNotFound,
			want: db.Task{
				WorkflowID: wfID,
				Name:       "approve please",
				CreatedAt:  hourAgo,
				UpdatedAt:  hourAgo,
			},
		},
		{
			desc:     "invalid task name",
			params:   map[string]string{"id": wfID.String(), "name": "invalid"},
			wantCode: http.StatusNotFound,
			want: db.Task{
				WorkflowID: wfID,
				Name:       "approve please",
				CreatedAt:  hourAgo,
				UpdatedAt:  hourAgo,
			},
		},
		{
			desc:     "successful approval",
			params:   map[string]string{"id": wfID.String(), "name": "approve please"},
			wantCode: http.StatusSeeOther,
			wantHeaders: map[string]string{
				"Location": path.Join("/workflows", wfID.String()),
			},
			want: db.Task{
				WorkflowID: wfID,
				Name:       "approve please",
				CreatedAt:  hourAgo,
				UpdatedAt:  time.Now(),
				ApprovedAt: sql.NullTime{Time: time.Now(), Valid: true},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			p := testDB(ctx, t)
			q := db.New(p)

			wf := db.CreateWorkflowParams{
				ID:        wfID,
				Params:    nullString(`{"farewell": "bye", "greeting": "hello"}`),
				Name:      nullString(`echo`),
				CreatedAt: hourAgo,
				UpdatedAt: hourAgo,
			}
			if _, err := q.CreateWorkflow(ctx, wf); err != nil {
				t.Fatalf("CreateWorkflow(_, %v) = _, %v, wanted no error", wf, err)
			}
			gtg := db.CreateTaskParams{
				WorkflowID: wf.ID,
				Name:       "approve please",
				Finished:   false,
				CreatedAt:  hourAgo,
				UpdatedAt:  hourAgo,
			}
			if _, err := q.CreateTask(ctx, gtg); err != nil {
				t.Fatalf("CreateTask(_, %v) = _, %v, wanted no error", gtg, err)
			}

			req := httptest.NewRequest(http.MethodPost, path.Join("/workflows/", c.params["id"], "tasks", url.PathEscape(c.params["name"]), "approve"), nil)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()
			s := NewServer(p, NewWorker(NewDefinitionHolder(), p, &PGListener{p}), nil, SiteHeader{}, nil)

			s.m.ServeHTTP(rec, req)
			resp := rec.Result()

			if resp.StatusCode != c.wantCode {
				t.Errorf("rep.StatusCode = %d, wanted %d", resp.StatusCode, c.wantCode)
			}
			for k, v := range c.wantHeaders {
				if resp.Header.Get(k) != v {
					t.Errorf("resp.Header.Get(%q) = %q, wanted %q", k, resp.Header.Get(k), v)
				}
			}
			if c.wantCode == http.StatusBadRequest {
				return
			}
			task, err := q.Task(ctx, db.TaskParams{
				WorkflowID: wf.ID,
				Name:       "approve please",
			})
			if err != nil {
				t.Fatalf("q.Task() = %v, %v, wanted no error", task, err)
			}
			if diff := cmp.Diff(c.want, task, cmpopts.EquateApproxTime(time.Minute)); diff != "" {
				t.Fatalf("q.Task() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestServerStopWorkflow(t *testing.T) {
	wfID := uuid.New()
	cases := []struct {
		desc        string
		params      map[string]string
		wantCode    int
		wantHeaders map[string]string
		wantLogs    []db.TaskLog
		wantCancel  bool
	}{
		{
			desc:     "no params",
			wantCode: http.StatusMethodNotAllowed,
		},
		{
			desc:     "invalid workflow id",
			params:   map[string]string{"id": "invalid"},
			wantCode: http.StatusBadRequest,
		},
		{
			desc:     "wrong workflow id",
			params:   map[string]string{"id": uuid.New().String()},
			wantCode: http.StatusNotFound,
		},
		{
			desc:     "successful stop",
			params:   map[string]string{"id": wfID.String()},
			wantCode: http.StatusSeeOther,
			wantHeaders: map[string]string{
				"Location": "/",
			},
			wantCancel: true,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			req := httptest.NewRequest(http.MethodPost, path.Join("/workflows/", c.params["id"], "stop"), nil)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()
			worker := NewWorker(NewDefinitionHolder(), nil, nil)

			wf := &workflow.Workflow{ID: wfID}
			if err := worker.markRunning(wf, cancel); err != nil {
				t.Fatalf("worker.markRunning(%v, %v) = %v, wanted no error", wf, cancel, err)
			}

			s := NewServer(nil, worker, nil, SiteHeader{}, nil)
			s.m.ServeHTTP(rec, req)
			resp := rec.Result()

			if resp.StatusCode != c.wantCode {
				t.Errorf("rep.StatusCode = %d, wanted %d", resp.StatusCode, c.wantCode)
			}
			for k, v := range c.wantHeaders {
				if resp.Header.Get(k) != v {
					t.Errorf("resp.Header.Get(%q) = %q, wanted %q", k, resp.Header.Get(k), v)
				}
			}
			if c.wantCancel {
				<-ctx.Done()
				return
			}
			if ctx.Err() != nil {
				t.Errorf("tx.Err() = %v, wanted no error", ctx.Err())
			}
		})
	}
}

func TestResultDetail(t *testing.T) {
	cases := []struct {
		desc     string
		input    string
		want     *resultDetail
		wantKind string
	}{
		{
			desc:     "string",
			input:    `"hello"`,
			want:     &resultDetail{String: "hello"},
			wantKind: "String",
		},
		{
			desc:     "artifact",
			input:    `{"Filename": "foo.exe", "Target": {"Name": "windows-test"}}`,
			want:     &resultDetail{Artifact: artifact{Filename: "foo.exe", Target: &releasetargets.Target{Name: "windows-test"}}},
			wantKind: "Artifact",
		},
		{
			desc:     "artifact missing target",
			input:    `{"Filename": "foo.exe"}`,
			want:     &resultDetail{Artifact: artifact{Filename: "foo.exe"}},
			wantKind: "Artifact",
		},
		{
			desc:     "nested json string",
			input:    `{"SomeOutput": "hello"}`,
			want:     &resultDetail{Outputs: map[string]*resultDetail{"SomeOutput": {String: "hello"}}},
			wantKind: "Outputs",
		},
		{
			desc:     "nested json complex",
			input:    `{"SomeOutput": {"Filename": "go.exe"}}`,
			want:     &resultDetail{Outputs: map[string]*resultDetail{"SomeOutput": {Artifact: artifact{Filename: "go.exe"}}}},
			wantKind: "Outputs",
		},
		{
			desc:  "nested json slice",
			input: `{"SomeOutput": [{"Filename": "go.exe"}]}`,
			want: &resultDetail{Outputs: map[string]*resultDetail{"SomeOutput": {Slice: []*resultDetail{{
				Artifact: artifact{Filename: "go.exe"},
			}}}}},
			wantKind: "Outputs",
		},
		{
			desc:  "nested json output",
			input: `{"SomeOutput": {"OtherOutput": "go.exe", "Next": 123, "Thing": {"foo": "bar"}, "Sauces": ["cranberry", "pizza"]}}`,
			want: &resultDetail{Outputs: map[string]*resultDetail{
				"SomeOutput": {Outputs: map[string]*resultDetail{
					"OtherOutput": {String: "go.exe"},
					"Next":        {Number: 123},
					"Thing":       {Outputs: map[string]*resultDetail{"foo": {String: "bar"}}},
					"Sauces":      {Slice: []*resultDetail{{String: "cranberry"}, {String: "pizza"}}},
				}}}},
			wantKind: "Outputs",
		},
		{
			desc:  "null json",
			input: `null`,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got := unmarshalResultDetail(c.input)

			if got.Kind() != c.wantKind {
				t.Errorf("got.Kind() = %q, wanted %q", got.Kind(), c.wantKind)
			}
			if diff := cmp.Diff(c.want, got); diff != "" {
				t.Errorf("unmarshalResultDetail mismatch (-want +got):\n%s", diff)
			}
		})
	}

}

func TestWorkflowParams(t *testing.T) {
	got, err := workflowParams(db.Workflow{Params: nullString(`{"greeting": "hello", "names": ["alice", "bob"]}`)})
	if err != nil {
		t.Fatalf("workflowParams: err = %v, wanted no error", err)
	}
	want := map[string]string{
		"greeting": `"hello"`,
		"names":    `["alice", "bob"]`,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("workflowParams mismatch (-want +got):\n%s", diff)
	}
}
