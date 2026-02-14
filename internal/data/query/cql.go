package query

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	cqlSelectRE       = regexp.MustCompile(`(?i)^\s*SELECT\s+modules(?:\s+WHERE\s+(.+))?\s*$`)
	cqlAndSplitRE     = regexp.MustCompile(`(?i)\s+AND\s+`)
	cqlNumericCondRE  = regexp.MustCompile(`(?i)^\s*([a-z_]+)\s*(>=|<=|!=|=|>|<)\s*(-?[0-9]+)\s*$`)
	cqlContainsCondRE = regexp.MustCompile(`(?i)^\s*([a-z_]+)\s+CONTAINS\s+['"]([^'"]+)['"]\s*$`)
	cqlStringCondRE   = regexp.MustCompile(`(?i)^\s*([a-z_]+)\s*(=|!=)\s*['"]([^'"]+)['"]\s*$`)
)

type CQLQuery struct {
	Target     string
	Conditions []CQLCondition
}

type CQLCondition struct {
	Field  string
	Op     string
	IntVal int
	StrVal string
	IsInt  bool
	IsStr  bool
}

func ParseCQL(raw string) (CQLQuery, error) {
	matches := cqlSelectRE.FindStringSubmatch(strings.TrimSpace(raw))
	if len(matches) == 0 {
		return CQLQuery{}, fmt.Errorf("invalid CQL query: expected SELECT modules [WHERE ...]")
	}

	query := CQLQuery{Target: "modules"}
	where := strings.TrimSpace(matches[1])
	if where == "" {
		return query, nil
	}

	parts := cqlAndSplitRE.Split(where, -1)
	query.Conditions = make([]CQLCondition, 0, len(parts))
	for _, part := range parts {
		condition, err := parseCQLCondition(part)
		if err != nil {
			return CQLQuery{}, err
		}
		query.Conditions = append(query.Conditions, condition)
	}
	return query, nil
}

func parseCQLCondition(raw string) (CQLCondition, error) {
	if match := cqlNumericCondRE.FindStringSubmatch(raw); len(match) == 4 {
		value, err := parseInt(match[3])
		if err != nil {
			return CQLCondition{}, fmt.Errorf("invalid numeric value %q: %w", match[3], err)
		}
		return CQLCondition{
			Field:  strings.ToLower(strings.TrimSpace(match[1])),
			Op:     strings.TrimSpace(match[2]),
			IntVal: value,
			IsInt:  true,
		}, nil
	}

	if match := cqlContainsCondRE.FindStringSubmatch(raw); len(match) == 3 {
		return CQLCondition{
			Field:  strings.ToLower(strings.TrimSpace(match[1])),
			Op:     "contains",
			StrVal: strings.TrimSpace(match[2]),
			IsStr:  true,
		}, nil
	}

	if match := cqlStringCondRE.FindStringSubmatch(raw); len(match) == 4 {
		return CQLCondition{
			Field:  strings.ToLower(strings.TrimSpace(match[1])),
			Op:     strings.TrimSpace(match[2]),
			StrVal: strings.TrimSpace(match[3]),
			IsStr:  true,
		}, nil
	}

	return CQLCondition{}, fmt.Errorf("invalid CQL condition %q", strings.TrimSpace(raw))
}

func parseInt(raw string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(raw))
}
