package memory

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func toStr(v any) string { return fmt.Sprintf("%v", v) }

type iterItem struct {
	ent   *entRow
	props []*propRow
	mask  uint64
}

type memIter struct {
	be     *MemoryBackend
	tx     *registry.Tx
	mode   int
	items  []iterItem
	pos    int
	size   uint64
	total  uint64
	offset uint64
}

func (it *memIter) Next() (*registry.Entity, *XRError) {
	if it.pos >= len(it.items) {
		return nil, nil
	}
	item := it.items[it.pos]
	it.pos++

	e, xErr := it.be.buildEntity(it.tx, item.ent, item.props, it.mode,
		item.mask)
	if xErr != nil {
		return nil, xErr
	}
	if it.mode == FOR_WRITE {
		if xErr := registry.LockEntityFamily(it.tx, e); xErr != nil {
			return nil, xErr
		}
	}
	it.offset++
	return e, nil
}

func (it *memIter) Close()         {}
func (it *memIter) Offset() uint64 { return it.offset }
func (it *memIter) Size() uint64   { return it.size }
func (it *memIter) Total() uint64  { return it.total }

// Convert each non-escaped * into SQL % for LIKE queries
func likeWildcardIt(str string) (string, bool) {
	wild := false
	res := strings.Builder{}
	prev := '\000'
	for _, ch := range str {
		if ch == '*' && prev != '\\' {
			res.WriteRune('%')
			wild = true
		} else {
			res.WriteRune(ch)
		}
		prev = ch
	}
	return res.String(), wild
}

// SQL LIKE (case-insensitive) -> regexp
func likeRegexp(pattern string) *regexp.Regexp {
	res := strings.Builder{}
	res.WriteString("(?is)^")
	rs := []rune(pattern)
	for i := 0; i < len(rs); i++ {
		switch ch := rs[i]; ch {
		case '%':
			res.WriteString(".*")
		case '_':
			res.WriteString(".")
		case '\\':
			if i+1 < len(rs) {
				i++
				res.WriteString(regexp.QuoteMeta(string(rs[i])))
			} else {
				res.WriteString(regexp.QuoteMeta("\\"))
			}
		default:
			res.WriteString(regexp.QuoteMeta(string(ch)))
		}
	}
	res.WriteString("$")
	return regexp.MustCompile(res.String())
}

// CAST(str AS DECIMAL): leading numeric prefix, rounded to an integer
func castDecimal(s string) float64 {
	s = strings.TrimSpace(s)
	end := 0
	seenDot, seenExp, seenDigit := false, false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			seenDigit = true
			end = i + 1
		case (c == '+' || c == '-') && (i == 0 || s[i-1] == 'e' || s[i-1] == 'E'):
		case c == '.' && !seenDot && !seenExp:
			seenDot = true
		case (c == 'e' || c == 'E') && seenDigit && !seenExp:
			seenExp = true
		default:
			i = len(s)
		}
	}
	if end == 0 {
		return 0
	}
	f, err := strconv.ParseFloat(s[:end], 64)
	if err != nil {
		return 0
	}
	return math.Round(f)
}

func isNumType(t string) bool {
	return t == INTEGER || t == DECIMAL || t == UINTEGER
}

func (tx *memTx) filterNameMatcher(f *registry.FilterExpr) func(string) bool {
	present := f.Operator == FILTER_PRESENT || f.Operator == FILTER_ABSENT
	if f.PP.HasWild {
		var pat string
		if present {
			pat, _ = f.PP.DBFilterExists()
		} else {
			pat, _ = f.PP.DBFilter()
		}
		re := regexp.MustCompile("(?i)" + pat)
		return re.MatchString
	}
	if present {
		re := likeRegexp(f.PropName + "%")
		return re.MatchString
	}
	return func(n string) bool { return strings.EqualFold(n, f.PropName) }
}

func valueMatcher(value string) func(p *propRow) bool {
	pat, wild := likeWildcardIt(value)
	if !wild {
		return func(p *propRow) bool {
			if p.PropType == STRING {
				return strings.EqualFold(p.Value, pat)
			}
			return p.Value == pat
		}
	}
	re := likeRegexp(pat)
	return func(p *propRow) bool {
		if p.PropType != STRING {
			return p.Value == pat
		}
		return re.MatchString(p.Value)
	}
}

func cmpResult(op int, c int) bool {
	switch op {
	case FILTER_LESS:
		return c < 0
	case FILTER_LESS_EQUAL:
		return c <= 0
	case FILTER_GREATER:
		return c > 0
	}
	return c >= 0
}

