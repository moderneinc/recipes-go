/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"strings"
	"testing"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
)

// Each handler the adapter cannot run the way v1 did holds its file back, and
// says why.
func TestHandlerBlockReasons(t *testing.T) {
	const header = `package main

import (
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

func use(...interface{}) {}

`
	for _, tc := range []struct{ name, body, reason string }{
		{"migrates", `func f() *sts.STS {
	sess := session.Must(session.NewSession())
	sess.Handlers.Send.PushBack(func(r *request.Request) { use(r.Operation.Name, r.AttemptTime, r.Params) })
	return sts.New(sess)
}`, ""},
		{"reads a field with no counterpart", `func f() {
	sess := session.Must(session.NewSession())
	sess.Handlers.Send.PushBack(func(r *request.Request) { use(r.Error) })
}`, "reading the request's Error"},
		{"keeps the input", `func f() {
	sess := session.Must(session.NewSession())
	sess.Handlers.Send.PushBack(func(r *request.Request) { p := r.Params; use(p) })
}`, "reading the request's Params"},
		{"passes the request on", `func f() {
	sess := session.Must(session.NewSession())
	sess.Handlers.Send.PushBack(func(r *request.Request) { use(r) })
}`, "passing its request on"},
		// The adapter's lists keep v1's Push and Set semantics, so a handler added
		// twice behaves as v1's did.
		{"in a loop", `func f() {
	sess := session.Must(session.NewSession())
	for i := 0; i < 2; i++ {
		sess.Handlers.Send.PushBack(func(r *request.Request) {})
	}
}`, ""},
		{"session from elsewhere", `func f(sess *session.Session) {
	sess.Handlers.Send.PushBack(func(r *request.Request) {})
}`, "did not build"},
		{"another list", `func f() {
	sess := session.Must(session.NewSession())
	sess.Handlers.Retry.PushBack(func(r *request.Request) {})
}`, "Retry handler list"},
		{"declared elsewhere", `func h(r *request.Request) {}

func f() {
	sess := session.Must(session.NewSession())
	sess.Handlers.Send.PushBack(h)
}`, "declared elsewhere"},
		{"replaces a core handler", `func f() {
	sess := session.Must(session.NewSession())
	sess.Handlers.Send.SetBackNamed(request.NamedHandler{Name: "core.SendHandler", Fn: func(r *request.Request) {}})
}`, "core.SendHandler"},
		{"same name twice", `func f() {
	sess := session.Must(session.NewSession())
	sess.Handlers.Send.SetBackNamed(request.NamedHandler{Name: "n", Fn: func(r *request.Request) {}})
	sess.Handlers.Send.PushBackNamed(request.NamedHandler{Name: "n", Fn: func(r *request.Request) {}})
}`, ""},
		{"on a client", `func f() {
	svc := sts.New(session.Must(session.NewSession()))
	svc.Handlers.Send.PushBack(func(r *request.Request) {})
}`, "request handlers reached"},
		{"request type elsewhere", `var last *request.Request`, "request.Request"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cu, err := parser.NewGoParser().Parse("main.go", header+tc.body)
			if err != nil {
				t.Fatal(err)
			}
			s := scanFile(cu)
			if tc.reason == "" {
				if !s.migratable {
					t.Fatalf("held back: %s", s.reason)
				}
				return
			}
			if s.migratable || !strings.Contains(s.reason, tc.reason) {
				t.Fatalf("want held back for %q, got migratable=%v reason %q", tc.reason, s.migratable, s.reason)
			}
		})
	}
}
