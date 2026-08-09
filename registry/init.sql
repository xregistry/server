-- DROP DATABASE IF EXISTS registry ;
-- CREATE DATABASE registry ;
-- USE registry ;
-- ^^ OLD STUF

-- MySQL config requirements:
-- sql_mode:
--   ANSI_QUOTES        <- enabled
--   ONLY_FULL_GROUP_BY <- disabled

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
The "Entities"/"Props" tables hold all properties for all
entities rather than having property specific columns in the appropriate
tables. No idea which is easier/faster but having it all in one table
made things a lot easier for filtering/searching. But we can switch it
if needed at some point. This also means that all properties (including
extensions) are processed the same way... via the generic Get/Set
methods.
*/


SET GLOBAL sql_mode = 'ANSI_QUOTES' ;
SET sql_mode = 'ANSI_QUOTES' ;

CREATE TABLE Registries (
    SID     VARCHAR(255) NOT NULL,  # System ID
    UID     VARCHAR(255) NOT NULL,  # User defined

    # Internal fast-path flag: true if this Registry currently has AT
    # LEAST ONE xref set anywhere in it (Metas.xRefPath IS NOT NULL for
    # some row). NOT a model/user-visible attribute - it's a plain
    # column, never surfaced via Object/System/Props, so it's
    # never serialized in any response and never bumps the Registry's
    # own epoch/modifiedAt. Lets Resource.runCascade() skip the "am I
    # an xref target?" fan-out check entirely for Registries that never
    # use xref. Set to true synchronously in Go (runCascade()) the
    # moment a first xref is created, so it's correct even within the
    # same Tx that creates it. Cleared back to false lazily by the
    # triggers below whenever an xref is removed/its source Resource is
    # deleted and a rescan finds no xrefs remain - safe to be "stuck
    # true" a little longer than strictly needed (just a missed
    # optimization), never a correctness issue.
    UsesXref BOOL NOT NULL DEFAULT false,

    PRIMARY KEY (SID),
    UNIQUE INDEX (UID)
);

CREATE TRIGGER RegistryTrigger BEFORE DELETE ON Registries
FOR EACH ROW
BEGIN
    DELETE FROM "Groups" WHERE RegistrySID=OLD.SID $$
    DELETE FROM Models   WHERE RegistrySID=OLD.SID $$
    DELETE FROM Props WHERE RegSID=OLD.SID $$
    DELETE FROM Entities  WHERE RegSID=OLD.SID $$
END ;

CREATE TABLE Models (
    RegistrySID VARCHAR(64) NOT NULL,
    Model       JSON,                     # Full model, not just Registry

    PRIMARY KEY (RegistrySID)
);

CREATE TRIGGER ModelsTrigger BEFORE DELETE ON Models
FOR EACH ROW
BEGIN
    DELETE FROM ModelEntities WHERE RegistrySID=OLD.RegistrySID $$
END ;

CREATE TABLE ModelEntities (        # Group or Resource (no parentSID=Group)
    SID               VARCHAR(255),       # my System ID
    RegistrySID       VARCHAR(64),
    ParentSID         VARCHAR(64),        # ID of parent ModelEntity
    Abstract          VARCHAR(255),       # /GROUPS, /GROUPS/RESOURCES

    # For Groups and Resources
    Plural              VARCHAR(64),
    Singular            VARCHAR(64),
    Description         VARCHAR(255),
    ModelVersion        VARCHAR(255),
    ModelCompatibleWith VARCHAR(255),
    Labels              JSON,
    XImportResources    VARCHAR($MAX_VARCHAR),
    Attributes          JSON,               # Until we use the Attributes table

    # For Resources
    MaxVersions         INT,
    SetVersionId        BOOL,
    HasDocument         BOOL,
    SingleVersionRoot   BOOL,
    TypeMap             JSON,
    XImportOrigin       VARCHAR(255),
    MetaAttributes      JSON,

    PRIMARY KEY(SID),
    UNIQUE INDEX (RegistrySID, ParentSID, Plural),
    UNIQUE INDEX (RegistrySID, Abstract),
    CONSTRAINT UC_Singular UNIQUE (RegistrySID, ParentSID, Singular)
);

CREATE TRIGGER ModelTrigger BEFORE DELETE ON ModelEntities
FOR EACH ROW
BEGIN
    DELETE FROM "Groups"        WHERE ModelSID=OLD.SID $$
    DELETE FROM Resources       WHERE ModelSID=OLD.SID $$