func (tx *memTx) filterResults(regSID string, f *registry.FilterExpr) []*entRow {
	db := tx.db
	nameOK := tx.filterNameMatcher(f)
	ents := tx.entsOfReg(regSID)
	res := []*entRow{}

	switch f.Operator {
	case FILTER_PRESENT:
		for _, e := range ents {
			if e.Abstract != f.Abstract {
				continue
			}
			for _, p := range db.props[e.SID] {
				if nameOK(p.Name) {
					res = append(res, e)
					break
				}
			}
		}

	case FILTER_ABSENT:
		for _, e := range ents {
			if e.Abstract != f.Abstract {
				continue
			}
			found := false
			for _, p := range db.props[e.SID] {
				if nameOK(p.Name) {
					found = true
					break
				}
			}
			if !found {
				res = append(res, e)
			}
		}

	case FILTER_EQUAL, FILTER_LESS, FILTER_LESS_EQUAL, FILTER_GREATER,
		FILTER_GREATER_EQUAL, FILTER_NOT_EQUAL:

		var valOK func(p *propRow) bool
		if f.Operator == FILTER_EQUAL || f.Operator == FILTER_NOT_EQUAL {
			valOK = valueMatcher(f.Value)
		} else {
			want := castDecimal(f.Value)
			lwant := strings.ToLower(f.Value)
			op := f.Operator
			valOK = func(p *propRow) bool {
				if isNumType(p.PropType) {
					c := 0
					if got := castDecimal(p.Value); got < want {
						c = -1
					} else if got > want {
						c = 1
					}
					return cmpResult(op, c)
				}
				return cmpResult(op, strings.Compare(strings.ToLower(p.Value),
					lwant))
			}
		}

		for _, e := range ents {
			if e.Abstract != f.Abstract && f.Operator == FILTER_NOT_EQUAL {
				continue
			}
			matches := 0
			for _, p := range db.props[e.SID] {
				if p.Abstract == f.Abstract && nameOK(p.Name) && valOK(p) {
					matches++
				}
			}
			if f.Operator == FILTER_NOT_EQUAL {
				if matches == 0 {
					res = append(res, e)
				}
				continue
			}
			for i := 0; i < matches; i++ {
				res = append(res, e)
			}
		}

	default:
		PanicIf(true, "Bad filter.op: %#v", f)
	}

	sortByXID(res)
	return res
}

func (tx *memTx) filterMatches(regSID string, filters [][]*registry.FilterExpr) map[string]uint64 {
	db := tx.db
	ents := tx.entsOfReg(regSID)

	hasChild := map[string]bool{}
	for _, e := range ents {
		hasChild[e.ParentSID] = true
	}
	leaves := []*entRow{}
	for _, e := range ents {
		if !hasChild[e.SID] {
			leaves = append(leaves, e)
		}
	}
	sortByXID(leaves)

	covered := func(r *entRow, fn func(l *entRow)) {
		if r.Type == ENTITY_META {
			if e := db.ents[r.ParentSID]; e != nil && !hasChild[e.SID] {
				fn(e)
			}
			for _, c := range tx.childEnts(r.ParentSID) {
				if !hasChild[c.SID] {
					fn(c)
				}
			}
			return
		}
		if r.XID == "/" {
			for _, l := range leaves {
				fn(l)
			}
			return
		}
		prefix := r.XID + "/"
		i := sort.Search(len(leaves), func(i int) bool {
			return leaves[i].XID >= r.XID
		})
		for ; i < len(leaves); i++ {
			l := leaves[i]
			if !strings.HasPrefix(l.XID, r.XID) {
				break
			}
			if l.XID == r.XID || strings.HasPrefix(l.XID, prefix) {
				fn(l)
			}
		}
	}

	masks := map[string]uint64{}
	for orIndex, ands := range filters {
		cnt := map[string]int{}
		for _, f := range ands {
			for _, r := range tx.filterResults(regSID, f) {
				covered(r, func(l *entRow) { cnt[l.SID]++ })
			}
		}
		for sid, c := range cnt {
			if c == len(ands) {
				masks[sid] |= uint64(1) << uint(orIndex)
			}
		}
	}

	work := make([]string, 0, len(masks))
	for sid := range masks {
		work = append(work, sid)
	}
	for len(work) > 0 {
		sid := work[len(work)-1]
		work = work[:len(work)-1]
		e := db.ents[sid]
		if e == nil {
			continue
		}
		mask := masks[sid]
		push := func(target string) {
			if masks[target]|mask != masks[target] {
				masks[target] |= mask
				work = append(work, target)
			}
		}
		if db.ents[e.ParentSID] != nil {
			push(e.ParentSID)
		}
		if e.Type == ENTITY_RESOURCE {
			for _, c := range tx.childEnts(e.SID) {
				if c.Type == ENTITY_META {
					push(c.SID)
				}
			}
		}
	}
	return masks
}

func topLevelSlashCount(xid string) int {
	switch strings.Count(xid, "/") {
	case 1:
		return 3
	case 3:
		return 5
	}
	return 7
}

func substringIndex(s string, n int) string {
	idx := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			idx++
			if idx == n {
				return s[:i]
			}
		}
	}
	return s
}

type sortRow struct {
	item    iterItem
	lxid    string
	topXID  string
	nonNull bool
	val     string
}

