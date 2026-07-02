package rpc

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	_ "github.com/pms/backend/gen/pms/v1" // регистрирует дескрипторы pms.v1
)

// expectedReads — ЯВНАЯ классификация каждой read-процедуры. Тест ниже требует,
// чтобы каждая зарегистрированная процедура была либо здесь, либо мутацией по
// isReadProcedure()==false. Добавил новый RPC — обязан классифицировать его тут,
// иначе тест падает. Это страховка от convention-дыры: мутация, случайно
// названная с read-префиксом, станет доступной гостям и read-only токенам.
var expectedReads = map[string]bool{
	"/pms.v1.AttachmentService/GetAttachmentURL": true,
	"/pms.v1.ChatService/GetMessages":            true,
	"/pms.v1.CommentService/GetComments":         true,
	"/pms.v1.IssueService/GetActivity":           true,
	"/pms.v1.QueryService/Anomalies":             true,
	"/pms.v1.QueryService/Metric":                true,
	"/pms.v1.QueryService/Mood":                  true,
	"/pms.v1.QueryService/RunView":               true,
	"/pms.v1.QueryService/Search":                true,
	"/pms.v1.QueryService/SemanticSearch":        true,
	"/pms.v1.QueryService/SimilarIssues":         true,
	"/pms.v1.QueryService/Suggest":               true,
	"/pms.v1.QueryService/Summarize":             true,
	"/pms.v1.QueryService/Themes":                true,
	"/pms.v1.SyncService/Bootstrap":              true,
	"/pms.v1.SyncService/Changes":                true,
	"/pms.v1.TimeService/Report":                 true,
}

// allProcedures собирает /pms.v1.Service/Method для всех зарегистрированных сервисов.
func allProcedures(t *testing.T) []string {
	t.Helper()
	var out []string
	protoregistry.GlobalFiles.RangeFilesByPackage("pms.v1", func(fd protoreflect.FileDescriptor) bool {
		for i := 0; i < fd.Services().Len(); i++ {
			svc := fd.Services().Get(i)
			for j := 0; j < svc.Methods().Len(); j++ {
				out = append(out, "/"+string(svc.FullName())+"/"+string(svc.Methods().Get(j).Name()))
			}
		}
		return true
	})
	if len(out) < 100 {
		t.Fatalf("expected 100+ procedures, got %d — descriptors not registered?", len(out))
	}
	return out
}

// TestProcedureClassification — каждый RPC классифицирован явно и корректно.
func TestProcedureClassification(t *testing.T) {
	for _, proc := range allProcedures(t) {
		got := isReadProcedure(proc)
		want := expectedReads[proc]
		if got != want {
			t.Errorf("%s: isReadProcedure=%v, expected %v — new RPC must be added to expectedReads or renamed off a read prefix", proc, got, want)
		}
	}
	// expectedReads не должен содержать несуществующих процедур (мусор при переименовании)
	registered := map[string]bool{}
	for _, p := range allProcedures(t) {
		registered[p] = true
	}
	for p := range expectedReads {
		if !registered[p] {
			t.Errorf("expectedReads contains unknown procedure %s", p)
		}
	}
}

// TestGuestAndAdminMapsExist — admin/guest-списки указывают только на реальные процедуры.
func TestGuestAndAdminMapsExist(t *testing.T) {
	registered := map[string]bool{}
	for _, p := range allProcedures(t) {
		registered[p] = true
	}
	for p := range adminProcedures {
		if !registered[p] {
			t.Errorf("adminProcedures contains unknown procedure %s", p)
		}
	}
	for p := range allowGuestProcedures {
		if !registered[p] {
			t.Errorf("allowGuestProcedures contains unknown procedure %s", p)
		}
		if isReadProcedure(p) {
			t.Errorf("allowGuestProcedures %s is already a read — remove", p)
		}
		if !strings.HasPrefix(p, "/pms.v1.") {
			t.Errorf("malformed procedure %s", p)
		}
	}
}