END ;

CREATE TABLE "Groups" (
    SID             VARCHAR(64) NOT NULL,   # System ID
    UID             VARCHAR(64) NOT NULL,   # User defined
    RegistrySID     VARCHAR(64) NOT NULL,
    ModelSID        VARCHAR(64) NOT NULL,
    Path            VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    Abstract        VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    Plural          VARCHAR(64) NOT NULL,
    Singular        VARCHAR(64) NOT NULL,

    PRIMARY KEY (SID),
    INDEX(RegistrySID, UID),   # DUG!! See if we can remove this, it's wrong
    UNIQUE INDEX (RegistrySID, ModelSID, UID)
);

CREATE TRIGGER GroupTrigger BEFORE DELETE ON "Groups"
FOR EACH ROW
BEGIN
    DELETE FROM Resources WHERE GroupSID=OLD.SID $$
    DELETE FROM Props WHERE eSID=OLD.SID $$
    DELETE FROM Entities  WHERE eSID=OLD.SID $$
END ;

CREATE TABLE Resources (
    SID             VARCHAR(64) NOT NULL,   # System ID
    UID             VARCHAR(64) NOT NULL,   # User defined
    RegistrySID     VARCHAR(64) NOT NULL,
    GroupSID        VARCHAR(64) NOT NULL,   # System ID
    ModelSID        VARCHAR(64) NOT NULL,
    Path            VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    Abstract        VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    Plural          VARCHAR(64) NOT NULL,
    Singular        VARCHAR(64) NOT NULL,

    PRIMARY KEY (SID),
    INDEX(GroupSID, UID),   # DUG!!! See if we can remove this, it's wrong
    INDEX(Path),            # DUG! this is wrong, see if we can remove it
    INDEX(RegistrySID),
    INDEX(RegistrySID, Path),
    UNIQUE INDEX (GroupSID, ModelSID, UID)
);

CREATE TRIGGER ResourcesTrigger BEFORE DELETE ON Resources
FOR EACH ROW
BEGIN
    # Swallow error 1442 ("Can't update table 'Registries' in stored
    # function/trigger because it is already used by statement which
    # invoked this stored function/trigger") from the Registries UPDATE
    # below. This fires when this Resource is being deleted as part of
    # a cascading whole-Registry delete (DELETE FROM Registries ->
    # RegistryTrigger -> ... -> this trigger) - MySQL forbids modifying
    # a table that's already being modified higher up the same trigger
    # chain. That's fine here: if the whole Registry is being deleted,
    # its UsesXref value is moot anyway (the row is going away too).
    DECLARE CONTINUE HANDLER FOR 1442 BEGIN END $$

    # Clear the stale xref mirror on every source Meta that points at
    # this Resource's Path, since Metas.xRefPath is a plain path string
    # (not a SID) that ISN'T touched by the eSID/ParentSID=OLD.SID
    # deletes below - a source's synthetic mirror rows live under ITS
    # OWN ParentSID (its own owning Resource's SID), not the target's.
    # This single trigger fires for every deletion path (direct
    # Resource delete, whole-Group delete via GroupTrigger, whole-
    # Registry delete), so it's the one place this needs to be handled
    # - no Go-level call site needs to remember to do it. Each source's
    # own xRefPath is left untouched, so if a new Resource is later
    # created at this same Path, its own creation-time Go-level fan-out
    # (fullSaveXrefFanOutForTargetMeta/Version) naturally re-populates
    # these sources again.
    DELETE ft FROM Props AS ft
    JOIN Metas AS srcM ON (ft.eSID=srcM.SID)
    WHERE srcM.RegistrySID=OLD.RegistrySID AND srcM.xRefPath=OLD.Path
          AND ft.IsXrefPropCopy=true $$

    DELETE ft FROM Props AS ft
    JOIN Metas AS srcM ON (ft.RegSID=srcM.RegistrySID AND
                            ft.ParentSID=srcM.ResourceSID)
    WHERE srcM.RegistrySID=OLD.RegistrySID AND srcM.xRefPath=OLD.Path
          AND ft.IsXrefVerCopy=true $$

    DELETE fe FROM Entities AS fe
    JOIN Metas AS srcM ON (fe.RegSID=srcM.RegistrySID AND
                            fe.ParentSID=srcM.ResourceSID)
    WHERE srcM.RegistrySID=OLD.RegistrySID AND srcM.xRefPath=OLD.Path
          AND fe.IsXrefVerCopy=true $$

    DELETE ft FROM Props AS ft
    JOIN Metas AS srcM ON (ft.eSID=srcM.ResourceSID)
    WHERE srcM.RegistrySID=OLD.RegistrySID AND srcM.xRefPath=OLD.Path
          AND ft.IsDefaultVerCopy=true $$

    DELETE FROM Metas WHERE ResourceSID=OLD.SID $$
    DELETE FROM Versions WHERE ResourceSID=OLD.SID $$
    DELETE FROM Props WHERE eSID=OLD.SID OR ParentSID=OLD.SID $$
    DELETE FROM Entities  WHERE eSID=OLD.SID OR ParentSID=OLD.SID $$

    # Lazily clear Registries.UsesXref if this deletion may have
    # removed the last remaining xref in the Registry (e.g. OLD itself
    # was an xref source). Guarded by "AND UsesXref=true" so this is a
    # cheap single-row PK no-match in the common case where the
    # Registry doesn't use xref at all - safe to run unconditionally on
    # every Resource delete since the EXISTS rescan only actually
    # executes when there's something to potentially clear.
    UPDATE Registries SET UsesXref = EXISTS(
        SELECT 1 FROM Metas WHERE RegistrySID=OLD.RegistrySID
                             AND xRefPath IS NOT NULL)
    WHERE SID=OLD.RegistrySID AND UsesXref=true $$
