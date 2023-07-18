-- DROP DATABASE IF EXISTS registry ;
-- CREATE DATABASE registry ;
-- USE registry ;
-- ^^ OLD STUF

/*
Notes:
SID -> System generated ID, usually primary key. Assumed to be globally
unique so that when searching we don't need to combine it with any other
scoping column to provide uniqueness
UID -> User provided ID. Only needs to be unique within its parent scope.
This is why we use SID for cross-table joins/links.
The code doesn't do delete propagation. Instead, the code will delete
whichever resource was asked to be deleted and then the DB triggers
will delete all necessarily (related) rows/resources as needed. So,
deleting a row from the "Registry" table should delete ALL other resources
in all other tables automatically.
The "Props" table holds all properties for all entities rather than
having property specific columns in the appropriate tables. No idea which
is easier/faster but having it all in one table made things a lot easier
for filtering/searching. But we can switch it if needed at some point. This
also means that all properties (including extensions) are processed the
same way... via the generic Get/Set methods.
*/


SET GLOBAL sql_mode = 'ANSI_QUOTES' ;

CREATE TABLE Registries (
    SID     VARCHAR(255) NOT NULL,  # System ID
    UID     VARCHAR(255) NOT NULL,  # User defined

    PRIMARY KEY (SID),
    UNIQUE INDEX (UID)
);

CREATE TRIGGER RegistryTrigger BEFORE DELETE ON Registries
FOR EACH ROW
BEGIN
    DELETE FROM Props WHERE EntitySID=OLD.SID @
    DELETE FROM "Groups" WHERE RegistrySID=OLD.SID @
    DELETE FROM ModelEntities WHERE RegistrySID=OLD.SID @
END ;

CREATE TABLE ModelEntities (        # Group or Resource (no parent->Group)
    SID         VARCHAR(64),        # my System ID
    RegistrySID VARCHAR(64),
    ParentSID   VARCHAR(64),        # ID of parent ModelEntity

    Plural      VARCHAR(64),
    Singular    VARCHAR(64),
    SchemaURL   VARCHAR(255),       # For Groups
    Versions    INT NOT NULL,       # For Resources
    VersionId   BOOL NOT NULL,      # For Resources
    Latest      BOOL NOT NULL,      # For Resources

    PRIMARY KEY(SID),
    UNIQUE INDEX (RegistrySID, ParentSID, Plural),
    CONSTRAINT UC_Singular UNIQUE (RegistrySID, ParentSID, Singular)
);

CREATE TRIGGER ModelTrigger BEFORE DELETE ON ModelEntities
FOR EACH ROW
BEGIN
    DELETE FROM "Groups" WHERE ModelSID=OLD.SID @
    DELETE FROM Resources WHERE ModelSID=OLD.SID @
END ;


CREATE TABLE "Groups" (
    SID             VARCHAR(64) NOT NULL,   # System ID
    UID             VARCHAR(64) NOT NULL,   # User defined
    RegistrySID     VARCHAR(64) NOT NULL,
    ModelSID        VARCHAR(64) NOT NULL,
    Path            VARCHAR(255) NOT NULL,
    Abstract        VARCHAR(255) NOT NULL,

    PRIMARY KEY (SID),
    INDEX(RegistrySID, UID),
    UNIQUE INDEX (RegistrySID, ModelSID, UID)
);

CREATE TRIGGER GroupTrigger BEFORE DELETE ON "Groups"
FOR EACH ROW
BEGIN
    DELETE FROM Props WHERE EntitySID=OLD.SID @
    DELETE FROM Resources WHERE GroupSID=OLD.SID @
END ;

CREATE TABLE Resources (
    SID             VARCHAR(64) NOT NULL,   # System ID
    UID             VARCHAR(64) NOT NULL,   # User defined
    GroupSID        VARCHAR(64) NOT NULL,   # System ID
    ModelSID        VARCHAR(64) NOT NULL,
    Path            VARCHAR(255) NOT NULL,
    Abstract        VARCHAR(255) NOT NULL,

    PRIMARY KEY (SID),
    INDEX(GroupSID, UID),
    UNIQUE INDEX (GroupSID, ModelSID, UID)
);

CREATE TRIGGER ResourcesTrigger BEFORE DELETE ON Resources
FOR EACH ROW
BEGIN
    DELETE FROM Props WHERE EntitySID=OLD.SID @
    DELETE FROM Versions WHERE ResourceSID=OLD.SID @