func (be *MemoryBackend) GenerateResultEntities(reg *registry.Registry, what string, XIDs []string) (registry.EntityIterator, *XRError) {
	if log.HasVerbose("Query") {
		// Just because our testing expect something to be printed
		log.Printf("tx: %s Query: \nQuerying...", reg.Tx.UUID)
	}

	tx, xErr := mt(reg.Tx)
	if xErr != nil {
		return nil, xErr
	}
	db := tx.db

	info := reg.GetRequestInfo()
	filters := info.Filters
	docView := info.DoDocView()
	sortKey := info.SortKey
	limit := info.Limit
	offset := info.Offset

	desc := false
	if sortKey != "" && sortKey[0] == '-' {
		desc = true
		sortKey = sortKey[1:]
	}

	if len(filters) > 64 {
		return nil, NewXRError("bad_filter",
			info.OriginalRequest.URL.RequestURI(),
			"value=filter",
			"error_detail=too many OR'd filter expressions "+
				fmt.Sprintf("(%d), the max supported is 64", len(filters)))
	}

	var matches map[string]uint64
	if len(filters) != 0 {
		matches = tx.filterMatches(reg.SID, filters)
	}

	slash := 7
	if len(XIDs) > 0 {
		slash = topLevelSlashCount(XIDs[0])
	}
	lsortKey := strings.ToLower(sortKey)

	rows := []*sortRow{}
	for _, e := range tx.entsOfReg(reg.SID) {
		mask := uint64(0)
		if len(filters) != 0 {
			m, ok := matches[e.SID]
			if !ok {
				continue
			}
			mask = m
		}

		if what != "Registry" && len(XIDs) > 0 {
			ok := false
			for _, p := range XIDs {
				if e.XID == p || strings.HasPrefix(e.XID, p+"/") {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}

		props := []*propRow{}
		for _, p := range tx.propsOf(e.SID) {
			if !docView || p.DocView {
				props = append(props, p)
			}
		}
		if len(props) == 0 {
			continue
		}

		row := &sortRow{
			item: iterItem{ent: e, props: props, mask: mask},
			lxid: e.LowerXID,
		}
		row.topXID = substringIndex(e.XID, slash)

		if sortKey != "" {
			if te := db.entByLXID[lxidKey(reg.SID,
				strings.ToLower(row.topXID))]; te != nil && te.XID == row.topXID {
				if sp := db.props[te.SID][lsortKey]; sp != nil {
					row.nonNull = true
					if isNumType(sp.PropType) {
						row.val = strconv.FormatFloat(castDecimal(sp.Value),
							'f', -1, 64)
					} else {
						row.val = sp.Value
					}
					row.val = strings.ToLower(row.val)
				}
			}
		}
		rows = append(rows, row)
	}

	cmpSort := func(a, b *sortRow) int {
		if sortKey == "" {
			return 0
		}
		c := 0
		if a.nonNull != b.nonNull {
			if !a.nonNull {
				c = -1
			} else {
				c = 1
			}
		} else {
			c = strings.Compare(a.val, b.val)
		}
		if desc {
			c = -c
		}
		return c
	}

	total := uint64(math.MaxUint64)
	if limit == 0 {
		sort.SliceStable(rows, func(i, j int) bool {
			if c := cmpSort(rows[i], rows[j]); c != 0 {
				return c < 0
			}
			return rows[i].lxid < rows[j].lxid
		})
	} else {
		sort.SliceStable(rows, func(i, j int) bool {
			if c := cmpSort(rows[i], rows[j]); c != 0 {
				return c < 0
			}
			return rows[i].topXID < rows[j].topXID
		})

		topIdx := make([]uint64, len(rows))
		rank := uint64(0)
		for i, r := range rows {
			if i == 0 || cmpSort(rows[i-1], r) != 0 ||
				rows[i-1].topXID != r.topXID {
				rank++
			}
			topIdx[i] = rank
		}

		idx := make([]int, 0, len(rows))
		for i := range rows {
			if topIdx[i] >= offset+1 && topIdx[i] <= offset+limit {
				idx = append(idx, i)
			}
		}
		sort.SliceStable(idx, func(a, b int) bool {
			if topIdx[idx[a]] != topIdx[idx[b]] {
				return topIdx[idx[a]] < topIdx[idx[b]]
			}
			return rows[idx[a]].lxid < rows[idx[b]].lxid
		})

		page := make([]*sortRow, len(idx))
		for i, ix := range idx {
			page[i] = rows[ix]
		}
		if len(page) > 0 {
			total = rank
		}
		rows = page
	}

	items := make([]iterItem, len(rows))
	for i, r := range rows {
		items[i] = r.item
	}

	log.VPrintf(3, "tx: %s memory query returned %d entities", reg.Tx.UUID,
		len(items))

	return &memIter{be: be, tx: reg.Tx, mode: FOR_READ, items: items,
		size: uint64(len(items)), total: total}, nil
}