END ;

CREATE TABLE Metas (
    SID             VARCHAR(64) NOT NULL,   # System ID
    RegistrySID     VARCHAR(64) NOT NULL,
    ResourceSID     VARCHAR(64) NOT NULL,   # System ID
    Path            VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    Abstract        VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    Plural          VARCHAR(64) NOT NULL,
    Singular        VARCHAR(64) NOT NULL,

    xRefPath        VARCHAR(255) COLLATE utf8mb4_bin, # Generated
    defaultVID      VARCHAR(64),           # Generated

    PRIMARY KEY (SID),
    INDEX(ResourceSID),
    INDEX(RegistrySID, Path),
    INDEX(RegistrySID),
    INDEX(xRefPath),
    # Speeds up the "who currently xrefs me" fan-out query
    # (fullSaveXrefFanOutForTarget: SELECT ResourceSID FROM Metas WHERE
    # RegistrySID=? AND xRefPath=?) with a single composite lookup
    # instead of relying on the single-column xRefPath index above.
    INDEX(RegistrySID, xRefPath)
);

# Can't use this because we get recursive triggers on meta.delete()
# CREATE TRIGGER MetasTrigger BEFORE DELETE ON Metas
# FOR EACH ROW
# BEGIN
    # DELETE FROM Props WHERE EntitySID=OLD.SID $$
# END ;

CREATE TABLE Versions (
    SID                 VARCHAR(64) NOT NULL,   # System ID
    UID                 VARCHAR(64) NOT NULL,   # User defined
    RegistrySID         VARCHAR(64) NOT NULL,
    ResourceSID         VARCHAR(64) NOT NULL,   # System ID
    Path                VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    Abstract            VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,

    AncestorID          VARCHAR(65) NOT NULL COLLATE utf8mb4_bin,  # Generated
    CreatedAt           VARCHAR(255),           # Generated (for ancestor stuff

    PRIMARY KEY (SID),
    UNIQUE INDEX (ResourceSID, UID),
    INDEX (ResourceSID, AncestorID)
);

CREATE TRIGGER VersionsTrigger BEFORE DELETE ON Versions
FOR EACH ROW
BEGIN
    DELETE FROM ResourceContents WHERE VersionSID=OLD.SID $$
    DELETE FROM Props WHERE eSID=OLD.SID $$
    DELETE FROM Entities  WHERE eSID=OLD.SID $$
END ;

CREATE TABLE ResourceContents (
    VersionSID      VARCHAR(255),
    Content         MEDIUMBLOB,

    PRIMARY KEY (VersionSID)
);

