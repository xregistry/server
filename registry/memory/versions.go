package memory

import (
	"sort"
	"strings"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func lowerLess(a, b string) bool {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	return la < lb || (la == lb && a < b)
}

func (be *MemoryBackend) findBadByTime(r *registry.Resource, modified bool) ([]*registry.AncestorVersion, *XRError) {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return nil, xErr
	}

	vers := tx.versionsOf(r.SID)
	ts := func(v *verRow) string {
		if modified {
			return v.ModifiedAt
		}
		return v.CreatedAt
	}
	sort.Slice(vers, func(i, j int) bool {
		if ts(vers[i]) != ts(vers[j]) {
			return ts(vers[i]) < ts(vers[j])
		}
		return lowerLess(vers[i].UID, vers[j].UID)
	})

	res := []*registry.AncestorVersion{}
	for i, v := range vers {
		expected := v.UID
		if i > 0 {
			expected = vers[i-1].UID
		}
		if v.AncestorID != expected {
			res = append(res, &registry.AncestorVersion{
				VersionUID: v.UID, AncestorID: expected})
		}
	}
	return res, nil
}

func (be *MemoryBackend) FindBadAncestorsCreatedAt(r *registry.Resource, lock bool) ([]*registry.AncestorVersion, *XRError) {
	return be.findBadByTime(r, false)
}

func (be *MemoryBackend) FindBadAncestorsModifiedAt(r *registry.Resource, lock bool) ([]*registry.AncestorVersion, *XRError) {
	return be.findBadByTime(r, true)
}

func (be *MemoryBackend) FindBadAncestorsSemVer(r *registry.Resource, lock bool) ([]*registry.AncestorVersion, *XRError) {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return nil, xErr
	}

	vers := tx.versionsOf(r.SID)
	sort.Slice(vers, func(i, j int) bool {
		if c := registry.CompareSemver(vers[i].UID, vers[j].UID); c != 0 {
			return c < 0
		}
		return strings.ToLower(vers[i].UID) < strings.ToLower(vers[j].UID)
	})

	res := []*registry.AncestorVersion{}
	for i, v := range vers {
		expected := v.UID
		if i > 0 {
			expected = vers[i-1].UID
		}
		if v.AncestorID != expected {
			res = append(res, &registry.AncestorVersion{
				VersionUID: v.UID, AncestorID: expected})
		}
	}
	return res, nil
}

func (be *MemoryBackend) FindNewestVersionID(r *registry.Resource, excludeTBD bool) (string, *XRError) {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return "", xErr
	}

	vers := tx.versionsOf(r.SID)
	cands := []*verRow{}
	for _, v := range vers {
		if excludeTBD && v.AncestorID == ANCESTORID_TBD {
			continue
		}
		hasChild := false
		for _, v2 := range vers {
			if v2.AncestorID == v.UID && v2.UID != v.UID {
				hasChild = true
				break
			}
		}
		if !hasChild {
			cands = append(cands, v)
		}
	}
	if len(cands) == 0 {
		return "", nil
	}

	sort.Slice(cands, func(i, j int) bool {
		if cands[i].CreatedAt != cands[j].CreatedAt {
			return cands[i].CreatedAt > cands[j].CreatedAt
		}
		return lowerLess(cands[j].UID, cands[i].UID)
	})
	return cands[0].UID, nil
}

func (be *MemoryBackend) versionAncestors(r *registry.Resource, pred func(v *verRow, all []*verRow) bool, byCreated bool) ([]*registry.VersionAncestor, *XRError) {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return nil, xErr
	}

	all := tx.versionsOf(r.SID)
	vers := []*verRow{}
	for _, v := range all {
		if pred(v, all) {
			vers = append(vers, v)
		}
	}
	sort.Slice(vers, func(i, j int) bool {
		if byCreated && vers[i].CreatedAt != vers[j].CreatedAt {
			return vers[i].CreatedAt < vers[j].CreatedAt
		}
		return lowerLess(vers[i].UID, vers[j].UID)
	})

	res := make([]*registry.VersionAncestor, len(vers))
	for i, v := range vers {
		res[i] = &registry.VersionAncestor{
			VID: v.UID, AncestorID: v.AncestorID, CreatedAt: v.CreatedAt}
	}
	return res, nil
}

func (be *MemoryBackend) GetVersionIDs(r *registry.Resource) ([]*registry.VersionAncestor, *XRError) {
	return be.versionAncestors(r,
		func(*verRow, []*verRow) bool { return true }, false)
}