END ;

CREATE TABLE Versions (
    SID                 VARCHAR(64) NOT NULL,   # System ID
    UID                 VARCHAR(64) NOT NULL,   # User defined
    ResourceSID         VARCHAR(64) NOT NULL,   # System ID
    Path                VARCHAR(255) NOT NULL,
    Abstract            VARCHAR(255) NOT NULL,
    Counter             SERIAL,                 # Counter, auto-increments

    ResourceURL         VARCHAR(255),
    ResourceProxyURL    VARCHAR(255),
    ResourceContentSID  VARCHAR(64),

    PRIMARY KEY (SID),
    INDEX (ResourceSID, UID),
    UNIQUE INDEX (ResourceSID, UID)
);

CREATE TRIGGER VersionsTrigger BEFORE DELETE ON Versions
FOR EACH ROW
BEGIN
    DELETE FROM Props WHERE EntitySID=OLD.SID @
    DELETE FROM ResourceContents WHERE VersionSID=OLD.SID @
END ;

CREATE TABLE Props (
    RegistrySID VARCHAR(64) NOT NULL,
    EntitySID   VARCHAR(64) NOT NULL,       # Reg,Group,Res,Ver System ID
    PropName    VARCHAR(64) NOT NULL,
    PropValue   VARCHAR(255),
    PropType    CHAR(1) NOT NULL,           # i(nt), f(loat), b(ool), s(tring)

    PRIMARY KEY (EntitySID, PropName),
    INDEX (EntitySID)
);

CREATE TABLE ResourceContents (
    VersionSID      VARCHAR(255),
    Content         MEDIUMBLOB,

    PRIMARY KEY (VersionSID)
);

CREATE VIEW LatestProps AS
SELECT
    p.RegistrySID,
    r.SID AS EntitySID,
    p.PropName,
    p.PropValue,
    p.PropType
FROM Props AS p
JOIN Versions AS v ON (p.EntitySID=v.SID)
JOIN Resources AS r ON (r.SID=v.ResourceSID)
JOIN Props AS p1 ON (p1.EntitySID=r.SID)
WHERE p1.PropName='LatestId' AND v.UID=p1.PropValue AND
      p.PropName<>'id';     # Don't overwrite 'id'

CREATE VIEW AllProps AS
SELECT * FROM Props
UNION SELECT * FROM LatestProps ;


CREATE VIEW Entities AS
SELECT                          # Gather Registries
    r.SID AS RegSID,
    0 AS Level,
    'registries' AS Plural,
    NULL AS ParentSID,
    r.SID AS eSID,
    r.UID AS UID,
    '' AS Abstract,
    '' AS Path
FROM Registries AS r

UNION SELECT                            # Gather Groups
    g.RegistrySID AS RegSID,
    1 AS Level,
    m.Plural AS Plural,
    g.RegistrySID AS ParentSID,
    g.SID AS eSID,
    g.UID AS UID,
    g.Abstract,
    g.Path
FROM "Groups" AS g
JOIN ModelEntities AS m ON (m.SID=g.ModelSID)

UNION SELECT                    # Add Resources
    m.RegistrySID AS RegSID,
    2 AS Level,
    m.Plural AS Plural,
    r.GroupSID AS ParentSID,
    r.SID AS eSID,
    r.UID AS UID,
    r.Abstract,
    r.Path
FROM Resources AS r
JOIN ModelEntities AS m ON (m.SID=r.ModelSID)

UNION SELECT                    # Add Versions
    rm.RegistrySID AS RegSID,
    3 AS Level,
    'versions' AS Plural,
    r.SID AS ParentSID,
    v.SID AS eSID,
    v.UID AS UID,
    v.Abstract,
    v.Path
FROM Versions AS v
JOIN Resources AS r ON (r.SID=v.ResourceSID)
JOIN ModelEntities AS rm ON (rm.SID=r.ModelSID) ;

CREATE VIEW FullTree AS
SELECT
    RegSID,
    Level,
    Plural,
    ParentSID,
    eSID,
    UID,
    Path,
    PropName,
    PropValue,
    PropType,
    Abstract
FROM Entities
LEFT JOIN AllProps ON (AllProps.EntitySID=Entities.eSID)
ORDER by Path, PropName;

CREATE VIEW Leaves AS
SELECT eSID FROM Entities
WHERE eSID NOT IN (
    SELECT DISTINCT ParentSID FROM Entities WHERE ParentSID IS NOT NULL
);