CREATE TABLE Props (
  RegSID     VARCHAR(64) NOT NULL,
  Type       BIGINT NOT NUll,
  Plural     VARCHAR(64) NOT NULL,
  Singular   VARCHAR(64) NOT NULL,
  ParentSID  VARCHAR(64) NULL,
  eSID       VARCHAR(64) NOT NULL,      # Reg,Group,Res,Ver System ID
  UID        VARCHAR(255) NOT NULL,      # User Defined
  Path       VARCHAR(329) NOT NULL COLLATE utf8mb4_bin,
  LowerPath  VARCHAR(329) GENERATED ALWAYS AS (LOWER(Path)) STORED,
  PropName   VARCHAR($MAX_PROPNAME) NOT NULL,
  PropValue  MEDIUMTEXT NULL, # VARCHAR($MAX_VARCHAR),
  PropType   CHAR(64) NOT NULL,          # string, boolean, int, ...
  Abstract   VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
  DocView    BOOL NOT NULL,

  # IMPORTANT: PropName always includes a trailing delimiter (DB_IN, which
  # is ','). For example: "fileurl," not "fileurl"
  # This is used throughout the codebase to simplify attribute name parsing and
  # prevent ambiguity (e.g., distinguishing "file" from "fileurl").
  # When querying Props by name, always append DB_IN to the attribute name:
  #   WHERE PropName = 'fileurl' || string(DB_IN)  -- CORRECT
  #   WHERE PropName = 'fileurl'                   -- WRONG (will never match)

  # non-doc-view-able attributes are ones that are generated at runtime
  # due to things like showing the Default Version props in the Resource
  # or entities/props that materialize due to an xref. Normally a GET
  # will show all props, but during /export or ?doc we want to exclude
  # these non-doc-view ones. In case where all of the props for an entity
  # are generated, the entire entity should vanish from the serialization.
  # e.g. Versions of an xref'd Resource.


  # These flag WHY this row exists when it's not a direct write of the
  # entity's own user-set value (multiple booleans, one per reason, so
  # each can be independently identified/refreshed without disturbing
  # the others). All false means this row is a direct, currently-live
  # user-set property (written by SetDBProperty()).
  IsDefaultVerCopy BOOL NOT NULL DEFAULT false, # Copied from default Version
  IsXrefPropCopy   BOOL NOT NULL DEFAULT false, # meta.* copied from xref target
  IsXrefVerCopy    BOOL NOT NULL DEFAULT false, # Synthetic Version copied via xref
  IsSystemProp     BOOL NOT NULL DEFAULT false, # Set via SetSystemDBProperty()

  # Calculated singleton attrs are split into two flags rather than one,
  # since they behave very differently: "static" ones (xid, a Version's
  # RESOURCEid, e.g. "fileid") are provably immutable after entity
  # creation (no rename API, a Version's owning Resource never
  # changes) so they're written ONCE by EntityInsert()/
  # SaveCalcStaticInsert() and never touched again. "dynamic" ones (a
  # Version's own isdefault) genuinely change over time (as the
  # Resource's default version pointer moves) so they're recomputed on
  # every relevant Save() by SaveVersionCalc(). A Resource's own
  # "isdefault" isn't a calculated singleton at all - it's simply
  # mirrored in from its default Version (same IsDefaultVerCopy
  # mechanism as createdat/modifiedat) by SaveDefaultVersionCascade(),
  # so it's naturally absent whenever there's no default Version to
  # copy from (e.g. a dangling xref).
  IsCalcStatic     BOOL NOT NULL DEFAULT false, # xid, Version.RESOURCEid
  IsCalcDynamic    BOOL NOT NULL DEFAULT false, # Version.isdefault

  PRIMARY KEY(RegSID, Path, PropName),
  UNIQUE INDEX(eSID, PropName),
  INDEX(ParentSID) # for cascade-copy cleanup (e.g.
                   # fullSaveXrefCascadeDelete's ParentSID scan)
  # INDEX(RegSID, Abstract)
);

