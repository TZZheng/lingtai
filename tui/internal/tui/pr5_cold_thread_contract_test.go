package tui

import (
	"fmt"
	"go/ast"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unicode"

	"github.com/anthropics/lingtai-tui/internal/inventory"
)

func TestPR5Stage1AsyncTargetIdentityContract(t *testing.T) {
	inv := loadAsyncSourceInventory(t)
	target, ok := pr5StructType(inv, "asyncTarget")
	if !ok {
		t.Fatal("missing PR5 async target identity contract: asyncTarget must carry an explicit target policy and exact PID/incarnation")
	}

	var issues []string
	var hasPolicy, hasPIDIncarnation bool
	for _, field := range target.Fields.List {
		for _, name := range field.Names {
			normalizedName := pr5Normalize(name.Name)
			if normalizedName == "inventorybound" {
				issues = append(issues, "asyncTarget still relies on the ambiguous inventoryBound Boolean")
			}
			if strings.Contains(normalizedName, "policy") || pr5ExprHasWord(field.Type, "policy") {
				if ident, isIdent := field.Type.(*ast.Ident); !isIdent || ident.Name != "bool" {
					hasPolicy = true
				}
			}
			if strings.Contains(normalizedName, "pid") || strings.Contains(normalizedName, "incarnation") ||
				pr5ExprHasWord(field.Type, "pid") || pr5ExprHasWord(field.Type, "incarnation") {
				hasPIDIncarnation = true
			}
		}
	}
	if !hasPolicy {
		issues = append(issues, "asyncTarget has no explicit non-Boolean target policy coordinate")
	}
	if !hasPIDIncarnation {
		issues = append(issues, "asyncTarget has no exact PID/incarnation coordinate")
	}
	if len(issues) != 0 {
		sort.Strings(issues)
		t.Fatalf("missing PR5 async target identity contract:\n  - %s", strings.Join(issues, "\n  - "))
	}
}

func TestPR5Stage1RailRevalidationDoesNotUseGlobalEnterable(t *testing.T) {
	enterable, ok := reflect.TypeOf(inventory.Record{}).FieldByName("Enterable")
	if !ok || enterable.Type.Kind() != reflect.Bool {
		t.Fatal("PR5 must leave global inventory.Record.Enterable unchanged as a Boolean while adding a separate ordinary-rail policy")
	}

	inv := loadAsyncSourceInventory(t)
	var candidates, gates []string
	for _, fn := range inv.funcs {
		name := strings.ToLower(fn.Name.Name)
		if !strings.Contains(name, "rail") ||
			!(strings.Contains(name, "eligible") || strings.Contains(name, "eligibility") || strings.Contains(name, "revalidate")) {
			continue
		}
		candidates = append(candidates, fn.Name.Name)
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Enterable" {
				return true
			}
			position := inv.fset.Position(selector.Sel.Pos())
			gates = append(gates, fmt.Sprintf("%s at %s:%d", fn.Name.Name, position.Filename, position.Line))
			return true
		})
	}
	if len(candidates) == 0 {
		t.Fatal("missing PR5 rail activation contract: no separate ordinary-rail eligibility/revalidation function exists; global project-visit Enterable behavior must remain independent")
	}
	if len(gates) != 0 {
		sort.Strings(gates)
		t.Fatalf("missing PR5 rail activation contract: the separate ordinary-rail policy is gated through global inventory.Record.Enterable (%s); global project-visit Enterable behavior must remain independent", strings.Join(gates, ", "))
	}
}

func TestPR5Stage1ColdThreadStateOwnershipContract(t *testing.T) {
	inv := loadAsyncSourceInventory(t)
	var issues []string

	threadState, ok := pr5StructType(inv, "ThreadState")
	if !ok {
		issues = append(issues, "the explicit cold lightweight ThreadState surface does not exist")
	} else {
		issues = append(issues, pr5ForbiddenColdOwnerFields("ThreadState", threadState)...)
	}

	for typeName, typeSpec := range inv.types {
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}
		lowerName := strings.ToLower(typeName)
		if typeName != "ThreadState" && strings.Contains(lowerName, "thread") &&
			(strings.Contains(lowerName, "coordinator") || strings.Contains(lowerName, "loader")) {
			issues = append(issues, pr5ForbiddenColdOwnerFields(typeName, structType)...)
		}
		for _, field := range structType.Fields.List {
			ast.Inspect(field.Type, func(node ast.Node) bool {
				mapType, ok := node.(*ast.MapType)
				if !ok || !pr5ExprHasWord(mapType.Value, "threadstate") {
					return true
				}
				fieldName := "<embedded>"
				if len(field.Names) != 0 {
					fieldName = field.Names[0].Name
				}
				issues = append(issues, fmt.Sprintf("%s.%s retains a warm ThreadState map owned by PR7", typeName, fieldName))
				return true
			})
		}
	}

	if len(issues) != 0 {
		sort.Strings(issues)
		t.Fatalf("missing PR5 cold ThreadState ownership contract (one lightweight cold state; no second ProjectMailStore/MailCache/scanner/tick owner and no retained warm-state map):\n  - %s", strings.Join(issues, "\n  - "))
	}
}

