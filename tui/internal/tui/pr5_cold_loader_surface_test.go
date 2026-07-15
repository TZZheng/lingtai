package tui

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestPR5Stage1ColdLoaderSurfaceContract is the compile-safe RED between the
// structural ThreadState/counter slice and typed behavioral loader tests. The
// behavioral tests must not be introduced until these internal production
// seams exist, otherwise missing identifiers would masquerade as behavior.
func TestPR5Stage1ColdLoaderSurfaceContract(t *testing.T) {
	inv := loadAsyncSourceInventory(t)
	var issues []string

	if !pr5Contains(inv.constantsOfType("asyncKind"), "asyncColdThreadLoad") {
		issues = append(issues, "asyncColdThreadLoad logical kind is absent")
	}

	for _, typeName := range []string{"threadLoadRequest", "threadLoadResultMsg", "threadLoadWorker"} {
		if _, ok := inv.types[typeName]; !ok {
			issues = append(issues, typeName+" production type is absent")
		}
	}

	for _, contract := range []struct {
		typeName string
		field    string
		wantType string
	}{
		{typeName: "threadLoadRequest", field: "envelope", wantType: "asyncEnvelope"},
		{typeName: "threadLoadRequest", field: "humanDir", wantType: "string"},
		{typeName: "threadLoadRequest", field: "humanAddress", wantType: "string"},
		{typeName: "threadLoadRequest", field: "targetAddress", wantType: "string"},
		{typeName: "threadLoadRequest", field: "targetDisplayName", wantType: "string"},
		{typeName: "threadLoadRequest", field: "acceptedMessages", wantType: "[]fs.MailMessage"},
		{typeName: "threadLoadRequest", field: "eventWindow", wantType: "int"},
		{typeName: "threadLoadRequest", field: "inquiryWindow", wantType: "int"},
		{typeName: "threadLoadResultMsg", field: "envelope", wantType: "asyncEnvelope"},
		{typeName: "threadLoadResultMsg", field: "sessionCache", wantType: "*fs.SessionCache"},
		{typeName: "threadLoadResultMsg", field: "err", wantType: "error"},
	} {
		if _, ok := inv.types[contract.typeName]; !ok {
			continue
		}
		got := pr5NamedFieldTypes(inv, contract.typeName, contract.field)
		if len(got) != 1 || got[0] != contract.wantType {
			issues = append(issues, contract.typeName+"."+contract.field+" type = "+strings.Join(got, ",")+"; want exactly "+contract.wantType)
		}
	}

	for _, owner := range []string{
		"newThreadLoadCoordinator",
		"ThreadLoadCoordinator.request",
		"ThreadLoadCoordinator.settle",
	} {
		if inv.findFunction(owner) == nil {
			issues = append(issues, owner+" production function is absent")
		}
	}

	for _, contract := range []struct {
		owner   string
		params  []string
		results []string
	}{
		{owner: "newThreadLoadCoordinator", params: []string{"threadLoadWorker"}, results: []string{"ThreadLoadCoordinator"}},
		{owner: "ThreadLoadCoordinator.request", params: []string{"threadLoadRequest"}, results: []string{"tea.Cmd"}},
		{owner: "ThreadLoadCoordinator.settle", params: []string{"asyncCurrent", "threadLoadResultMsg"}, results: []string{"*ThreadState", "tea.Cmd", "bool"}},
	} {
		fn := inv.findFunction(contract.owner)
		if fn == nil {
			continue
		}
		gotParams, gotResults := pr5FunctionSignature(inv.fset, fn)
		if strings.Join(gotParams, ",") != strings.Join(contract.params, ",") || strings.Join(gotResults, ",") != strings.Join(contract.results, ",") {
			issues = append(issues, contract.owner+" signature = ("+strings.Join(gotParams, ",")+") ("+strings.Join(gotResults, ",")+"); want ("+strings.Join(contract.params, ",")+") ("+strings.Join(contract.results, ",")+")")
		}
	}

	if _, ok := inv.types["threadLoadWorker"]; ok {
		gotParams, gotResults, found := pr5InterfaceMethodSignature(inv, "threadLoadWorker", "Load")
		if !found {
			issues = append(issues, "threadLoadWorker.Load production method is absent")
		} else if strings.Join(gotParams, ",") != "threadLoadRequest" || strings.Join(gotResults, ",") != "*fs.SessionCache,error" {
			issues = append(issues, "threadLoadWorker.Load signature = ("+strings.Join(gotParams, ",")+") ("+strings.Join(gotResults, ",")+"); want (threadLoadRequest) (*fs.SessionCache,error)")
		}
	}

	gotParams, gotResults, found := pr5SessionMethodSignature(t, "RebuildDirectThreadWindowedInMemory")
	if !found {
		issues = append(issues, "fs.SessionCache.RebuildDirectThreadWindowedInMemory production method is absent")
	} else {
		wantParams := []string{"[]MailMessage", "string", "string", "string", "string", "int", "int"}
		if strings.Join(gotParams, ",") != strings.Join(wantParams, ",") || len(gotResults) != 0 {
			issues = append(issues, "fs.SessionCache.RebuildDirectThreadWindowedInMemory signature = ("+strings.Join(gotParams, ",")+") ("+strings.Join(gotResults, ",")+"); want ("+strings.Join(wantParams, ",")+") ()")
		}
	}

	issues = append(issues, pr5ForbiddenColdLoaderOwners(inv)...)
	if len(issues) == 0 {
		return
	}
	sort.Strings(issues)
	t.Fatalf("missing PR5 behavioral cold-loader surface:\n  - %s", strings.Join(issues, "\n  - "))
}