func (be *MemoryBackend) GetProblematicVersions(r *registry.Resource) ([]*registry.VersionAncestor, *XRError) {
	return be.versionAncestors(r, func(v *verRow, all []*verRow) bool {
		if v.AncestorID == ANCESTORID_TBD {
			return true
		}
		if v.UID == v.AncestorID {
			return false
		}
		for _, o := range all {
			if o.UID == v.AncestorID {
				return false
			}
		}
		return true
	}, true)
}

func (be *MemoryBackend) GetRootVersionIDs(r *registry.Resource) ([]string, *XRError) {
	return be.versionIDsWhere(r, func(v *verRow) bool { return v.UID == v.AncestorID })
}

func (be *MemoryBackend) GetChildVersionIDs(r *registry.Resource, parentVID string) ([]string, *XRError) {
	return be.versionIDsWhere(r, func(v *verRow) bool { return v.AncestorID == parentVID })
}

func (be *MemoryBackend) versionIDsWhere(r *registry.Resource, pred func(*verRow) bool) ([]string, *XRError) {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return nil, xErr
	}

	res := []string{}
	for _, v := range tx.versionsOf(r.SID) {
		if pred(v) {
			res = append(res, v.UID)
		}
	}
	sort.Slice(res, func(i, j int) bool { return lowerLess(res[i], res[j]) })
	return res, nil
}

func (be *MemoryBackend) GetNumberOfVersions(r *registry.Resource) (int, *XRError) {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return 0, xErr
	}
	return len(tx.versionsOf(r.SID)), nil
}

// total # of distinct values, # non-empty, # empty (missing)
func (be *MemoryBackend) CountNonMatchingVersions(r *registry.Resource, vAttrPath *PropPath) (int, int, int, *XRError) {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return 0, 0, 0, xErr
	}

	name := strings.ToLower(vAttrPath.DB())
	values := map[string]bool{}
	nonEmpty := map[string]bool{}
	empty := 0
	for _, v := range tx.versionsOf(r.SID) {
		p := tx.db.props[v.SID][name]
		if p == nil {
			empty++
			values["\x00null"] = true
			continue
		}
		k := strings.ToLower(p.Value)
		values[k] = true
		nonEmpty[k] = true
	}
	return len(values), len(nonEmpty), empty, nil
}

func (be *MemoryBackend) versionsMatching(g *registry.Group, rm *registry.ResourceModel, regSID string, bad func(tx *memTx, res *resRow, v *entRow) bool) (string, []string, *XRError) {
	tx, xErr := mt(g.Tx)
	if xErr != nil {
		return "", nil, xErr
	}

	ress := []*resRow{}
	for _, r := range tx.db.resources {
		if r.GroupSID == g.SID && r.Plural == rm.Plural {
			ress = append(ress, r)
		}
	}
	sort.Slice(ress, func(i, j int) bool { return ress[i].XID < ress[j].XID })

	for _, res := range ress {
		vIDs := []string{}
		for _, e := range tx.childEnts(res.SID) {
			if e.Type == ENTITY_VERSION && bad(tx, res, e) {
				vIDs = append(vIDs, e.UID)
			}
		}
		if len(vIDs) > 0 {
			return res.XID, vIDs, nil
		}
	}
	return "", []string{}, nil
}

func (be *MemoryBackend) FindBadEqualsVersions(g *registry.Group, gPP *PropPath, rm *registry.ResourceModel, rPP *PropPath) (string, []string, *XRError) {
	tx, xErr := mt(g.Tx)
	if xErr != nil {
		return "", nil, xErr
	}

	gp := tx.db.props[g.SID][strings.ToLower(gPP.DB())]
	if gp == nil {
		return "", []string{}, nil
	}
	rName := strings.ToLower(rPP.DB())

	return be.versionsMatching(g, rm, g.Registry.SID,
		func(tx *memTx, res *resRow, v *entRow) bool {
			vp := tx.db.props[v.SID][rName]
			return vp == nil || !strings.EqualFold(vp.Value, gp.Value)
		})
}

func (be *MemoryBackend) FindBadEnumVersions(g *registry.Group, c *registry.Constraint, rm *registry.ResourceModel, rPP *PropPath) (string, []string, *XRError) {
	enum := map[string]bool{}
	for _, v := range c.Enum {
		enum[strings.ToLower(toStr(v))] = true
	}
	rName := strings.ToLower(rPP.DB())

	return be.versionsMatching(g, rm, g.Registry.SID,
		func(tx *memTx, res *resRow, v *entRow) bool {
			vp := tx.db.props[v.SID][rName]
			return vp != nil && !enum[strings.ToLower(vp.Value)]
		})
}