func TestPR5Stage1HonestResourceAccountingContract(t *testing.T) {
	inv := loadAsyncSourceInventory(t)
	surfaces := make(map[string]map[string]string)
	genericCancellation := make(map[string]string)

	for typeName, typeSpec := range inv.types {
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}
		for _, field := range structType.Fields.List {
			for _, name := range field.Names {
				pr5RecordCounterSurface(surfaces, genericCancellation, typeName, name.Name)
			}
		}
	}
	for _, fn := range inv.funcs {
		if fn.Recv == nil || len(fn.Recv.List) == 0 {
			continue
		}
		if typeName := pr5ReceiverName(fn.Recv.List[0].Type); typeName != "" {
			pr5RecordCounterSurface(surfaces, genericCancellation, typeName, fn.Name.Name)
		}
	}

	want := []string{"started", "coalesced", "completed", "true-cancelled", "stale-dropped"}
	bestName := "<none>"
	bestScore := 0
	var best map[string]string
	for typeName, got := range surfaces {
		if len(got) > bestScore || len(got) == bestScore && typeName < bestName {
			bestName, bestScore, best = typeName, len(got), got
		}
	}
	var missing []string
	for _, counter := range want {
		if best == nil || best[counter] == "" {
			missing = append(missing, counter)
		}
	}
	if len(missing) != 0 {
		detail := ""
		if generic := genericCancellation[bestName]; generic != "" {
			detail = fmt.Sprintf("; %s is only a generic cancellation name and cannot substitute for true-cancelled", generic)
		}
		t.Fatalf("missing PR5 honest resource-accounting contract: one surface must expose separate started/coalesced/completed/true-cancelled/stale-dropped counters; closest surface %s is missing %s%s. Work that physically completes must not be reported as cancelled", bestName, strings.Join(missing, ", "), detail)
	}
}

func pr5StructType(inv *asyncSourceInventory, name string) (*ast.StructType, bool) {
	typeSpec, ok := inv.types[name]
	if !ok {
		return nil, false
	}
	structType, ok := typeSpec.Type.(*ast.StructType)
	return structType, ok
}

func pr5Normalize(name string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, name)
}

func pr5ExprHasWord(expr ast.Expr, word string) bool {
	word = pr5Normalize(word)
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if ok && strings.Contains(pr5Normalize(ident.Name), word) {
			found = true
			return false
		}
		return !found
	})
	return found
}

func pr5ForbiddenColdOwnerFields(typeName string, structType *ast.StructType) []string {
	var issues []string
	for _, field := range structType.Fields.List {
		fieldName := "<embedded>"
		if len(field.Names) != 0 {
			fieldName = field.Names[0].Name
		}
		for _, forbidden := range []string{"ProjectMailStore", "MailCache", "projectMailScanner"} {
			if pr5ExprHasWord(field.Type, forbidden) {
				issues = append(issues, fmt.Sprintf("%s.%s owns forbidden %s", typeName, fieldName, forbidden))
			}
		}
		normalizedField := pr5Normalize(fieldName)
		if pr5ExprHasWord(field.Type, "ticker") || pr5ExprHasWord(field.Type, "timer") ||
			((strings.Contains(normalizedField, "tick") || strings.Contains(normalizedField, "poll")) && pr5OwnsRecurringWork(field.Type)) {
			issues = append(issues, fmt.Sprintf("%s.%s owns forbidden recurring tick/poll work", typeName, fieldName))
		}
	}
	return issues
}

func pr5OwnsRecurringWork(expr ast.Expr) bool {
	owns := false
	ast.Inspect(expr, func(node ast.Node) bool {
		switch node.(type) {
		case *ast.ChanType, *ast.FuncType:
			owns = true
			return false
		}
		return !owns
	})
	return owns
}

func pr5ReceiverName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.StarExpr:
		return pr5ReceiverName(expr.X)
	case *ast.IndexExpr:
		return pr5ReceiverName(expr.X)
	case *ast.IndexListExpr:
		return pr5ReceiverName(expr.X)
	default:
		return ""
	}
}

func pr5RecordCounterSurface(surfaces map[string]map[string]string, genericCancellation map[string]string, typeName, memberName string) {
	normalized := pr5Normalize(memberName)
	counter := ""
	switch {
	case strings.Contains(normalized, "stale") && strings.Contains(normalized, "drop"):
		counter = "stale-dropped"
	case strings.Contains(normalized, "true") && strings.Contains(normalized, "cancel"):
		counter = "true-cancelled"
	case strings.Contains(normalized, "coalesc"):
		counter = "coalesced"
	case strings.Contains(normalized, "complete"):
		counter = "completed"
	case strings.Contains(normalized, "start"):
		counter = "started"
	case strings.Contains(normalized, "cancel"):
		genericCancellation[typeName] = memberName
	}
	if counter == "" {
		return
	}
	if surfaces[typeName] == nil {
		surfaces[typeName] = make(map[string]string)
	}
	surfaces[typeName][counter] = memberName
}
