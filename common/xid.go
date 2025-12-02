package common

import (
	"fmt"
	"strings"
)

type Xid struct {
	Type       int // one of ENTITY_XXX constants below
	IsEntity   bool
	HasDetails bool

	Group      string
	GroupID    string
	Resource   string
	ResourceID string
	Version    string // "versions" or "meta"
	VersionID  string
}

type XidType struct {
	Type int // one of ENTITY_XXX constants below

	Group    string
	Resource string
	Version  string // "versions" or "meta"
}

const (
	ENTITY_REGISTRY = iota
	ENTITY_GROUP
	ENTITY_RESOURCE
	ENTITY_META
	ENTITY_VERSION
	ENTITY_MODEL

	ENTITY_GROUP_TYPE
	ENTITY_RESOURCE_TYPE
	ENTITY_VERSION_TYPE
)

func ParseXidType(xidTypeStr string) (*XidType, error) {
	xidTypeStr = strings.TrimSpace(xidTypeStr)
	if xidTypeStr == "" {
		return nil, fmt.Errorf("can't be an empty string")
	}

	if xidTypeStr[0] != '/' {
		return nil, fmt.Errorf("%q must start with /", xidTypeStr)
	}

	// xidTypeStr = strings.TrimLeft(xidTypeStr, "/")

	parts := strings.Split(xidTypeStr[1:], "/")

	if len(parts) == 1 && parts[0] == "" {
		parts = []string{}
	}

	for i, str := range parts {
		if str == "" {
			return nil, fmt.Errorf("%q has an empty part at position %d",
				xidTypeStr, i+1)
		}
	}

	xidType := &XidType{
		Type: ENTITY_REGISTRY,
	}

	if len(parts) > 0 {
		xidType.Group = parts[0]
		if xidType.Group != "" {
			xidType.Type = ENTITY_GROUP_TYPE
		}
		if len(parts) > 1 {
			xidType.Resource = parts[1]
			if xidType.Resource != "" {
				xidType.Type = ENTITY_RESOURCE_TYPE
			}
			if len(parts) > 2 {
				xidType.Version = parts[2]
				if xidType.Version == "versions" {
					xidType.Type = ENTITY_VERSION_TYPE
				} else if xidType.Version == "meta" {
					xidType.Type = ENTITY_META
				} else {
					return nil, fmt.Errorf("%q has %q at position 3, "+
						"needs to be either \"versions\" or \"meta\"",
						xidTypeStr, parts[2])
				}
				if len(parts) > 3 {
					return nil, fmt.Errorf("XIDType is too long")
				}
			}
		}
	}
	return xidType, nil
}

func ParseXid(xidStr string) (*Xid, error) {
	hasDetails := false
	xidStr = strings.TrimSpace(xidStr)
	if xidStr == "" {
		return nil, fmt.Errorf("can't be an empty string")
	}

	if xidStr[0] != '/' {
		return nil, fmt.Errorf("%q must start with /", xidStr)
	}

	if strings.HasSuffix(xidStr, "$details") {
		hasDetails = true
		xidStr = xidStr[:len(xidStr)-8]
	}

	parts := strings.Split(xidStr[1:], "/")

	if len(parts) == 1 && parts[0] == "" {
		parts = []string{}
	}

	for i, str := range parts {
		if str == "" {
			return nil, fmt.Errorf("%q has an empty part at position %d",
				xidStr, i+1)
		}
	}

	xid := &Xid{
		Type:       ENTITY_REGISTRY,
		IsEntity:   true,
		HasDetails: hasDetails,
	}

	if len(parts) > 0 {
		xid.Group = parts[0]
		if xid.Group != "" {
			xid.Type = ENTITY_GROUP_TYPE
			xid.IsEntity = false
		}
		if len(parts) > 1 {
			xid.GroupID = parts[1]
			if xid.GroupID != "" {
				xid.Type = ENTITY_GROUP
				xid.IsEntity = true
			}
			if len(parts) > 2 {
				xid.Resource = parts[2]
				if xid.Resource != "" {
					xid.Type = ENTITY_RESOURCE_TYPE
					xid.IsEntity = false
				}
				if len(parts) > 3 {
					xid.ResourceID = parts[3]
					if xid.ResourceID != "" {
						xid.Type = ENTITY_RESOURCE
						xid.IsEntity = true
					}
					if len(parts) > 4 {
						xid.Version = parts[4]
						if xid.Version == "versions" {
							xid.Type = ENTITY_VERSION_TYPE
							xid.IsEntity = false

							if len(parts) > 5 {
								xid.VersionID = parts[5]
								if xid.VersionID != "" {
									xid.Type = ENTITY_VERSION
									xid.IsEntity = true
								}
							}
							if len(parts) > 6 {
								return nil, fmt.Errorf("XID is too long")
							}
						} else if xid.Version == "meta" {
							xid.Type = ENTITY_META
							xid.IsEntity = false
							if len(parts) > 5 {
								return nil, fmt.Errorf("XID is too long")
							}
						} else {
							return nil, fmt.Errorf("%q has %q at position 5, "+
								"needs to be either \"versions\" or \"meta\"",
								xidStr, parts[4])
						}

					}
				}
			}
		}
	}
	return xid, nil
}