# This is the authoritative store for all entity properties (own,
# system, calculated, and cascaded/copied) - see fulltree.go.
CREATE TABLE Entities (
  RegSID     VARCHAR(64) NOT NULL,
  Type       BIGINT NOT NULL,
  Plural     VARCHAR(64) NOT NULL,
  Singular   VARCHAR(64) NOT NULL,
  ParentSID  VARCHAR(64) NULL,
  eSID       VARCHAR(64) NOT NULL,      # Reg,Group,Res,Ver System ID (or
                                          # synthetic "-<srcRSID>-<verSID>")
  UID        VARCHAR(255) NOT NULL,
  Abstract   VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
  Path       VARCHAR(329) NOT NULL COLLATE utf8mb4_bin,
  LowerPath  VARCHAR(329) GENERATED ALWAYS AS (LOWER(Path)) STORED,

  # True for the synthetic Version rows added because a Resource
  # xref's another Resource (mirrors the "-" eSID prefix convention
  # this table already uses, but as an explicit flag for clarity).
  IsXrefVerCopy BOOL NOT NULL DEFAULT false,

  # eSID is already globally unique (NewUUID()), so it alone can be the
  # PK - RegSID would be redundant there. RegSID is kept as its own
  # index so RegistryTrigger's bulk "DELETE FROM Entities WHERE
  # RegSID=..." stays fast. ParentSID is also a real (globally unique)
  # SID, so it needs no RegSID qualifier either. Path is NOT globally
  # unique (only unique per-Registry), so RegSID must stay paired with
  # it.
  PRIMARY KEY(eSID),
  INDEX(RegSID),
  INDEX(ParentSID),
  UNIQUE INDEX (RegSID, Path),
  UNIQUE INDEX (RegSID, LowerPath)
);

# These maintain Versions.AncestorID/CreatedAt and Metas.xRefPath/
# defaultVID whenever the corresponding OWN (non-cascaded, non-
# calculated) property row is written/removed on Props, which
# is the sole authoritative store for entity properties now (see
# fulltree.go) - those DB columns are relied on throughout the codebase
# (ancestor-chain resolution, xref detection, default-version lookups)
# and are otherwise never set anywhere else. xRefPath stores the raw
# xref target path text (not a resolved SID) so it never needs to be
# re-resolved/self-healed: it stays correct even if the target
# Resource doesn't exist yet (or existed, was deleted, and later gets
# recreated) - every consumer joins against Resources.Path live, at
# query time, instead of relying on a point-in-time-resolved SID.
CREATE TRIGGER FullTreeAncestor BEFORE INSERT ON Props
FOR EACH ROW
BEGIN
    IF (NEW.Type=$ENTITY_VERSION AND NEW.IsDefaultVerCopy=false AND
        NEW.IsXrefPropCopy=false AND NEW.IsXrefVerCopy=false) THEN
        IF (NEW.PropName='ancestorid$DB_IN') THEN
          UPDATE Versions SET AncestorID=NEW.PropValue
              WHERE SID=NEW.eSID $$
        END IF $$
        IF (NEW.PropName='createdat$DB_IN') THEN
          UPDATE Versions SET CreatedAt=NEW.PropValue
              WHERE SID=NEW.eSID $$
        END IF $$
    END IF $$

    IF (NEW.Type=$ENTITY_META AND NEW.IsDefaultVerCopy=false AND
        NEW.IsXrefPropCopy=false AND NEW.IsXrefVerCopy=false) THEN
        IF (NEW.PropName='xref$DB_IN') THEN
          # Remove leading / - store the path text as-is, no lookup.
          UPDATE Metas AS m SET xRefPath=SUBSTRING(NEW.PropValue,2)
            WHERE m.SID=NEW.eSID $$
        END IF $$
        IF (NEW.PropName='defaultversionid$DB_IN') THEN
          UPDATE Metas AS m SET defaultVID=NEW.PropValue
            WHERE m.SID=NEW.eSID $$
        END IF $$
    END IF $$
END ;

CREATE TRIGGER FullTreeXref BEFORE DELETE ON Props
FOR EACH ROW
BEGIN
    # See ResourcesTrigger's comment: swallow error 1442 from the
    # Registries UPDATE below when this fires as part of a cascading
    # whole-Registry delete (Registries is already in use higher up
    # that same trigger chain) - harmless since the Registry row itself
    # is being removed too in that case.
    DECLARE CONTINUE HANDLER FOR 1442 BEGIN END $$

    IF (OLD.Type=$ENTITY_META AND OLD.IsDefaultVerCopy=false AND
        OLD.IsXrefPropCopy=false AND OLD.IsXrefVerCopy=false) THEN
        IF (OLD.PropName='xref$DB_IN') THEN
          UPDATE Metas SET xRefPath=NULL
          WHERE SID=OLD.eSID $$

          # Lazily clear Registries.UsesXref if this was the last
          # remaining xref in the Registry - see ResourcesTrigger's
          # comment for why this is safe/cheap to run unconditionally.
          UPDATE Registries SET UsesXref = EXISTS(
              SELECT 1 FROM Metas WHERE RegistrySID=OLD.RegSID
                                   AND xRefPath IS NOT NULL)
          WHERE SID=OLD.RegSID AND UsesXref=true $$
        END IF $$
        IF (OLD.PropName='defaultversionid$DB_IN') THEN
          UPDATE Metas AS m SET defaultVID=NULL
            WHERE m.SID=OLD.eSID $$
        END IF $$
    END IF $$