// TestPR5Stage5OrdinaryActivationUsesOneProjectionOwner prevents the rail from
// constructing a second MailModel for an ordinary target and then repairing a
// mirrored ThreadState after every completion. ThreadState owns the active
// target coordinates; MailModel is the single reusable presentation surface.
func TestPR5Stage5OrdinaryActivationUsesOneProjectionOwner(t *testing.T) {
	inv := loadAsyncSourceInventory(t)
	var issues []string

	ordinary := inv.findFunction("App.activateOrdinaryRailRow")
	if ordinary == nil {
		issues = append(issues, "App.activateOrdinaryRailRow production function is absent")
	} else if got := countCalls(ordinary.Body, "NewMailModel"); got != 0 {
		issues = append(issues, "App.activateOrdinaryRailRow constructs NewMailModel; want the existing presentation surface rebound to one active ThreadState")
	}
	if inv.findFunction("App.syncCurrentThreadFromMail") != nil {
		issues = append(issues, "App.syncCurrentThreadFromMail mirror remains; want accepted state projected at the owning publication seam")
	}

	if len(issues) == 0 {
		return
	}
	sort.Strings(issues)
	t.Fatalf("ordinary rail activation still has split projection ownership:\n  - %s", strings.Join(issues, "\n  - "))
}

func pr5NamedFieldTypes(inv *asyncSourceInventory, typeName, fieldName string) []string {
	typeSpec := inv.types[typeName]
	if typeSpec == nil {
		return nil
	}
	structure, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return nil
	}
	var types []string
	for _, field := range structure.Fields.List {
		for _, name := range field.Names {
			if name.Name == fieldName {
				types = append(types, pr5ExprString(inv.fset, field.Type))
			}
		}
	}
	sort.Strings(types)
	return types
}

func pr5FunctionSignature(fset *token.FileSet, fn *ast.FuncDecl) ([]string, []string) {
	return pr5FieldListTypes(fset, fn.Type.Params), pr5FieldListTypes(fset, fn.Type.Results)
}

func pr5InterfaceMethodSignature(inv *asyncSourceInventory, typeName, methodName string) ([]string, []string, bool) {
	typeSpec := inv.types[typeName]
	if typeSpec == nil {
		return nil, nil, false
	}
	iface, ok := typeSpec.Type.(*ast.InterfaceType)
	if !ok {
		return nil, nil, false
	}
	for _, method := range iface.Methods.List {
		if len(method.Names) != 1 || method.Names[0].Name != methodName {
			continue
		}
		fn, ok := method.Type.(*ast.FuncType)
		if !ok {
			return nil, nil, false
		}
		return pr5FieldListTypes(inv.fset, fn.Params), pr5FieldListTypes(inv.fset, fn.Results), true
	}
	return nil, nil, false
}

func pr5FieldListTypes(fset *token.FileSet, fields *ast.FieldList) []string {
	if fields == nil {
		return nil
	}
	var types []string
	for _, field := range fields.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for range count {
			types = append(types, pr5ExprString(fset, field.Type))
		}
	}
	return types
}

func pr5SessionMethodSignature(t *testing.T, methodName string) ([]string, []string, bool) {
	t.Helper()
	fset := token.NewFileSet()
	path := filepath.Join("..", "fs", "session.go")
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse production %s: %v", path, err)
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name.Name != methodName {
			continue
		}
		if len(fn.Recv.List) == 1 && expressionName(fn.Recv.List[0].Type) == "SessionCache" {
			params, results := pr5FunctionSignature(fset, fn)
			return params, results, true
		}
	}
	return nil, nil, false
}

func pr5ForbiddenColdLoaderOwners(inv *asyncSourceInventory) []string {
	forbidden := []string{
		"ProjectMailStore",
		"ProjectMailSnapshot",
		"projectMailScanner",
		"fs.MailCache",
		"MailModel",
		"time.Ticker",
		"time.Timer",
	}
	var issues []string
	for _, typeName := range []string{
		"ThreadState",
		"ThreadLoadCoordinator",
		"threadLoadRequest",
		"threadLoadResultMsg",
		"threadLoadWorker",
	} {
		typeSpec := inv.types[typeName]
		if typeSpec == nil {
			continue
		}
		ast.Inspect(typeSpec.Type, func(node ast.Node) bool {
			expr, ok := node.(ast.Expr)
			if !ok {
				return true
			}
			rendered := pr5ExprString(inv.fset, expr)
			for _, name := range forbidden {
				if rendered == name || strings.Contains(rendered, name) {
					issues = append(issues, typeName+" must not own "+name)
				}
			}
			if strings.HasPrefix(rendered, "map[") && strings.Contains(rendered, "ThreadState") {
				issues = append(issues, typeName+" must not own an inactive ThreadState map")
			}
			return true
		})
	}
	return uniqueStrings(issues)
}

func pr5ExprString(fset *token.FileSet, expr ast.Expr) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, expr); err != nil {
		return ""
	}
	return buf.String()
}

func pr5Contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