func ParseXref(xidStr string) (*Xid, error) {
	hasDetails := false
	xidStr = strings.TrimSpace(xidStr)
	if xidStr == "" {
		return nil, fmt.Errorf("can't be an empty string")
	}

	if xidStr[0] != '/' {
		return nil, fmt.Errorf("%q must start with /", xidStr)
	}

	if strings.HasSuffix(xidStr, "$details") {
		hasDetails = true
		xidStr = xidStr[:len(xidStr)-8]
	}

	parts := strings.Split(xidStr[1:], "/")

	if len(parts) == 1 && parts[0] == "" {
		parts = []string{}
	}

	for i, str := range parts {
		if str == "" {
			return nil, fmt.Errorf("%q has an empty part at position %d",
				xidStr, i+1)
		}
	}

	if len(parts) != 4 {
		return nil, fmt.Errorf("%q must be of the form: "+
			"/GROUPS/GID/RESOURCES/RID", xidStr)
	}

	xid := &Xid{
		Type:       ENTITY_REGISTRY,
		IsEntity:   true,
		HasDetails: hasDetails,
	}

	if len(parts) > 0 {
		xid.Group = parts[0]
		if xid.Group != "" {
			xid.Type = ENTITY_GROUP_TYPE
			xid.IsEntity = false
		}
		if len(parts) > 1 {
			xid.GroupID = parts[1]
			if xid.GroupID != "" {
				xid.Type = ENTITY_GROUP
				xid.IsEntity = true
			}
			if len(parts) > 2 {
				xid.Resource = parts[2]
				if xid.Resource != "" {
					xid.Type = ENTITY_RESOURCE_TYPE
					xid.IsEntity = false
				}
				if len(parts) > 3 {
					xid.ResourceID = parts[3]
					if xid.ResourceID != "" {
						xid.Type = ENTITY_RESOURCE
						xid.IsEntity = true
					}
					if len(parts) > 4 {
						xid.Version = parts[4]
						if xid.Version == "versions" {
							xid.Type = ENTITY_VERSION_TYPE
							xid.IsEntity = false

							if len(parts) > 5 {
								xid.VersionID = parts[5]
								if xid.VersionID != "" {
									xid.Type = ENTITY_VERSION
									xid.IsEntity = true
								}
							}
							if len(parts) > 6 {
								return nil, fmt.Errorf("%q is too long", xidStr)
							}
						} else if xid.Version == "meta" {
							xid.Type = ENTITY_META
							xid.IsEntity = false
							if len(parts) > 5 {
								return nil, fmt.Errorf("%q is too long", xidStr)
							}
						} else {
							return nil, fmt.Errorf("%q references an unknown "+
								"entity %q", xidStr, xid.Version)
						}

					}
				}
			}
		}
	}
	return xid, nil
}

func (xid *Xid) String() string {
	str := "/"

	if xid.Group != "" {
		str += xid.Group
		if xid.GroupID != "" {
			str += "/" + xid.GroupID
			if xid.Resource != "" {
				str += "/" + xid.Resource
				if xid.ResourceID != "" {
					str += "/" + xid.ResourceID
					if xid.Version != "" {
						str += "/" + xid.Version
						if xid.VersionID != "" {
							str += "/" + xid.VersionID
						}
					}
				}
			}
		}
	}

	if xid.HasDetails {
		str += "$details"
	}

	return str
}

func (xid *Xid) ToAbstract() string {
	res := "/"
	if xid.Group != "" {
		res += xid.Group
		if xid.Resource != "" {
			res += "/" + xid.Resource
			if xid.Version != "" {
				res += "/" + xid.Version
			}
		}
	}
	return res
}

func Xid2Abstract(str string) (string, error) {
	// GROUPS/gid/RESOURCES/rid/versions/vID
	xid, err := ParseXid(str)
	if err != nil {
		return "", err
	}
	return xid.ToAbstract(), nil
}

func (xid *Xid) AddPath(str string) (*Xid, error) {
	if str == "" {
		return xid, nil
	}

	xidStr := xid.String()
	if xidStr[len(xidStr)-1] == '/' {
		return ParseXid(xidStr + str)
	}

	return ParseXid(xidStr + "/" + str)
}