END ;

CREATE VIEW Leaves AS
SELECT eSID FROM Entities
WHERE eSID NOT IN (
    SELECT DISTINCT ParentSID FROM Entities WHERE ParentSID IS NOT NULL
);

# Find all of the versions of a resource. Users of this should order
# the results: ORDER BY Pos ASC, Time ASC, VersionUID ASC
# to get oldest first, newest last.
# Pos (postion) makes sure roots are first, leaves are last.
# For similar rows, order by createdat timestamps and then versionIDs
CREATE VIEW VersionAncestors AS
SELECT
    v.RegistrySID AS RegistrySID,
    v.ResourceSID AS ResourceSID,
    v.SID AS VersionSID,
    v.UID AS VersionUID,
    v.AncestorID AS AncestorID,
    v.CreatedAt AS CTime,
    CASE
        WHEN v.UID=v.AncestorID THEN '0-root'
        WHEN EXISTS(SELECT 1 FROM Versions AS v2 WHERE
                    # v2.RegistrySID=v2.ResistrySID AND
                    v2.ResourceSID=v.ResourceSID AND v2.AncestorID=v.UID)
             THEN '1-middle'
        ELSE '2-leaf'
    END AS Pos
FROM Versions AS v ;

# Find all Versions that are part of circular references (circles)
# Would this be better to do in code and use args(?) for regSID?
CREATE VIEW VersionCircles AS
WITH RECURSIVE cte (RegistrySID,ResourceSID,UID) AS
(
    # Start with the roots and leaves, they can never be part of a circle
    SELECT v.RegistrySID,v.ResourceSID,v.UID FROM Versions AS v
    WHERE v.AncestorID=UID OR
        NOT EXISTS(SELECT 1 FROM Versions AS v2 WHERE
                   v2.RegistrySID=v.RegistrySID AND
                   v2.ResourceSID=v.ResourceSID AND
                   v2.AncestorID=v.UID)
    UNION
    # Now find all Versions whose AncestorID is in cte
    SELECT v3.RegistrySID,v3.ResourceSID,v3.UID FROM Versions AS v3
    INNER JOIN cte ON (
        v3.RegistrySID=cte.RegistrySID AND
        v3.ResourceSID=cte.ResourceSID AND
        v3.AncestorID=cte.UID )
)
# And finally, return all Version UID that are NOT in cte (these are circular)
SELECT v.RegistrySID, v.ResourceSID, v.UID FROM Versions AS v
WHERE NOT EXISTS(SELECT 1 FROM cte
                 WHERE cte.RegistrySID=v.RegistrySID AND
                       cte.ResourceSID=v.ResourceSID AND
                       cte.UID=v.UID);

# Just for debugging purposes
CREATE VIEW VerboseProps AS
SELECT
    p.RegSID,
    p.eSID,
    e.Abstract,
    e.Path,
    p.PropName,
    p.PropValue,
    p.PropType
FROM Props as p
JOIN Entities as e ON (e.eSID=p.eSID)
ORDER by Path ;

CREATE VIEW NewVAs AS
SELECT
    r.UID AS rUID,
    v.UID AS VersionUID,
    v.AncestorID AS AncestorID,
    v.CreatedAt AS CTime,
    CASE
        WHEN v.UID=v.AncestorID THEN '0-root'
        WHEN EXISTS(SELECT 1 FROM Versions AS v2 WHERE
                    # v2.RegistrySID=v2.ResistrySID AND
                    v2.ResourceSID=v.ResourceSID AND v2.AncestorID=v.UID)
             THEN '1-middle'
        ELSE '2-leaf'
    END AS Pos
FROM Versions AS v
JOIN Resources as r on (r.SID=v.ResourceSID) ;
